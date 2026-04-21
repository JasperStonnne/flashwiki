package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/httpserver/middleware"
	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/service"
	"fpgwiki/backend/internal/ws"
)

type UserHandler struct {
	userService service.UserService
	authService service.AuthService
	hubManager  *ws.HubManager
}

type UserResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	Locale      string    `json:"locale"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewUserHandler(us service.UserService, as service.AuthService, hm *ws.HubManager) *UserHandler {
	return &UserHandler{
		userService: us,
		authService: as,
		hubManager:  hm,
	}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		WriteErr(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
		return
	}

	user, err := h.userService.GetByID(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			WriteErr(c, http.StatusNotFound, service.ErrUserNotFound.Error(), "user not found")
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusOK, toUserResponse(user))
}

func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		WriteErr(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	user, err := h.userService.Update(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			WriteErr(c, http.StatusNotFound, service.ErrUserNotFound.Error(), "user not found")
		case isValidationError(err):
			WriteErr(c, http.StatusBadRequest, "invalid_request", err.Error())
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	WriteOK(c, http.StatusOK, toUserResponse(user))
}

func (h *UserHandler) ForceLogout(c *gin.Context) {
	targetID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_user_id", "user id must be uuid")
		return
	}

	callerID := middleware.GetUserID(c)
	if callerID == targetID {
		WriteErr(c, http.StatusBadRequest, "cannot_force_logout_self", "cannot force logout self")
		return
	}

	if err := h.authService.ForceLogout(c.Request.Context(), targetID); err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			WriteErr(c, http.StatusNotFound, service.ErrUserNotFound.Error(), "user not found")
		default:
			WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	if h.hubManager != nil {
		h.hubManager.DisconnectUser(targetID)
	}

	WriteOK(c, http.StatusOK, nil)
}

func toUserResponse(user *models.User) UserResponse {
	if user == nil {
		return UserResponse{}
	}

	return UserResponse{
		ID:          user.ID,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Locale:      user.Locale,
		CreatedAt:   user.CreatedAt,
	}
}
