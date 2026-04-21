package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"fpgwiki/backend/internal/httpserver/middleware"
	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func NewAuthHandler(s service.AuthService) *AuthHandler {
	return &AuthHandler{authService: s}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			WriteErr(c, http.StatusConflict, service.ErrEmailTaken.Error(), "email already exists")
		case isValidationError(err):
			WriteErr(c, http.StatusBadRequest, "invalid_request", err.Error())
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			WriteErr(c, http.StatusUnauthorized, service.ErrInvalidCredentials.Error(), "invalid credentials")
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	resp, err := h.authService.Refresh(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			WriteErr(c, http.StatusUnauthorized, service.ErrInvalidToken.Error(), "invalid token")
		case errors.Is(err, service.ErrTokenExpired):
			WriteErr(c, http.StatusUnauthorized, service.ErrTokenExpired.Error(), "token expired")
		case errors.Is(err, service.ErrTokenReused):
			WriteErr(c, http.StatusUnauthorized, service.ErrTokenReused.Error(), "token reused")
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
	}

	if err := h.authService.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidToken):
			WriteErr(c, http.StatusUnauthorized, service.ErrInvalidToken.Error(), "invalid token")
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		WriteErr(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
		return
	}

	resp, err := h.authService.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidPassword):
			WriteErr(c, http.StatusUnauthorized, service.ErrInvalidPassword.Error(), "invalid password")
		case errors.Is(err, service.ErrUserNotFound):
			WriteErr(c, http.StatusNotFound, service.ErrUserNotFound.Error(), "user not found")
		case isValidationError(err):
			WriteErr(c, http.StatusBadRequest, "invalid_request", err.Error())
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusOK, resp)
}

func isValidationError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "validation failed")
}
