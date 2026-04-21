package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/config"
	"fpgwiki/backend/internal/models"
)

type authErrorEnvelope struct {
	Success bool `json:"success"`
	Error   struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type mockUserRepoForMiddleware struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*models.User, error)
}

func (m *mockUserRepoForMiddleware) CreateUser(ctx context.Context, user *models.User) error {
	return errors.New("unexpected CreateUser call")
}

func (m *mockUserRepoForMiddleware) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return nil, errors.New("unexpected FindByEmail call")
}

func (m *mockUserRepoForMiddleware) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.findByIDFn == nil {
		return nil, errors.New("unexpected FindByID call")
	}
	return m.findByIDFn(ctx, id)
}

func (m *mockUserRepoForMiddleware) UpdateUser(ctx context.Context, user *models.User) error {
	return errors.New("unexpected UpdateUser call")
}

func (m *mockUserRepoForMiddleware) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string) error {
	return errors.New("unexpected UpdatePasswordHash call")
}

func (m *mockUserRepoForMiddleware) IncrementTokenVersion(ctx context.Context, id uuid.UUID) error {
	return errors.New("unexpected IncrementTokenVersion call")
}

func TestRequireAuthMissingAuthorizationHeader(t *testing.T) {
	cfg := testAuthConfig()
	repo := &mockUserRepoForMiddleware{}

	router := gin.New()
	router.Use(RequireAuth(cfg, repo))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec, "missing_token")
}

func TestRequireAuthInvalidTokenFormat(t *testing.T) {
	cfg := testAuthConfig()
	repo := &mockUserRepoForMiddleware{}

	router := gin.New()
	router.Use(RequireAuth(cfg, repo))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token abc.def")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec, "invalid_token")
}

func TestRequireAuthExpiredToken(t *testing.T) {
	cfg := testAuthConfig()
	userID := uuid.New()
	token := signToken(t, cfg, userID, "member", 0, time.Now().UTC().Add(-1*time.Minute))

	repo := &mockUserRepoForMiddleware{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Role: "member", TokenVersion: 0}, nil
		},
	}

	router := gin.New()
	router.Use(RequireAuth(cfg, repo))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec, "invalid_token")
}

func TestRequireAuthTokenVersionMismatch(t *testing.T) {
	cfg := testAuthConfig()
	userID := uuid.New()
	token := signToken(t, cfg, userID, "member", 1, time.Now().UTC().Add(10*time.Minute))

	repo := &mockUserRepoForMiddleware{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Role: "member", TokenVersion: 2}, nil
		},
	}

	router := gin.New()
	router.Use(RequireAuth(cfg, repo))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec, "token_revoked")
}

func TestRequireAuthUserNotFound(t *testing.T) {
	cfg := testAuthConfig()
	userID := uuid.New()
	token := signToken(t, cfg, userID, "member", 1, time.Now().UTC().Add(10*time.Minute))

	repo := &mockUserRepoForMiddleware{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return nil, nil
		},
	}

	router := gin.New()
	router.Use(RequireAuth(cfg, repo))
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	assertErrorCode(t, rec, "user_not_found")
}

func TestRequireAuthSuccessInjectsContext(t *testing.T) {
	cfg := testAuthConfig()
	userID := uuid.New()
	token := signToken(t, cfg, userID, "manager", 5, time.Now().UTC().Add(10*time.Minute))

	expectedUser := &models.User{
		ID:           userID,
		Role:         "manager",
		TokenVersion: 5,
	}
	repo := &mockUserRepoForMiddleware{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return expectedUser, nil
		},
	}

	nextCalled := false
	router := gin.New()
	router.Use(RequireAuth(cfg, repo))
	router.GET("/protected", func(c *gin.Context) {
		nextCalled = true

		if got := GetUserID(c); got != userID {
			t.Fatalf("expected user id %s, got %s", userID, got)
		}
		if got := GetUserRole(c); got != "manager" {
			t.Fatalf("expected role manager, got %s", got)
		}
		gotUser := GetUser(c)
		if gotUser == nil || gotUser.ID != expectedUser.ID {
			t.Fatalf("expected user %+v, got %+v", expectedUser, gotUser)
		}

		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to execute")
	}
}

func TestRequireRoleMatched(t *testing.T) {
	nextCalled := false
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(contextUserRoleKey, "manager")
		c.Next()
	})
	router.Use(RequireRole("manager", "member"))
	router.GET("/admin", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to execute")
	}
}

func TestRequireRoleForbidden(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(contextUserRoleKey, "member")
		c.Next()
	})
	router.Use(RequireRole("manager"))
	router.GET("/admin", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
	assertErrorCode(t, rec, "forbidden")
}

func TestContextHelperFallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	if GetUserID(c) != uuid.Nil {
		t.Fatal("expected zero user id for empty context")
	}
	if GetUserRole(c) != "" {
		t.Fatal("expected empty role for empty context")
	}
	if GetUser(c) != nil {
		t.Fatal("expected nil user for empty context")
	}

	user := models.User{ID: uuid.New(), Role: "member"}
	c.Set(contextUserKey, user)
	gotUser := GetUser(c)
	if gotUser == nil || gotUser.ID != user.ID {
		t.Fatalf("expected user %+v, got %+v", user, gotUser)
	}
}

func TestOptionalAuthPassThrough(t *testing.T) {
	cfg := testAuthConfig()
	nextCalled := false

	router := gin.New()
	router.Use(OptionalAuth(cfg))
	router.GET("/any", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/any", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to execute")
	}
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, code string) {
	t.Helper()

	var envelope authErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if envelope.Success {
		t.Fatalf("expected success=false, got true")
	}
	if envelope.Error.Code != code {
		t.Fatalf("expected error code %q, got %q", code, envelope.Error.Code)
	}
}

func testAuthConfig() config.Config {
	return config.Config{
		JWTSecret: "test-secret",
	}
}

func signToken(t *testing.T, cfg config.Config, userID uuid.UUID, role string, tokenVersion int64, expiresAt time.Time) string {
	t.Helper()

	now := expiresAt.Add(-5 * time.Minute)
	claims := models.JWTClaims{
		Sub:  userID.String(),
		Role: role,
		TV:   tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return tokenString
}
