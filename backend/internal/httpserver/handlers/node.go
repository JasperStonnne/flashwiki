package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/httpserver/middleware"
	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/service"
)

type NodeHandler struct {
	nodeService       service.NodeService
	permissionService service.PermissionService
}

type createNodeRequest struct {
	ParentID *uuid.UUID `json:"parent_id"`
	Kind     string     `json:"kind"`
	Title    string     `json:"title"`
}

type updateNodeRequest struct {
	Title    *string         `json:"title"`
	ParentID json.RawMessage `json:"parent_id"`
}

type nodeResponse struct {
	ID        uuid.UUID  `json:"id"`
	ParentID  *uuid.UUID `json:"parent_id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	OwnerID   uuid.UUID  `json:"owner_id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type nodeDetailResponse struct {
	nodeResponse
	Permission string `json:"permission"`
}

type nodeListItemResponse struct {
	nodeResponse
	Permission  string `json:"permission"`
	HasChildren bool   `json:"has_children"`
}

func NewNodeHandler(ns service.NodeService, ps service.PermissionService) *NodeHandler {
	return &NodeHandler{
		nodeService:       ns,
		permissionService: ps,
	}
}

func (h *NodeHandler) CreateNode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	var req createNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	node, err := h.nodeService.CreateNode(c.Request.Context(), userID, service.CreateNodeRequest{
		ParentID: req.ParentID,
		Kind:     req.Kind,
		Title:    req.Title,
	})
	if err != nil {
		writeNodeServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusCreated, toNodeResponse(node))
}

func (h *NodeHandler) GetNode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	nodeID, ok := nodeIDParamOrAbort(c)
	if !ok {
		return
	}

	node, permission, err := h.nodeService.GetNode(c.Request.Context(), userID, nodeID)
	if err != nil {
		writeNodeServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, toNodeDetailResponse(node, permission))
}

func (h *NodeHandler) ListChildren(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	parentID, ok := parentQueryOrAbort(c)
	if !ok {
		return
	}

	nodes, err := h.nodeService.ListChildren(c.Request.Context(), userID, parentID)
	if err != nil {
		writeNodeServiceError(c, err)
		return
	}

	resp := make([]nodeListItemResponse, 0, len(nodes))
	for _, node := range nodes {
		resp = append(resp, toNodeListItemResponse(node))
	}

	WriteOK(c, http.StatusOK, resp)
}

func (h *NodeHandler) UpdateNode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	nodeID, ok := nodeIDParamOrAbort(c)
	if !ok {
		return
	}

	var req updateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	hasParentID := len(req.ParentID) > 0
	if !hasParentID && req.Title == nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "at least one field must be provided")
		return
	}

	var (
		node *models.Node
		err  error
	)

	if hasParentID {
		parentID, parseErr := parseOptionalUUIDJSON(req.ParentID)
		if parseErr != nil {
			WriteErr(c, http.StatusBadRequest, "invalid_parent_id", "parent id must be uuid or null")
			return
		}

		node, err = h.nodeService.MoveNode(c.Request.Context(), userID, nodeID, parentID)
		if err != nil {
			writeNodeServiceError(c, err)
			return
		}
	}

	if req.Title != nil {
		node, err = h.nodeService.RenameNode(c.Request.Context(), userID, nodeID, *req.Title)
		if err != nil {
			writeNodeServiceError(c, err)
			return
		}
	}

	WriteOK(c, http.StatusOK, toNodeResponse(node))
}

