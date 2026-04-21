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

type UserRepo interface {
	CreateUser(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error
	IncrementTokenVersion(ctx context.Context, id uuid.UUID) error
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) UserRepo {
	return &userRepo{pool: pool}
}

func (r *userRepo) CreateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("failed to create user: %w", errors.New("user is nil"))
	}

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.Role == "" {
		user.Role = "member"
	}
	if user.Locale == "" {
		user.Locale = "zh"
	}

	const query = `
INSERT INTO users (id, email, password_hash, display_name, role, token_version, locale)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING created_at, updated_at
`

	if err := r.pool.QueryRow(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.DisplayName,
		user.Role,
		user.TokenVersion,
		user.Locale,
	).Scan(&user.CreatedAt, &user.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	const query = `
SELECT id, email, password_hash, display_name, role, token_version, locale, created_at, updated_at
FROM users
WHERE email = $1
`

	user := &models.User{}
	if err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.TokenVersion,
		&user.Locale,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return user, nil
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	const query = `
SELECT id, email, password_hash, display_name, role, token_version, locale, created_at, updated_at
FROM users
WHERE id = $1
`

	user := &models.User{}
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.DisplayName,
		&user.Role,
		&user.TokenVersion,
		&user.Locale,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return user, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return fmt.Errorf("failed to update user: %w", errors.New("user is nil"))
	}

	const query = `
UPDATE users
SET display_name = $2, locale = $3, updated_at = now()
WHERE id = $1
RETURNING updated_at
`

	if err := r.pool.QueryRow(ctx, query, user.ID, user.DisplayName, user.Locale).Scan(&user.UpdatedAt); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *userRepo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	const query = `
UPDATE users
SET password_hash = $2, token_version = token_version + 1, updated_at = now()
WHERE id = $1
`

	tag, err := r.pool.Exec(ctx, query, id, hash)
	if err != nil {
		return fmt.Errorf("failed to update password hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to update password hash: %w", pgx.ErrNoRows)
	}

	return nil
}

func (r *userRepo) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
	const query = `
UPDATE users
SET token_version = token_version + 1, updated_at = now()
WHERE id = $1
`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment token version: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to increment token version: %w", pgx.ErrNoRows)
	}

	return nil
}
