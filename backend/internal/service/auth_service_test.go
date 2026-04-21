package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/models"
)

type mockUserRepo struct {
	createUserFn             func(ctx context.Context, user *models.User) error
	findByEmailFn            func(ctx context.Context, email string) (*models.User, error)
	findByIDFn               func(ctx context.Context, id uuid.UUID) (*models.User, error)
	updatePasswordHashFn     func(ctx context.Context, id uuid.UUID, hash string) error
	incrementTokenVersionFn  func(ctx context.Context, id uuid.UUID) error
	updateUserFn             func(ctx context.Context, user *models.User) error
	incrementTokenVersionHit int
}

func (m *mockUserRepo) CreateUser(ctx context.Context, user *models.User) error {
	if m.createUserFn == nil {
		return errors.New("unexpected CreateUser call")
	}
	return m.createUserFn(ctx, user)
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.findByEmailFn == nil {
		return nil, errors.New("unexpected FindByEmail call")
	}
	return m.findByEmailFn(ctx, email)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.findByIDFn == nil {
		return nil, errors.New("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id)
}

func (m *mockUserRepo) UpdateUser(ctx context.Context, user *models.User) error {
	if m.updateUserFn == nil {
		return nil
	}
	return m.updateUserFn(ctx, user)
}

func (m *mockUserRepo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	if m.updatePasswordHashFn == nil {
		return errors.New("unexpected UpdatePasswordHash call")
	}
	return m.updatePasswordHashFn(ctx, id, hash)
}

func (m *mockUserRepo) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
	m.incrementTokenVersionHit++
	if m.incrementTokenVersionFn == nil {
		return nil
	}
	return m.incrementTokenVersionFn(ctx, id)
}

type mockRefreshTokenRepo struct {
	createFn             func(ctx context.Context, token *models.RefreshToken) error
	findByTokenHashFn    func(ctx context.Context, hash []byte) (*models.RefreshToken, error)
	markReplacedFn       func(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error
	revokeAllByUserIDFn  func(ctx context.Context, userID uuid.UUID) error
	markReplacedHit      int
	revokeAllByUserIDHit int
	createHit            int
}

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	m.createHit++
	if m.createFn == nil {
		return errors.New("unexpected Create call")
	}
	return m.createFn(ctx, token)
}

func (m *mockRefreshTokenRepo) FindByTokenHash(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
	if m.findByTokenHashFn == nil {
		return nil, errors.New("unexpected FindByTokenHash call")
	}
	return m.findByTokenHashFn(ctx, hash)
}

func (m *mockRefreshTokenRepo) MarkReplaced(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error {
	m.markReplacedHit++
	if m.markReplacedFn == nil {
		return errors.New("unexpected MarkReplaced call")
	}
	return m.markReplacedFn(ctx, id, replacedBy)
}

func (m *mockRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID uuid.UUID) error {
	m.revokeAllByUserIDHit++
	if m.revokeAllByUserIDFn == nil {
		return nil
	}
	return m.revokeAllByUserIDFn(ctx, userID)
}

type fixedReader struct {
	value byte
}

func (r fixedReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.value
	}
	return len(p), nil
}

type errReader struct{}

func (r errReader) Read(p []byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestRegisterSuccess(t *testing.T) {
	userRepo := &mockUserRepo{}
	refreshRepo := &mockRefreshTokenRepo{}

	userRepo.findByEmailFn = func(ctx context.Context, email string) (*models.User, error) {
		if email != "new@example.com" {
			t.Fatalf("unexpected normalized email: %s", email)
		}
		return nil, nil
	}

	userRepo.createUserFn = func(ctx context.Context, user *models.User) error {
		if user.Role != "member" {
			t.Fatalf("expected role member, got %s", user.Role)
		}
		user.ID = uuid.New()
		return nil
	}

	refreshRepo.createFn = func(ctx context.Context, token *models.RefreshToken) error {
		token.ID = uuid.New()
		return nil
	}

	svc := newTestAuthService(userRepo, refreshRepo)

	resp, err := svc.Register(context.Background(), models.RegisterRequest{
		Email:       "New@Example.com",
		Password:    "password123",
		DisplayName: "New User",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if resp == nil || resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty auth response tokens")
	}
	if resp.User.PasswordHash != "" {
		t.Fatal("expected password hash to be hidden in response")
	}
}

func TestRegisterEmailTaken(t *testing.T) {
	existing := &models.User{ID: uuid.New(), Email: "exists@example.com"}
	userRepo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return existing, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			t.Fatal("refresh token should not be created when email exists")
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	_, err := svc.Register(context.Background(), models.RegisterRequest{
		Email:       "exists@example.com",
		Password:    "password123",
		DisplayName: "Exists",
	})
	if !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegisterShortPasswordValidation(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			t.Fatal("find by email should not be called on invalid input")
			return nil, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			t.Fatal("refresh token should not be created on invalid input")
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	_, err := svc.Register(context.Background(), models.RegisterRequest{
		Email:       "new@example.com",
		Password:    "1234567",
		DisplayName: "New User",
	})
	if err == nil {
		t.Fatal("expected validation error for short password")
	}
}

func TestLoginSuccess(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &models.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		PasswordHash: string(passwordHash),
		DisplayName:  "User",
		Role:         "member",
		TokenVersion: 2,
	}

	userRepo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return user, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	resp, err := svc.Login(context.Background(), models.LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if resp == nil || resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty login response")
	}
}

func TestLoginUserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return nil, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			t.Fatal("refresh token should not be created when login fails")
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	_, err := svc.Login(context.Background(), models.LoginRequest{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	userRepo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email string) (*models.User, error) {
			return &models.User{
				ID:           uuid.New(),
				Email:        email,
				PasswordHash: string(passwordHash),
				Role:         "member",
			}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			t.Fatal("refresh token should not be created with wrong password")
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	_, err = svc.Login(context.Background(), models.LoginRequest{
		Email:    "user@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestRefreshSuccess(t *testing.T) {
	userID := uuid.New()
	oldTokenID := uuid.New()
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	refreshHex := "01020304"
	rawRefresh, _ := hex.DecodeString(refreshHex)
	expectedHash := sha256.Sum256(rawRefresh)

	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{
				ID:           userID,
				Role:         "member",
				TokenVersion: 3,
			}, nil
		},
	}

	var createdToken *models.RefreshToken
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			if hex.EncodeToString(hash) != hex.EncodeToString(expectedHash[:]) {
				t.Fatal("unexpected refresh token hash")
			}
			return &models.RefreshToken{
				ID:        oldTokenID,
				UserID:    userID,
				ExpiresAt: now.Add(1 * time.Hour),
			}, nil
		},
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.ID = uuid.New()
			createdToken = token
			return nil
		},
		markReplacedFn: func(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error {
			if id != oldTokenID {
				t.Fatalf("expected old token id %s, got %s", oldTokenID, id)
			}
			if createdToken == nil || replacedBy != createdToken.ID {
				t.Fatal("expected replacedBy to match created token id")
			}
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	svc.now = func() time.Time { return now }

	resp, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: refreshHex})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if resp == nil || resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty refresh response")
	}
	if refreshRepo.markReplacedHit != 1 {
		t.Fatalf("expected MarkReplaced called once, got %d", refreshRepo.markReplacedHit)
	}
}

func TestRefreshRevokedTokenTriggersLeakPath(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	rawRefresh := []byte{1, 2, 3, 4}
	revokedAt := now.Add(-10 * time.Minute)

	userRepo := &mockUserRepo{
		incrementTokenVersionFn: func(ctx context.Context, id uuid.UUID) error {
			if id != userID {
				t.Fatalf("unexpected user id: %s", id)
			}
			return nil
		},
	}

	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:        uuid.New(),
				UserID:    userID,
				ExpiresAt: now.Add(1 * time.Hour),
				RevokedAt: &revokedAt,
			}, nil
		},
		revokeAllByUserIDFn: func(ctx context.Context, id uuid.UUID) error {
			if id != userID {
				t.Fatalf("unexpected user id: %s", id)
			}
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	svc.now = func() time.Time { return now }

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: hex.EncodeToString(rawRefresh)})
	if !errors.Is(err, ErrTokenReused) {
		t.Fatalf("expected ErrTokenReused, got %v", err)
	}
	if refreshRepo.revokeAllByUserIDHit != 1 {
		t.Fatalf("expected RevokeAllByUserID called once, got %d", refreshRepo.revokeAllByUserIDHit)
	}
	if userRepo.incrementTokenVersionHit != 1 {
		t.Fatalf("expected IncrementTokenVersion called once, got %d", userRepo.incrementTokenVersionHit)
	}
}

