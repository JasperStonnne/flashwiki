package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/service"
)

type GroupHandler struct {
	groupService service.GroupService
}

type createGroupRequest struct {
	Name     string    `json:"name"`
	LeaderID uuid.UUID `json:"leader_id"`
}

type updateGroupRequest struct {
	Name     *string    `json:"name"`
	LeaderID *uuid.UUID `json:"leader_id"`
}

type addMemberRequest struct {
	UserID uuid.UUID `json:"user_id"`
}

type groupResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	LeaderID  uuid.UUID `json:"leader_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type groupListItemResponse struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Leader      groupLeaderResponse `json:"leader"`
	MemberCount int                 `json:"member_count"`
	CreatedAt   time.Time           `json:"created_at"`
}

type groupLeaderResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}

type groupMemberResponse struct {
	GroupID  uuid.UUID `json:"group_id"`
	UserID   uuid.UUID `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

func NewGroupHandler(gs service.GroupService) *GroupHandler {
	return &GroupHandler{groupService: gs}
}

func (h *GroupHandler) ListGroups(c *gin.Context) {
	groups, err := h.groupService.ListGroups(c.Request.Context())
	if err != nil {
		writeGroupServiceError(c, err)
		return
	}

	resp := make([]groupListItemResponse, 0, len(groups))
	for _, group := range groups {
		resp = append(resp, toGroupListItemResponse(group))
	}

	WriteOK(c, http.StatusOK, resp)
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req createGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	group, err := h.groupService.CreateGroup(c.Request.Context(), req.Name, req.LeaderID)
	if err != nil {
		writeGroupServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusCreated, toGroupResponse(group))
}

func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	groupID, ok := groupIDParamOrAbort(c)
	if !ok {
		return
	}

	var req updateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if req.Name == nil && req.LeaderID == nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "at least one field must be provided")
		return
	}

	group, err := h.groupService.UpdateGroup(c.Request.Context(), groupID, req.Name, req.LeaderID)
	if err != nil {
		writeGroupServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, toGroupResponse(group))
}

func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	groupID, ok := groupIDParamOrAbort(c)
	if !ok {
		return
	}

	if err := h.groupService.DeleteGroup(c.Request.Context(), groupID); err != nil {
		writeGroupServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func (h *GroupHandler) ListMembers(c *gin.Context) {
	groupID, ok := groupIDParamOrAbort(c)
	if !ok {
		return
	}

	members, err := h.groupService.ListMembers(c.Request.Context(), groupID)
	if err != nil {
		writeGroupServiceError(c, err)
		return
	}

	resp := make([]groupMemberResponse, 0, len(members))
	for _, member := range members {
		resp = append(resp, toGroupMemberResponse(member))
	}

	WriteOK(c, http.StatusOK, resp)
}

func (h *GroupHandler) AddMember(c *gin.Context) {
	callerID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	groupID, ok := groupIDParamOrAbort(c)
	if !ok {
		return
	}

	var req addMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if err := h.groupService.AddMember(c.Request.Context(), callerID, groupID, req.UserID); err != nil {
		writeGroupServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func (h *GroupHandler) RemoveMember(c *gin.Context) {
	callerID, ok := currentUserIDOrAbort(c)
	if !ok {
		return
	}

	groupID, ok := groupIDParamOrAbort(c)
	if !ok {
		return
	}

	memberUserID, ok := memberUserIDParamOrAbort(c)
	if !ok {
		return
	}

	if err := h.groupService.RemoveMember(c.Request.Context(), callerID, groupID, memberUserID); err != nil {
		writeGroupServiceError(c, err)
		return
	}

	WriteOK(c, http.StatusOK, nil)
}

func groupIDParamOrAbort(c *gin.Context) (uuid.UUID, bool) {
	groupID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_group_id", "group id must be uuid")
		return uuid.Nil, false
	}

	return groupID, true
}

func memberUserIDParamOrAbort(c *gin.Context) (uuid.UUID, bool) {
	userID, err := uuid.Parse(strings.TrimSpace(c.Param("uid")))
	if err != nil {
		WriteErr(c, http.StatusBadRequest, "invalid_user_id", "user id must be uuid")
		return uuid.Nil, false
	}

	return userID, true
}

func writeGroupServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrNotFound.Error(), "group not found")
	case errors.Is(err, service.ErrForbidden):
		WriteErr(c, http.StatusForbidden, service.ErrForbidden.Error(), "forbidden")
	case errors.Is(err, service.ErrInvalidName):
		WriteErr(c, http.StatusBadRequest, service.ErrInvalidName.Error(), "group name is invalid")
	case errors.Is(err, service.ErrLeaderNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrLeaderNotFound.Error(), "leader not found")
	case errors.Is(err, service.ErrNameTaken):
		WriteErr(c, http.StatusConflict, service.ErrNameTaken.Error(), "group name already exists")
	case errors.Is(err, service.ErrCannotRemoveLeader):
		WriteErr(c, http.StatusBadRequest, service.ErrCannotRemoveLeader.Error(), "cannot remove group leader")
	case errors.Is(err, service.ErrMemberNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrMemberNotFound.Error(), "group member not found")
	case errors.Is(err, service.ErrUserNotFound):
		WriteErr(c, http.StatusNotFound, service.ErrUserNotFound.Error(), "user not found")
	case errors.Is(err, service.ErrInvalidInput):
		WriteErr(c, http.StatusBadRequest, service.ErrInvalidInput.Error(), "invalid input")
	default:
		WriteErr(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func toGroupResponse(group *models.Group) groupResponse {
	if group == nil {
		return groupResponse{}
	}

	return groupResponse{
		ID:        group.ID,
		Name:      group.Name,
		LeaderID:  group.LeaderID,
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}
}

func toGroupListItemResponse(group models.GroupWithDetails) groupListItemResponse {
	return groupListItemResponse{
		ID:   group.ID,
		Name: group.Name,
		Leader: groupLeaderResponse{
			ID:          group.LeaderID,
			DisplayName: group.LeaderName,
			Email:       group.LeaderEmail,
		},
		MemberCount: group.MemberCount,
		CreatedAt:   group.CreatedAt,
	}
}

func toGroupMemberResponse(member models.GroupMember) groupMemberResponse {
	return groupMemberResponse{
		GroupID:  member.GroupID,
		UserID:   member.UserID,
		JoinedAt: member.JoinedAt,
	}
}
