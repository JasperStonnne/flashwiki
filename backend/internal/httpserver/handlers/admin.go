package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/service"
)

type AdminHandler struct {
	userService service.UserService
	authService service.AuthService
}

type changeRoleRequest struct {
	Role string `json:"role"`
}

func NewAdminHandler(us service.UserService, as service.AuthService) *AdminHandler {
	return &AdminHandler{
		userService: us,
		authService: as,
	}
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListAll(c.Request.Context())
	if err != nil {
		writeAdminUserServiceError(c, err)
		return
	}

	resp := make([]UserResponse, 0, len(users))
	for _, user := range users {
		resp = append(resp, toUserResponse(user))
	}

	WriteOK(c, http.StatusOK, resp)
}

func (h *AdminHandler) ChangeRole(c *gin.Context) {
	callerID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	targetID, ok := adminUserIDParamOrAbort(c)
	if !ok {
		return
	}

	var req changeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	user, err := h.userService.ChangeRole(c.Request.Context(), callerID, targetID, req.Role)
	if err != nil {
		writeAdminUserServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, toUserResponse(user))
}

func adminUserIDParamOrAbort(c *gin.Context) (uuid.UUID, bool) {
	userID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_user_id", "user id must be uuid")
		return uuid.Nil, false
	}

	return userID, true
}

func writeAdminUserServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrUserNotFound.Error(), "user not found")
	case errors.Is(err, service.ErrCannotChangeSelf):
		WriteErr(c, http.StatusBadRequest, service.ErrCannotChangeSelf.Error(), "cannot change own role")
	case errors.Is(err, service.ErrInvalidInput):
		WriteErr(c, http.StatusBadRequest, service.ErrInvalidInput.Error(), "invalid input")
	default:
		WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