func TestRefreshReplacedTokenTriggersLeakPath(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	rawRefresh := []byte{5, 6, 7, 8}
	replacedBy := uuid.New()

	userRepo := &mockUserRepo{
		incrementTokenVersionFn: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:         uuid.New(),
				UserID:     userID,
				ExpiresAt:  now.Add(1 * time.Hour),
				ReplacedBy: &replacedBy,
			}, nil
		},
		revokeAllByUserIDFn: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	svc.now = func() time.Time { return now }

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: hex.EncodeToString(rawRefresh)})
	if !errors.Is(err, ErrTokenReused) {
		t.Fatalf("expected ErrTokenReused, got %v", err)
	}
	if refreshRepo.revokeAllByUserIDHit != 1 {
		t.Fatalf("expected RevokeAllByUserID called once, got %d", refreshRepo.revokeAllByUserIDHit)
	}
	if userRepo.incrementTokenVersionHit != 1 {
		t.Fatalf("expected IncrementTokenVersion called once, got %d", userRepo.incrementTokenVersionHit)
	}
}

func TestRefreshExpired(t *testing.T) {
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				ExpiresAt: now.Add(-1 * time.Minute),
			}, nil
		},
	}

	svc := newTestAuthService(&mockUserRepo{}, refreshRepo)
	svc.now = func() time.Time { return now }

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: "0102"})
	if !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestLogoutWithTokenRevokes(t *testing.T) {
	userID := uuid.New()
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:     uuid.New(),
				UserID: userID,
			}, nil
		},
		revokeAllByUserIDFn: func(ctx context.Context, id uuid.UUID) error {
			if id != userID {
				t.Fatalf("unexpected user id: %s", id)
			}
			return nil
		},
	}

	svc := newTestAuthService(&mockUserRepo{}, refreshRepo)
	if err := svc.Logout(context.Background(), "0102"); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if refreshRepo.revokeAllByUserIDHit != 1 {
		t.Fatalf("expected RevokeAllByUserID called once, got %d", refreshRepo.revokeAllByUserIDHit)
	}
}

