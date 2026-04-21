package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/service"
	"fpgwiki/backend/internal/ws"
)

type userHandlerMockUserService struct {
	getByIDFn func(ctx context.Context, userID uuid.UUID) (*models.User, error)
	updateFn  func(ctx context.Context, userID uuid.UUID, req models.UpdateUserRequest) (*models.User, error)
}

func (m *userHandlerMockUserService) GetByID(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	if m.getByIDFn == nil {
		return nil, errors.New("unexpected GetByID call")
	}
	return m.getByIDFn(ctx, userID)
}

func (m *userHandlerMockUserService) Update(ctx context.Context, userID uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
	if m.updateFn == nil {
		return nil, errors.New("unexpected Update call")
	}
	return m.updateFn(ctx, userID, req)
}

type userHandlerMockAuthService struct {
	forceLogoutFn func(ctx context.Context, targetUserID uuid.UUID) error
}

func (m *userHandlerMockAuthService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	return nil, errors.New("unexpected Register call")
}

func (m *userHandlerMockAuthService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	return nil, errors.New("unexpected Login call")
}

func (m *userHandlerMockAuthService) Refresh(ctx context.Context, req models.RefreshRequest) (*models.TokenResponse, error) {
	return nil, errors.New("unexpected Refresh call")
}

func (m *userHandlerMockAuthService) Logout(ctx context.Context, refreshTokenHex string) error {
	return errors.New("unexpected Logout call")
}

func (m *userHandlerMockAuthService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.TokenResponse, error) {
	return nil, errors.New("unexpected ChangePassword call")
}

func (m *userHandlerMockAuthService) ForceLogout(ctx context.Context, targetUserID uuid.UUID) error {
	if m.forceLogoutFn == nil {
		return errors.New("unexpected ForceLogout call")
	}
	return m.forceLogoutFn(ctx, targetUserID)
}

type userHandlerEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		ID          uuid.UUID `json:"id"`
		Email       string    `json:"email"`
		DisplayName string    `json:"display_name"`
		Role        string    `json:"role"`
		Locale      string    `json:"locale"`
		CreatedAt   time.Time `json:"created_at"`
	} `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestUserHandlerGetMeSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	user := &models.User{
		ID:           userID,
		Email:        "user@example.com",
		PasswordHash: "secret",
		DisplayName:  "User",
		Role:         "member",
		Locale:       "zh",
		CreatedAt:    time.Now().UTC(),
	}

	userSvc := &userHandlerMockUserService{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			if id != userID {
				t.Fatalf("unexpected user id: %s", id)
			}
			return user, nil
		},
	}
	authSvc := &userHandlerMockAuthService{
		forceLogoutFn: func(ctx context.Context, targetUserID uuid.UUID) error {
			return nil
		},
	}
	h := NewUserHandler(userSvc, authSvc, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.GET("/api/me", h.GetMe)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var env userHandlerEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !env.Success {
		t.Fatal("expected success=true")
	}
	if env.Data.ID != userID {
		t.Fatalf("expected id %s, got %s", userID, env.Data.ID)
	}
}

func TestUserHandlerUpdateMeDisplayNameSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	userSvc := &userHandlerMockUserService{
		updateFn: func(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
			if id != userID {
				t.Fatalf("unexpected user id: %s", id)
			}
			if req.DisplayName == nil || *req.DisplayName != "New Name" {
				t.Fatalf("unexpected display_name: %+v", req.DisplayName)
			}
			return &models.User{
				ID:          userID,
				Email:       "user@example.com",
				DisplayName: "New Name",
				Role:        "member",
				Locale:      "zh",
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}
	authSvc := &userHandlerMockAuthService{
		forceLogoutFn: func(ctx context.Context, targetUserID uuid.UUID) error { return nil },
	}
	h := NewUserHandler(userSvc, authSvc, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.PATCH("/api/me", h.UpdateMe)

	body := []byte(`{"display_name":"New Name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUserHandlerUpdateMeLocaleSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()

	userSvc := &userHandlerMockUserService{
		updateFn: func(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
			if req.Locale == nil || *req.Locale != "en" {
				t.Fatalf("unexpected locale: %+v", req.Locale)
			}
			return &models.User{
				ID:          userID,
				Email:       "user@example.com",
				DisplayName: "User",
				Role:        "member",
				Locale:      "en",
				CreatedAt:   time.Now().UTC(),
			}, nil
		},
	}
	authSvc := &userHandlerMockAuthService{
		forceLogoutFn: func(ctx context.Context, targetUserID uuid.UUID) error { return nil },
	}
	h := NewUserHandler(userSvc, authSvc, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	router.PATCH("/api/me", h.UpdateMe)

	body := []byte(`{"locale":"en"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestUserHandlerForceLogoutSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	callerID := uuid.New()
	targetID := uuid.New()

	userSvc := &userHandlerMockUserService{
		getByIDFn: func(ctx context.Context, userID uuid.UUID) (*models.User, error) {
			return nil, errors.New("unexpected GetByID call")
		},
		updateFn: func(ctx context.Context, userID uuid.UUID, req models.UpdateUserRequest) (*models.User, error) {
			return nil, errors.New("unexpected Update call")
		},
	}

	forceCalled := false
	authSvc := &userHandlerMockAuthService{
		forceLogoutFn: func(ctx context.Context, uid uuid.UUID) error {
			forceCalled = true
			if uid != targetID {
				t.Fatalf("expected target id %s, got %s", targetID, uid)
			}
			return nil
		},
	}

	hubManager := ws.NewHubManager(zerolog.New(bytes.NewBuffer(nil)))
	h := NewUserHandler(userSvc, authSvc, hubManager)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Next()
	})
	router.POST("/api/admin/users/:id/force-logout", h.ForceLogout)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+targetID.String()+"/force-logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !forceCalled {
		t.Fatal("expected ForceLogout to be called")
	}
}

func TestUserHandlerForceLogoutSelf(t *testing.T) {
	gin.SetMode(gin.TestMode)
	callerID := uuid.New()

	userSvc := &userHandlerMockUserService{}
	authSvc := &userHandlerMockAuthService{
		forceLogoutFn: func(ctx context.Context, uid uuid.UUID) error {
			t.Fatal("force logout should not be called for self")
			return nil
		},
	}
	h := NewUserHandler(userSvc, authSvc, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Next()
	})
	router.POST("/api/admin/users/:id/force-logout", h.ForceLogout)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+callerID.String()+"/force-logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUserHandlerForceLogoutUserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	callerID := uuid.New()
	targetID := uuid.New()

	userSvc := &userHandlerMockUserService{}
	authSvc := &userHandlerMockAuthService{
		forceLogoutFn: func(ctx context.Context, uid uuid.UUID) error {
			return service.ErrUserNotFound
		},
	}
	h := NewUserHandler(userSvc, authSvc, nil)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", callerID)
		c.Next()
	})
	router.POST("/api/admin/users/:id/force-logout", h.ForceLogout)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/users/"+targetID.String()+"/force-logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
