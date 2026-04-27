package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
)

// DocUpdateRepo manages pending Y.js update persistence.
type DocUpdateRepo interface {
	// Append stores one Y.js update for a document.
	Append(ctx context.Context, nodeID uuid.UUID, update []byte, clientID string) error
	// ListByNode lists all pending updates for a document in append order.
	ListByNode(ctx context.Context, nodeID uuid.UUID) ([]models.DocUpdate, error)
	// CountByNode counts pending updates for a document.
	CountByNode(ctx context.Context, nodeID uuid.UUID) (int64, error)
	// DeleteUpTo deletes updates for nodeID with id less than or equal to maxID inside tx.
	DeleteUpTo(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID, maxID int64) error
	// FindCompactCandidates finds documents that should be compacted.
	FindCompactCandidates(ctx context.Context, countThreshold int64, staleInterval string) ([]uuid.UUID, error)
}

type docUpdateRepo struct {
	pool *pgxpool.Pool
}

func NewDocUpdateRepo(pool *pgxpool.Pool) DocUpdateRepo {
	return &docUpdateRepo{pool: pool}
}

func (r *docUpdateRepo) Append(ctx context.Context, nodeID uuid.UUID, update []byte, clientID string) error {
	const query = `
INSERT INTO doc_updates (node_id, update, client_id)
VALUES ($1, $2, $3)
`

	if _, err := r.pool.Exec(ctx, query, nodeID, update, clientID); err != nil {
		return fmt.Errorf("failed to append doc update: %w", err)
	}

	return nil
}

func (r *docUpdateRepo) ListByNode(ctx context.Context, nodeID uuid.UUID) ([]models.DocUpdate, error) {
	const query = `
SELECT id, node_id, update, client_id, created_at
FROM doc_updates
WHERE node_id = $1
ORDER BY id ASC
`

	rows, err := r.pool.Query(ctx, query, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list doc updates: %w", err)
	}
	defer rows.Close()

	updates := make([]models.DocUpdate, 0)
	for rows.Next() {
		var update models.DocUpdate
		if err := rows.Scan(
			&update.ID,
			&update.NodeID,
			&update.Update,
			&update.ClientID,
			&update.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to list doc updates: %w", err)
		}
		updates = append(updates, update)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list doc updates: %w", err)
	}

	return updates, nil
}

func (r *docUpdateRepo) CountByNode(ctx context.Context, nodeID uuid.UUID) (int64, error) {
	const query = `
SELECT count(*)
FROM doc_updates
WHERE node_id = $1
`

	var count int64
	if err := r.pool.QueryRow(ctx, query, nodeID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count doc updates: %w", err)
	}

	return count, nil
}

func (r *docUpdateRepo) DeleteUpTo(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID, maxID int64) error {
	const query = `
DELETE FROM doc_updates
WHERE node_id = $1 AND id <= $2
`

	if _, err := tx.Exec(ctx, query, nodeID, maxID); err != nil {
		return fmt.Errorf("failed to delete doc updates up to id %d: %w", maxID, err)
	}

	return nil
}

func (r *docUpdateRepo) FindCompactCandidates(
	ctx context.Context,
	countThreshold int64,
	staleInterval string,
) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
SELECT node_id FROM doc_updates
GROUP BY node_id
HAVING count(*) >= $1

UNION

SELECT ds.node_id FROM doc_states ds
WHERE ds.last_compacted_at < now() - interval '%s'
  AND EXISTS (SELECT 1 FROM doc_updates du WHERE du.node_id = ds.node_id)
`, staleInterval)

	rows, err := r.pool.Query(ctx, query, countThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to find compact candidates: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to find compact candidates: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to find compact candidates: %w", err)
	}

	return ids, nil
}
