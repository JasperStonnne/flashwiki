package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/service"
)

type PermissionHandler struct {
	permissionService service.PermissionService
	nodeService       service.NodeService
}

type setNodePermissionsRequest struct {
	Permissions []setPermissionEntryRequest `json:"permissions"`
}

type setPermissionEntryRequest struct {
	SubjectType string    `json:"subject_type"`
	SubjectID   uuid.UUID `json:"subject_id"`
	Level       string    `json:"level"`
}

type nodePermissionResultResponse struct {
	NodeID        uuid.UUID                     `json:"node_id"`
	Permissions   []nodePermissionEntryResponse `json:"permissions"`
	InheritedFrom *uuid.UUID                    `json:"inherited_from"`
}

type nodePermissionEntryResponse struct {
	ID          uuid.UUID `json:"id"`
	NodeID      uuid.UUID `json:"node_id"`
	SubjectType string    `json:"subject_type"`
	SubjectID   uuid.UUID `json:"subject_id"`
	Level       string    `json:"level"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func NewPermissionHandler(ps service.PermissionService, ns service.NodeService) *PermissionHandler {
	return &PermissionHandler{
		permissionService: ps,
		nodeService:       ns,
	}
}

func (h *PermissionHandler) GetNodePermissions(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	nodeID, ok := nodeIDParamOrAbort(c)
	if !ok {
		return
	}

	_, permission, err := h.nodeService.GetNode(c.Request.Context(), userID, nodeID)
	if err != nil {
		writeNodeServiceError(c, err)
		return
	}
	if permission != "manage" {
		WriteErr(c, http.StatusForbidden, service.ErrForbidden.Error(), "forbidden")
		return
	}

	result, err := h.permissionService.GetNodePermissions(c.Request.Context(), nodeID)
	if err != nil {
		writePermissionServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, toNodePermissionResultResponse(result))
}

func (h *PermissionHandler) SetNodePermissions(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	nodeID, ok := nodeIDParamOrAbort(c)
	if !ok {
		return
	}

	_, permission, err := h.nodeService.GetNode(c.Request.Context(), userID, nodeID)
	if err != nil {
		writeNodeServiceError(c, err)
		return
	}
	if permission != "manage" {
		WriteErr(c, http.StatusForbidden, service.ErrForbidden.Error(), "forbidden")
		return
	}

	var req setNodePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	perms := make([]service.SetPermissionEntry, 0, len(req.Permissions))
	for _, perm := range req.Permissions {
		perms = append(perms, service.SetPermissionEntry{
			SubjectType: perm.SubjectType,
			SubjectID:   perm.SubjectID,
			Level:       perm.Level,
		})
	}

	if err := h.permissionService.SetNodePermissions(c.Request.Context(), nodeID, perms); err != nil {
		writePermissionServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func writePermissionServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrNotFound.Error(), "node not found")
	case errors.Is(err, service.ErrForbidden):
		WriteErr(c, http.StatusForbidden, service.ErrForbidden.Error(), "forbidden")
	case isValidationError(err):
		WriteErr(c, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func toNodePermissionResultResponse(result *service.NodePermissionResult) nodePermissionResultResponse {
	if result == nil {
		return nodePermissionResultResponse{}
	}

	perms := make([]nodePermissionEntryResponse, 0, len(result.Permissions))
	for _, perm := range result.Permissions {
		perms = append(perms, nodePermissionEntryResponse{
			ID:          perm.ID,
			NodeID:      perm.NodeID,
			SubjectType: perm.SubjectType,
			SubjectID:   perm.SubjectID,
			Level:       perm.Level,
			CreatedAt:   perm.CreatedAt,
			UpdatedAt:   perm.UpdatedAt,
		})
	}

	return nodePermissionResultResponse{
		NodeID:        result.NodeID,
		Permissions:   perms,
		InheritedFrom: result.InheritedFrom,
	}
}
