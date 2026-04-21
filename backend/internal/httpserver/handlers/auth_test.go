package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/service"
)

type authHandlerMockService struct {
	registerFn       func(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error)
	loginFn          func(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error)
	refreshFn        func(ctx context.Context, req models.RefreshRequest) (*models.TokenResponse, error)
	logoutFn         func(ctx context.Context, refreshTokenHex string) error
	changePasswordFn func(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.TokenResponse, error)
	forceLogoutFn    func(ctx context.Context, targetUserID uuid.UUID) error
}

func (m *authHandlerMockService) Register(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
	if m.registerFn == nil {
		return nil, errors.New("unexpected Register call")
	}
	return m.registerFn(ctx, req)
}

func (m *authHandlerMockService) Login(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
	if m.loginFn == nil {
		return nil, errors.New("unexpected Login call")
	}
	return m.loginFn(ctx, req)
}

func (m *authHandlerMockService) Refresh(ctx context.Context, req models.RefreshRequest) (*models.TokenResponse, error) {
	if m.refreshFn == nil {
		return nil, errors.New("unexpected Refresh call")
	}
	return m.refreshFn(ctx, req)
}

func (m *authHandlerMockService) Logout(ctx context.Context, refreshTokenHex string) error {
	if m.logoutFn == nil {
		return errors.New("unexpected Logout call")
	}
	return m.logoutFn(ctx, refreshTokenHex)
}

func (m *authHandlerMockService) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) (*models.TokenResponse, error) {
	if m.changePasswordFn == nil {
		return nil, errors.New("unexpected ChangePassword call")
	}
	return m.changePasswordFn(ctx, userID, oldPassword, newPassword)
}

func (m *authHandlerMockService) ForceLogout(ctx context.Context, targetUserID uuid.UUID) error {
	if m.forceLogoutFn == nil {
		return errors.New("unexpected ForceLogout call")
	}
	return m.forceLogoutFn(ctx, targetUserID)
}

type authHandlerErrorEnvelope struct {
	Success bool `json:"success"`
	Error   struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestAuthHandlerRegisterEmailTaken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &authHandlerMockService{
		registerFn: func(ctx context.Context, req models.RegisterRequest) (*models.AuthResponse, error) {
			return nil, service.ErrEmailTaken
		},
	}
	h := NewAuthHandler(mockSvc)

	router := gin.New()
	router.POST("/api/auth/register", h.Register)

	body := []byte(`{"email":"taken@example.com","password":"password123","display_name":"User"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}

	var env authHandlerErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if env.Error.Code != service.ErrEmailTaken.Error() {
		t.Fatalf("expected error code %q, got %q", service.ErrEmailTaken.Error(), env.Error.Code)
	}
}

func TestAuthHandlerLoginInvalidCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &authHandlerMockService{
		loginFn: func(ctx context.Context, req models.LoginRequest) (*models.AuthResponse, error) {
			return nil, service.ErrInvalidCredentials
		},
	}
	h := NewAuthHandler(mockSvc)

	router := gin.New()
	router.POST("/api/auth/login", h.Login)

	body := []byte(`{"email":"missing@example.com","password":"bad"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandlerRefreshTokenReused(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mockSvc := &authHandlerMockService{
		refreshFn: func(ctx context.Context, req models.RefreshRequest) (*models.TokenResponse, error) {
			return nil, service.ErrTokenReused
		},
	}
	h := NewAuthHandler(mockSvc)

	router := gin.New()
	router.POST("/api/auth/refresh", h.Refresh)

	body := []byte(`{"refresh_token":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
