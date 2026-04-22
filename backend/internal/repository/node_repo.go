package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
)

type NodeRepo interface {
	Create(ctx context.Context, node *models.Node) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Node, error)
	ListByParent(ctx context.Context, parentID *uuid.UUID) ([]*models.Node, error)
	Update(ctx context.Context, node *models.Node) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Restore(ctx context.Context, id uuid.UUID) error
	GetAncestorsWithPermissions(ctx context.Context, nodeID uuid.UUID) ([]models.AncestorRow, error)
	GetDescendantIDs(ctx context.Context, nodeID uuid.UUID) ([]uuid.UUID, error)
	HasChildren(ctx context.Context, nodeID uuid.UUID) (bool, error)
}

type nodeRepo struct {
	pool *pgxpool.Pool
}

func NewNodeRepo(pool *pgxpool.Pool) NodeRepo {
	return &nodeRepo{pool: pool}
}

func (r *nodeRepo) Create(ctx context.Context, node *models.Node) error {
	if node == nil {
		return fmt.Errorf("failed to create node: %w", errors.New("node is nil"))
	}
	if node.ID == uuid.Nil {
		node.ID = uuid.New()
	}

	const query = `
INSERT INTO nodes (id, parent_id, kind, title, owner_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING created_at, updated_at
`

	if err := r.pool.QueryRow(
		ctx,
		query,
		node.ID,
		node.ParentID,
		node.Kind,
		node.Title,
		node.OwnerID,
	).Scan(&node.CreatedAt, &node.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	return nil
}

func (r *nodeRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Node, error) {
	const query = `
SELECT id, parent_id, kind, title, owner_id, deleted_at, created_at, updated_at
FROM nodes
WHERE id = $1
`

	node := &models.Node{}
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&node.ID,
		&node.ParentID,
		&node.Kind,
		&node.Title,
		&node.OwnerID,
		&node.DeletedAt,
		&node.CreatedAt,
		&node.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find node by id: %w", err)
	}

	return node, nil
}

func (r *nodeRepo) ListByParent(ctx context.Context, parentID *uuid.UUID) ([]*models.Node, error) {
	const queryRoot = `
SELECT id, parent_id, kind, title, owner_id, deleted_at, created_at, updated_at
FROM nodes
WHERE parent_id IS NULL AND deleted_at IS NULL
ORDER BY kind ASC, title ASC
`
	const queryByParent = `
SELECT id, parent_id, kind, title, owner_id, deleted_at, created_at, updated_at
FROM nodes
WHERE parent_id = $1 AND deleted_at IS NULL
ORDER BY kind ASC, title ASC
`

	var (
		rows pgx.Rows
		err  error
	)
	if parentID == nil {
		rows, err = r.pool.Query(ctx, queryRoot)
	} else {
		rows, err = r.pool.Query(ctx, queryByParent, *parentID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes by parent: %w", err)
	}
	defer rows.Close()

	nodes := make([]*models.Node, 0)
	for rows.Next() {
		node := &models.Node{}
		if err := rows.Scan(
			&node.ID,
			&node.ParentID,
			&node.Kind,
			&node.Title,
			&node.OwnerID,
			&node.DeletedAt,
			&node.CreatedAt,
			&node.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to list nodes by parent: %w", err)
		}
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list nodes by parent: %w", err)
	}

	return nodes, nil
}

func (r *nodeRepo) Update(ctx context.Context, node *models.Node) error {
	if node == nil {
		return fmt.Errorf("failed to update node: %w", errors.New("node is nil"))
	}

	const query = `
UPDATE nodes
SET title = $2, parent_id = $3, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING updated_at
`

	if err := r.pool.QueryRow(ctx, query, node.ID, node.Title, node.ParentID).Scan(&node.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to update node: %w", pgx.ErrNoRows)
		}
		return fmt.Errorf("failed to update node: %w", err)
	}

	return nil
}

func (r *nodeRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	const query = `
WITH RECURSIVE subtree AS (
  SELECT id
  FROM nodes
  WHERE id = $1 AND deleted_at IS NULL
  UNION ALL
  SELECT n.id
  FROM nodes n
  JOIN subtree s ON n.parent_id = s.id
  WHERE n.deleted_at IS NULL
)
UPDATE nodes
SET deleted_at = now()
WHERE id IN (SELECT id FROM subtree)
`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete node: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to soft delete node: %w", pgx.ErrNoRows)
	}

	return nil
}

func (r *nodeRepo) Restore(ctx context.Context, id uuid.UUID) error {
	const query = `
WITH RECURSIVE subtree AS (
  SELECT id
  FROM nodes
  WHERE id = $1 AND deleted_at IS NOT NULL
  UNION ALL
  SELECT n.id
  FROM nodes n
  JOIN subtree s ON n.parent_id = s.id
  WHERE n.deleted_at IS NOT NULL
)
UPDATE nodes
SET deleted_at = NULL, updated_at = now()
WHERE id IN (SELECT id FROM subtree)
`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to restore node: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to restore node: %w", pgx.ErrNoRows)
	}

	return nil
}

func (r *nodeRepo) GetAncestorsWithPermissions(ctx context.Context, nodeID uuid.UUID) ([]models.AncestorRow, error) {
	const query = `
WITH RECURSIVE ancestors AS (
  SELECT id, parent_id, 0 AS depth FROM nodes WHERE id = $1 AND deleted_at IS NULL
  UNION ALL
  SELECT n.id, n.parent_id, a.depth + 1
  FROM nodes n
  JOIN ancestors a ON n.id = a.parent_id
  WHERE n.deleted_at IS NULL
)
SELECT a.id AS node_id, a.depth,
       np.subject_type, np.subject_id, np.level
FROM ancestors a
LEFT JOIN node_permissions np ON np.node_id = a.id
ORDER BY a.depth ASC
`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ancestors with permissions for node: %w", err)
	}
	defer rows.Close()

	result := make([]models.AncestorRow, 0)
	for rows.Next() {
		row := models.AncestorRow{}
		if err := rows.Scan(
			&row.NodeID,
			&row.Depth,
			&row.SubjectType,
			&row.SubjectID,
			&row.Level,
		); err != nil {
			return nil, fmt.Errorf("failed to get ancestors with permissions for node: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get ancestors with permissions for node: %w", err)
	}

	return result, nil
}

func (r *nodeRepo) GetDescendantIDs(ctx context.Context, nodeID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
WITH RECURSIVE subtree AS (
  SELECT id
  FROM nodes
  WHERE parent_id = $1 AND deleted_at IS NULL
  UNION ALL
  SELECT n.id
  FROM nodes n
  JOIN subtree s ON n.parent_id = s.id
  WHERE n.deleted_at IS NULL
)
SELECT id FROM subtree
`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get descendant ids for node: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to get descendant ids for node: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get descendant ids for node: %w", err)
	}

	return ids, nil
}

func (r *nodeRepo) HasChildren(ctx context.Context, nodeID uuid.UUID) (bool, error) {
	const query = `
SELECT EXISTS(
  SELECT 1
  FROM nodes
  WHERE parent_id = $1 AND deleted_at IS NULL
)
`

	var hasChildren bool
	if err := r.pool.QueryRow(ctx, query, nodeID).Scan(&hasChildren); err != nil {
		return false, fmt.Errorf("failed to check node children: %w", err)
	}

	return hasChildren, nil
}
