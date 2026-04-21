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

type RefreshTokenRepo interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByTokenHash(ctx context.Context, hash []byte) (*models.RefreshToken, error)
	MarkReplaced(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error
	RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error
}

type refreshTokenRepo struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepo(pool *pgxpool.Pool) RefreshTokenRepo {
	return &refreshTokenRepo{pool: pool}
}

func (r *refreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	if token == nil {
		return fmt.Errorf("failed to create refresh token: %w", errors.New("token is nil"))
	}

	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}

	const query = `
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, replaced_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING created_at
`

	if err := r.pool.QueryRow(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.RevokedAt,
		token.ReplacedBy,
	).Scan(&token.CreatedAt); err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

func (r *refreshTokenRepo) FindByTokenHash(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
	const query = `
SELECT id, user_id, token_hash, expires_at, revoked_at, replaced_by, created_at
FROM refresh_tokens
WHERE token_hash = $1
`

	token := &models.RefreshToken{}
	if err := r.pool.QueryRow(ctx, query, hash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.ReplacedBy,
		&token.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find refresh token by hash: %w", err)
	}

	return token, nil
}

func (r *refreshTokenRepo) MarkReplaced(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error {
	const query = `
UPDATE refresh_tokens
SET replaced_by = $2
WHERE id = $1
`

	tag, err := r.pool.Exec(ctx, query, id, replacedBy)
	if err != nil {
		return fmt.Errorf("failed to mark refresh token replaced: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to mark refresh token replaced: %w", pgx.ErrNoRows)
	}

	return nil
}

func (r *refreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	const query = `
UPDATE refresh_tokens
SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL
`

	if _, err := r.pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("failed to revoke refresh tokens by user id: %w", err)
	}

	return nil
}
