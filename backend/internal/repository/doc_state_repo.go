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

// DocStateRepo manages compacted document state persistence.
type DocStateRepo interface {
	// GetOrInit gets the document state, inserting an empty Y.Doc when missing.
	GetOrInit(ctx context.Context, nodeID uuid.UUID) (*models.DocState, error)
	// GetForUpdate gets the document state row inside tx with a row lock.
	GetForUpdate(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID) (*models.DocState, error)
	// UpdateAfterCompact stores compacted document state inside tx.
	UpdateAfterCompact(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID, ydocState []byte, markdownPlain string) error
}

type docStateRepo struct {
	pool *pgxpool.Pool
}

func NewDocStateRepo(pool *pgxpool.Pool) DocStateRepo {
	return &docStateRepo{pool: pool}
}

func (r *docStateRepo) GetOrInit(ctx context.Context, nodeID uuid.UUID) (*models.DocState, error) {
	emptyYDoc := []byte{0, 0}

	const insertQuery = `
INSERT INTO doc_states (node_id, ydoc_state)
VALUES ($1, $2)
ON CONFLICT (node_id) DO NOTHING
`
	if _, err := r.pool.Exec(ctx, insertQuery, nodeID, emptyYDoc); err != nil {
		return nil, fmt.Errorf("failed to init doc state: %w", err)
	}

	const selectQuery = `
SELECT node_id, ydoc_state, version, markdown_plain, last_compacted_at
FROM doc_states
WHERE node_id = $1
`

	ds := &models.DocState{}
	if err := r.pool.QueryRow(ctx, selectQuery, nodeID).Scan(
		&ds.NodeID,
		&ds.YDocState,
		&ds.Version,
		&ds.MarkdownPlain,
		&ds.LastCompactedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to get doc state: %w", err)
	}

	return ds, nil
}

func (r *docStateRepo) GetForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
) (*models.DocState, error) {
	const query = `
SELECT node_id, ydoc_state, version, markdown_plain, last_compacted_at
FROM doc_states
WHERE node_id = $1
FOR UPDATE
`

	ds := &models.DocState{}
	if err := tx.QueryRow(ctx, query, nodeID).Scan(
		&ds.NodeID,
		&ds.YDocState,
		&ds.Version,
		&ds.MarkdownPlain,
		&ds.LastCompactedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get doc state for update: %w", err)
	}

	return ds, nil
}

func (r *docStateRepo) UpdateAfterCompact(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	ydocState []byte,
	markdownPlain string,
) error {
	const query = `
UPDATE doc_states
SET ydoc_state = $2,
    markdown_plain = $3,
    version = version + 1,
    last_compacted_at = now()
WHERE node_id = $1
`

	tag, err := tx.Exec(ctx, query, nodeID, ydocState, markdownPlain)
	if err != nil {
		return fmt.Errorf("failed to update doc state after compact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to update doc state after compact: %w", pgx.ErrNoRows)
	}

	return nil
}