func TestLogoutWithoutTokenIsSilent(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			t.Fatal("find should not be called on empty token")
			return nil, nil
		},
	}

	svc := newTestAuthService(&mockUserRepo{}, refreshRepo)
	if err := svc.Logout(context.Background(), "   "); err != nil {
		t.Fatalf("logout: %v", err)
	}
}

func TestChangePasswordSuccess(t *testing.T) {
	userID := uuid.New()
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	updatedHash := ""
	findByIDCalls := 0

	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			findByIDCalls++
			if findByIDCalls == 1 {
				return &models.User{
					ID:           userID,
					PasswordHash: string(oldHash),
					Role:         "member",
					TokenVersion: 0,
				}, nil
			}
			return &models.User{
				ID:           userID,
				PasswordHash: updatedHash,
				Role:         "member",
				TokenVersion: 1,
			}, nil
		},
		updatePasswordHashFn: func(ctx context.Context, id uuid.UUID, hash string) error {
			updatedHash = hash
			return nil
		},
	}

	refreshRepo := &mockRefreshTokenRepo{
		revokeAllByUserIDFn: func(ctx context.Context, id uuid.UUID) error {
			if id != userID {
				t.Fatalf("unexpected user id: %s", id)
			}
			return nil
		},
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	resp, err := svc.ChangePassword(context.Background(), userID, "old-password", "new-password")
	if err != nil {
		t.Fatalf("change password: %v", err)
	}
	if resp == nil || resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected non-empty token response")
	}
	if updatedHash == "" {
		t.Fatal("expected password hash to be updated")
	}
	if refreshRepo.revokeAllByUserIDHit != 1 {
		t.Fatalf("expected RevokeAllByUserID called once, got %d", refreshRepo.revokeAllByUserIDHit)
	}
}

