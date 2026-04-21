//go:build integration

package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"fpgwiki/backend/internal/models"
)

func TestRefreshTokenRepoCreateAndFindByTokenHash(t *testing.T) {
	pool := newIntegrationPool(t)
	userRepo := NewUserRepo(pool)
	repo := NewRefreshTokenRepo(pool)

	user := createIntegrationUser(t, userRepo)
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	hash := sha256.Sum256([]byte("token-1"))
	token := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash[:],
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
	}

	if err := repo.Create(context.Background(), token); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), hash[:])
	if err != nil {
		t.Fatalf("find by token hash: %v", err)
	}
	if got == nil {
		t.Fatal("expected refresh token, got nil")
	}
	if got.ID != token.ID {
		t.Fatalf("expected id %s, got %s", token.ID, got.ID)
	}
	if !bytes.Equal(got.TokenHash, hash[:]) {
		t.Fatal("expected token_hash to match")
	}
}

func TestRefreshTokenRepoRevokeAllByUserID(t *testing.T) {
	pool := newIntegrationPool(t)
	userRepo := NewUserRepo(pool)
	repo := NewRefreshTokenRepo(pool)

	user := createIntegrationUser(t, userRepo)
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	hash1 := sha256.Sum256([]byte("token-revoke-1"))
	hash2 := sha256.Sum256([]byte("token-revoke-2"))
	token1 := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash1[:],
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
	}
	token2 := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash2[:],
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
	}

	if err := repo.Create(context.Background(), token1); err != nil {
		t.Fatalf("create token1: %v", err)
	}
	if err := repo.Create(context.Background(), token2); err != nil {
		t.Fatalf("create token2: %v", err)
	}

	if err := repo.RevokeAllByUserID(context.Background(), user.ID); err != nil {
		t.Fatalf("revoke all by user id: %v", err)
	}

	got1, err := repo.FindByTokenHash(context.Background(), hash1[:])
	if err != nil {
		t.Fatalf("find token1: %v", err)
	}
	got2, err := repo.FindByTokenHash(context.Background(), hash2[:])
	if err != nil {
		t.Fatalf("find token2: %v", err)
	}

	if got1 == nil || got1.RevokedAt == nil {
		t.Fatal("expected token1 revoked_at to be set")
	}
	if got2 == nil || got2.RevokedAt == nil {
		t.Fatal("expected token2 revoked_at to be set")
	}
}

func TestRefreshTokenRepoMarkReplaced(t *testing.T) {
	pool := newIntegrationPool(t)
	userRepo := NewUserRepo(pool)
	repo := NewRefreshTokenRepo(pool)

	user := createIntegrationUser(t, userRepo)
	t.Cleanup(func() { cleanupUser(t, pool, user.ID) })

	oldHash := sha256.Sum256([]byte("token-old"))
	newHash := sha256.Sum256([]byte("token-new"))
	oldToken := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: oldHash[:],
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
	}
	newToken := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: newHash[:],
		ExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
	}

	if err := repo.Create(context.Background(), oldToken); err != nil {
		t.Fatalf("create old token: %v", err)
	}
	if err := repo.Create(context.Background(), newToken); err != nil {
		t.Fatalf("create new token: %v", err)
	}

	if err := repo.MarkReplaced(context.Background(), oldToken.ID, newToken.ID); err != nil {
		t.Fatalf("mark replaced: %v", err)
	}

	got, err := repo.FindByTokenHash(context.Background(), oldHash[:])
	if err != nil {
		t.Fatalf("find old token: %v", err)
	}
	if got == nil {
		t.Fatal("expected old token, got nil")
	}
	if got.ReplacedBy == nil {
		t.Fatal("expected replaced_by to be set")
	}
	if *got.ReplacedBy != newToken.ID {
		t.Fatalf("expected replaced_by %s, got %s", newToken.ID, *got.ReplacedBy)
	}
}

func createIntegrationUser(t *testing.T, repo UserRepo) *models.User {
	t.Helper()

	user := &models.User{
		Email:        testEmail(),
		PasswordHash: "hash-integration",
		DisplayName:  "Integration User",
		Role:         "member",
		Locale:       "zh",
	}

	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("create integration user: %v", err)
	}

	return user
}
