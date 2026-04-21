//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
)

const defaultIntegrationDSN = "postgres://fpgwiki:fpgwiki_dev@localhost:5432/fpgwiki?sslmode=disable"

func TestUserRepoCreateAndFindByEmail(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	user := &models.User{
		Email:        testEmail(),
		PasswordHash: "hash-1",
		DisplayName:  "User One",
		Role:         "member",
		Locale:       "zh",
	}

	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	got, err := repo.FindByEmail(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != user.ID {
		t.Fatalf("expected id %s, got %s", user.ID, got.ID)
	}
}

func TestUserRepoCreateUserDuplicateEmail(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	email := testEmail()
	user1 := &models.User{
		Email:        email,
		PasswordHash: "hash-1",
		DisplayName:  "User One",
		Role:         "member",
		Locale:       "zh",
	}
	if err := repo.CreateUser(context.Background(), user1); err != nil {
		t.Fatalf("create first user: %v", err)
	}
	t.Cleanup(func() { cleanupUser(t, pool, user1.ID) })

	user2 := &models.User{
		Email:        email,
		PasswordHash: "hash-2",
		DisplayName:  "User Two",
		Role:         "member",
		Locale:       "en",
	}
	if err := repo.CreateUser(context.Background(), user2); err == nil {
		t.Fatal("expected duplicate email error, got nil")
	}
}

func TestUserRepoFindByEmailNotFound(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	got, err := repo.FindByEmail(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil user, got %+v", got)
	}
}

func TestUserRepoFindByID(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	user := &models.User{
		Email:        testEmail(),
		PasswordHash: "hash-1",
		DisplayName:  "User One",
		Role:         "member",
		Locale:       "zh",
	}
	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	got, err := repo.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.ID != user.ID {
		t.Fatalf("expected id %s, got %s", user.ID, got.ID)
	}
}

func TestUserRepoFindByIDNotFound(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	got, err := repo.FindByID(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil user, got %+v", got)
	}
}

func TestUserRepoUpdateUser(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	user := &models.User{
		Email:        testEmail(),
		PasswordHash: "hash-1",
		DisplayName:  "Before Update",
		Role:         "member",
		Locale:       "zh",
	}
	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	user.DisplayName = "After Update"
	user.Locale = "en"
	if err := repo.UpdateUser(context.Background(), user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	got, err := repo.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.DisplayName != "After Update" {
		t.Fatalf("expected display_name %q, got %q", "After Update", got.DisplayName)
	}
	if got.Locale != "en" {
		t.Fatalf("expected locale %q, got %q", "en", got.Locale)
	}
}

func TestUserRepoUpdatePasswordHashIncrementsTokenVersion(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	user := &models.User{
		Email:        testEmail(),
		PasswordHash: "hash-1",
		DisplayName:  "Password User",
		Role:         "member",
		Locale:       "zh",
	}
	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	if err := repo.UpdatePasswordHash(context.Background(), user.ID, "hash-2"); err != nil {
		t.Fatalf("update password hash: %v", err)
	}

	got, err := repo.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.PasswordHash != "hash-2" {
		t.Fatalf("expected password_hash %q, got %q", "hash-2", got.PasswordHash)
	}
	if got.TokenVersion != 1 {
		t.Fatalf("expected token_version 1, got %d", got.TokenVersion)
	}
}

func TestUserRepoIncrementTokenVersion(t *testing.T) {
	pool := newIntegrationPool(t)
	repo := NewUserRepo(pool)

	user := &models.User{
		Email:        testEmail(),
		PasswordHash: "hash-1",
		DisplayName:  "Version User",
		Role:         "member",
		Locale:       "zh",
	}
	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	if err := repo.IncrementTokenVersion(context.Background(), user.ID); err != nil {
		t.Fatalf("increment token version: %v", err)
	}

	got, err := repo.FindByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got == nil {
		t.Fatal("expected user, got nil")
	}
	if got.TokenVersion != 1 {
		t.Fatalf("expected token_version 1, got %d", got.TokenVersion)
	}
}

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = defaultIntegrationDSN
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip integration tests: connect postgres failed: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skip integration tests: postgres unavailable: %v", err)
	}

	var usersTable *string
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.users')::text").Scan(&usersTable); err != nil {
		t.Skipf("skip integration tests: schema check failed: %v", err)
	}
	if usersTable == nil {
		t.Skip("skip integration tests: users table not found, run migrations first")
	}

	return pool
}

func cleanupUser(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := pool.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID); err != nil {
		t.Fatalf("cleanup refresh_tokens: %v", err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID); err != nil {
		t.Fatalf("cleanup users: %v", err)
	}
}

func testEmail() string {
	return fmt.Sprintf("user_%d_%s@example.com", time.Now().UnixNano(), uuid.NewString())
}