func TestChangePasswordOldPasswordWrong(t *testing.T) {
	userID := uuid.New()
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}

	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{
				ID:           userID,
				PasswordHash: string(oldHash),
				Role:         "member",
			}, nil
		},
		updatePasswordHashFn: func(ctx context.Context, id uuid.UUID, hash string) error {
			t.Fatal("UpdatePasswordHash should not be called when old password is wrong")
			return nil
		},
	}

	refreshRepo := &mockRefreshTokenRepo{
		revokeAllByUserIDFn: func(ctx context.Context, userID uuid.UUID) error {
			t.Fatal("RevokeAllByUserID should not be called when old password is wrong")
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	_, err = svc.ChangePassword(context.Background(), userID, "wrong-old", "new-password")
	if !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("expected ErrInvalidPassword, got %v", err)
	}
}

func TestIssueAccessTokenContainsExpectedClaims(t *testing.T) {
	userRepo := &mockUserRepo{}
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	user := &models.User{
		ID:           uuid.New(),
		Role:         "manager",
		TokenVersion: 42,
	}

	tokenStr, refreshToken, err := svc.issueTokens(context.Background(), user)
	if err != nil {
		t.Fatalf("issue tokens: %v", err)
	}
	if len(refreshToken) != 64 {
		t.Fatalf("expected refresh token hex length 64, got %d", len(refreshToken))
	}

	claims := &models.JWTClaims{}
	parsed, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(svc.cfg.JWTSecret), nil
		},
		jwt.WithoutClaimsValidation(),
	)
	if err != nil || !parsed.Valid {
		t.Fatalf("parse token: %v", err)
	}

	if claims.Sub != user.ID.String() {
		t.Fatalf("expected sub %s, got %s", user.ID, claims.Sub)
	}
	if claims.Role != "manager" {
		t.Fatalf("expected role manager, got %s", claims.Role)
	}
	if claims.TV != 42 {
		t.Fatalf("expected tv 42, got %d", claims.TV)
	}
}

func newTestAuthService(userRepo *mockUserRepo, refreshRepo *mockRefreshTokenRepo) *authService {
	cfg := config.Config{
		JWTSecret:     "test-secret-1234567890",
		JWTAccessTTL:  15 * time.Minute,
		JWTRefreshTTL: 7 * 24 * time.Hour,
	}

	svc := &authService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshRepo,
		cfg:              cfg,
		now:              func() time.Time { return time.Date(2026, 4, 21, 8, 0, 0, 0, time.UTC) },
		randReader:       fixedReader{value: 0xAB},
		bcryptCost:       bcrypt.MinCost,
	}

	return svc
}

func TestRefreshInvalidHexReturnsInvalidToken(t *testing.T) {
	svc := newTestAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{})

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: "not-hex"})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestChangePasswordShortNewPasswordValidation(t *testing.T) {
	svc := newTestAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{})

	_, err := svc.ChangePassword(context.Background(), uuid.New(), "old-password", "short")
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestNewAuthService(t *testing.T) {
	svc := NewAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{}, config.Config{
		JWTSecret: "secret",
	})
	if svc == nil {
		t.Fatal("expected service instance")
	}
}