func (h *NodeHandler) DeleteNode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	nodeID, ok := nodeIDParamOrAbort(c)
	if !ok {
		return
	}

	if err := h.nodeService.SoftDeleteNode(c.Request.Context(), userID, nodeID); err != nil {
		writeNodeServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func (h *NodeHandler) RestoreNode(c *gin.Context) {
	userID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	nodeID, ok := nodeIDParamOrAbort(c)
	if !ok {
		return
	}

	if err := h.nodeService.RestoreNode(c.Request.Context(), userID, nodeID); err != nil {
		writeNodeServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func currentUserIDOrAbort(c *gin.Context) (uuid.UUID, bool) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		WriteErr(c, http.StatusUnauthorized, "missing_token", "authorization token is required")
		return uuid.Nil, false
	}

	return userID, true
}

func nodeIDParamOrAbort(c *gin.Context) (uuid.UUID, bool) {
	nodeID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_node_id", "node id must be uuid")
		return uuid.Nil, false
	}

	return nodeID, true
}

func parentQueryOrAbort(c *gin.Context) (*uuid.UUID, bool) {
	parentParam := strings.TrimSpace(c.Query("parent"))
	if parentParam == "" || strings.EqualFold(parentParam, "null") {
		return nil, true
	}

	parentID, err := uuid.Parse(parentParam)
	if err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_parent_id", "parent id must be uuid or null")
		return nil, false
	}

	return &parentID, true
}

func parseOptionalUUIDJSON(raw json.RawMessage) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}

	var id uuid.UUID
	if err := json.Unmarshal(raw, &id); err != nil {
		return nil, err
	}

	return &id, nil
}

func writeNodeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrNotFound.Error(), "node not found")
	case errors.Is(err, service.ErrForbidden):
		WriteErr(c, http.StatusForbidden, service.ErrForbidden.Error(), "forbidden")
	case errors.Is(err, service.ErrParentNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrParentNotFound.Error(), "parent node not found")
	case errors.Is(err, service.ErrParentNotFolder):
		WriteErr(c, http.StatusBadRequest, service.ErrParentNotFolder.Error(), "parent node must be folder")
	case errors.Is(err, service.ErrInvalidKind):
		WriteErr(c, http.StatusBadRequest, service.ErrInvalidKind.Error(), "kind must be folder or doc")
	case errors.Is(err, service.ErrTitleTooLong):
		WriteErr(c, http.StatusBadRequest, service.ErrTitleTooLong.Error(), "title is too long")
	case errors.Is(err, service.ErrInvalidInput):
		WriteErr(c, http.StatusBadRequest, service.ErrInvalidInput.Error(), "invalid input")
	case errors.Is(err, service.ErrCircularMove):
		WriteErr(c, http.StatusBadRequest, service.ErrCircularMove.Error(), "cannot move node into its descendant")
	case errors.Is(err, service.ErrMoveToSelf):
		WriteErr(c, http.StatusBadRequest, service.ErrMoveToSelf.Error(), "cannot move node to itself")
	case errors.Is(err, service.ErrNotDeleted):
		WriteErr(c, http.StatusBadRequest, service.ErrNotDeleted.Error(), "node is not deleted")
	case errors.Is(err, service.ErrRestoreExpired):
		WriteErr(c, http.StatusBadRequest, service.ErrRestoreExpired.Error(), "restore window has expired")
	case errors.Is(err, service.ErrParentDeleted):
		WriteErr(c, http.StatusBadRequest, service.ErrParentDeleted.Error(), "parent node is deleted")
	default:
		WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func toNodeResponse(node *models.Node) nodeResponse {
	if node == nil {
		return nodeResponse{}
	}

	return nodeResponse{
		ID:        node.ID,
		ParentID:  node.ParentID,
		Kind:      node.Kind,
		Title:     node.Title,
		OwnerID:   node.OwnerID,
		CreatedAt: node.CreatedAt,
		UpdatedAt: node.UpdatedAt,
	}
}

func toNodeDetailResponse(node *models.Node, permission string) nodeDetailResponse {
	return nodeDetailResponse{
		nodeResponse: toNodeResponse(node),
		Permission:   permission,
	}
}

func toNodeListItemResponse(node models.NodeWithPermission) nodeListItemResponse {
	return nodeListItemResponse{
		nodeResponse: toNodeResponse(&node.Node),
		Permission:   node.Permission,
		HasChildren:  node.HasChildren,
	}
}
