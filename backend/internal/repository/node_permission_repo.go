package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
)

type NodePermissionRepo interface {
	ReplacePermissions(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID, perms []models.NodePermission) error
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]models.NodePermission, error)
	DeleteByGroupID(ctx context.Context, tx pgx.Tx, groupID uuid.UUID) error
	HasAnyByNode(ctx context.Context, nodeID uuid.UUID) (bool, error)
}

type nodePermissionRepo struct {
	pool *pgxpool.Pool
}

func NewNodePermissionRepo(pool *pgxpool.Pool) NodePermissionRepo {
	return &nodePermissionRepo{pool: pool}
}

func (r *nodePermissionRepo) ReplacePermissions(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	perms []models.NodePermission,
) error {
	const deleteQuery = `
DELETE FROM node_permissions
WHERE node_id = $1
`

	if _, err := tx.Exec(ctx, deleteQuery, nodeID); err != nil {
		return fmt.Errorf("failed to replace node permissions: %w", err)
	}

	if len(perms) == 0 {
		return nil
	}

	const insertQuery = `
INSERT INTO node_permissions (id, node_id, subject_type, subject_id, level)
VALUES ($1, $2, $3, $4, $5)
`

	batch := &pgx.Batch{}
	for i := range perms {
		id := perms[i].ID
		if id == uuid.Nil {
			id = uuid.New()
		}

		batch.Queue(
			insertQuery,
			id,
			nodeID,
			perms[i].SubjectType,
			perms[i].SubjectID,
			perms[i].Level,
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	for range perms {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to replace node permissions: %w", err)
		}
	}

	return nil
}

func (r *nodePermissionRepo) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]models.NodePermission, error) {
	const query = `
SELECT id, node_id, subject_type, subject_id, level, created_at, updated_at
FROM node_permissions
WHERE node_id = $1
ORDER BY subject_type, created_at
`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list node permissions: %w", err)
	}
	defer rows.Close()

	perms := make([]models.NodePermission, 0)
	for rows.Next() {
		var perm models.NodePermission
		if err := rows.Scan(
			&perm.ID,
			&perm.NodeID,
			&perm.SubjectType,
			&perm.SubjectID,
			&perm.Level,
			&perm.CreatedAt,
			&perm.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to list node permissions: %w", err)
		}
		perms = append(perms, perm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list node permissions: %w", err)
	}

	return perms, nil
}

func (r *nodePermissionRepo) DeleteByGroupID(ctx context.Context, tx pgx.Tx, groupID uuid.UUID) error {
	const query = `
DELETE FROM node_permissions
WHERE subject_type = 'group' AND subject_id = $1
`

	if _, err := tx.Exec(ctx, query, groupID); err != nil {
		return fmt.Errorf("failed to delete node permissions: %w", err)
	}

	return nil
}

func (r *nodePermissionRepo) HasAnyByNode(ctx context.Context, nodeID uuid.UUID) (bool, error) {
	const query = `
SELECT EXISTS(
  SELECT 1
  FROM node_permissions
  WHERE node_id = $1
)
`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, nodeID).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check node permissions: %w", err)
	}

	return exists, nil
}