func TestRefreshTokenNotFound(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return nil, nil
		},
	}
	svc := newTestAuthService(&mockUserRepo{}, refreshRepo)

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: "0102"})
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRefreshUserNotFound(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)

	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return nil, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:        uuid.New(),
				UserID:    userID,
				ExpiresAt: now.Add(1 * time.Hour),
			}, nil
		},
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
		markReplacedFn: func(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error {
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	svc.now = func() time.Time { return now }

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: "0102"})
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRefreshMarkReplacedError(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC)

	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Role: "member"}, nil
		},
	}
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:        uuid.New(),
				UserID:    userID,
				ExpiresAt: now.Add(1 * time.Hour),
			}, nil
		},
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			token.ID = uuid.New()
			return nil
		},
		markReplacedFn: func(ctx context.Context, id uuid.UUID, replacedBy uuid.UUID) error {
			return errors.New("mark failed")
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	svc.now = func() time.Time { return now }

	_, err := svc.Refresh(context.Background(), models.RefreshRequest{RefreshToken: "0102"})
	if err == nil || !strings.Contains(err.Error(), "failed to mark refresh token replaced") {
		t.Fatalf("expected mark replaced error, got %v", err)
	}
}

func TestLogoutInvalidToken(t *testing.T) {
	svc := newTestAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{})
	if err := svc.Logout(context.Background(), "bad-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestChangePasswordUserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return nil, nil
		},
	}
	svc := newTestAuthService(userRepo, &mockRefreshTokenRepo{})

	_, err := svc.ChangePassword(context.Background(), uuid.New(), "old-password", "new-password")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}

func TestIssueTokensRefreshCreateError(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepo{
		createFn: func(ctx context.Context, token *models.RefreshToken) error {
			return errors.New("create failed")
		},
	}
	svc := newTestAuthService(&mockUserRepo{}, refreshRepo)
	user := &models.User{ID: uuid.New(), Role: "member"}

	_, _, err := svc.issueTokens(context.Background(), user)
	if err == nil || !strings.Contains(err.Error(), "failed to create refresh token") {
		t.Fatalf("expected create refresh token error, got %v", err)
	}
}

func TestGenerateRefreshTokenRawReaderError(t *testing.T) {
	svc := newTestAuthService(&mockUserRepo{}, &mockRefreshTokenRepo{})
	svc.randReader = errReader{}

	_, err := svc.generateRefreshTokenRaw()
	if err == nil || !strings.Contains(err.Error(), "failed to generate refresh token bytes") {
		t.Fatalf("expected reader error, got %v", err)
	}
}

func TestLogoutAlreadyRevokedTokenDoesNothing(t *testing.T) {
	now := time.Now().UTC()
	refreshRepo := &mockRefreshTokenRepo{
		findByTokenHashFn: func(ctx context.Context, hash []byte) (*models.RefreshToken, error) {
			return &models.RefreshToken{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				RevokedAt: &now,
			}, nil
		},
		revokeAllByUserIDFn: func(ctx context.Context, userID uuid.UUID) error {
			t.Fatal("RevokeAllByUserID should not be called for already revoked token")
			return nil
		},
	}

	svc := newTestAuthService(&mockUserRepo{}, refreshRepo)
	if err := svc.Logout(context.Background(), "0102"); err != nil {
		t.Fatalf("logout: %v", err)
	}
}

func TestChangePasswordUpdateHashError(t *testing.T) {
	userID := uuid.New()
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}

	userRepo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{
				ID:           userID,
				PasswordHash: string(oldHash),
				Role:         "member",
			}, nil
		},
		updatePasswordHashFn: func(ctx context.Context, id uuid.UUID, hash string) error {
			return errors.New("update failed")
		},
	}

	refreshRepo := &mockRefreshTokenRepo{
		revokeAllByUserIDFn: func(ctx context.Context, userID uuid.UUID) error {
			t.Fatal("RevokeAllByUserID should not run when UpdatePasswordHash fails")
			return nil
		},
	}

	svc := newTestAuthService(userRepo, refreshRepo)
	_, err = svc.ChangePassword(context.Background(), userID, "old-password", "new-password")
	if err == nil || !strings.Contains(err.Error(), "failed to update password hash") {
		t.Fatalf("expected update password hash error, got %v", err)
	}
}
