//go:build integration

package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"fpgwiki/backend/internal/service"
)

type meResponse struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	Locale      string    `json:"locale"`
	CreatedAt   time.Time `json:"created_at"`
}

type nodeItemResponse struct {
	ID          uuid.UUID  `json:"id"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	Permission  string     `json:"permission"`
	HasChildren bool       `json:"has_children"`
}

type groupResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	LeaderID  uuid.UUID `json:"leader_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type groupLeaderResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
}

type groupListItemResponse struct {
	ID          uuid.UUID           `json:"id"`
	Name        string              `json:"name"`
	Leader      groupLeaderResponse `json:"leader"`
	MemberCount int                 `json:"member_count"`
	CreatedAt   time.Time           `json:"created_at"`
}

type groupMemberResponse struct {
	GroupID  uuid.UUID `json:"group_id"`
	UserID   uuid.UUID `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type nodePermissionResponse struct {
	NodeID        uuid.UUID  `json:"node_id"`
	InheritedFrom *uuid.UUID `json:"inherited_from"`
}

func TestNodeTreeIntegration(t *testing.T) {
	pool := newIntegrationPool(t)
	router := newIntegrationRouter(t, pool)

	suffix := time.Now().UnixNano()
	managerEmail := fmt.Sprintf("nodetree_manager_%d@example.com", suffix)
	userBEmail := fmt.Sprintf("nodetree_userb_%d@example.com", suffix)
	userCEmail := fmt.Sprintf("nodetree_userc_%d@example.com", suffix)
	managerPassword := "Manager123!"
	userBPassword := "UserB123!"
	userCPassword := "UserC123!"

	managerRegister := doJSONRequest(t, router, http.MethodPost, "/api/auth/register", map[string]any{
		"email":        managerEmail,
		"password":     managerPassword,
		"display_name": "Manager A",
	}, "")
	if managerRegister.Code != http.StatusCreated {
		t.Fatalf("register managerA: expected 201, got %d, body=%s", managerRegister.Code, managerRegister.Body.String())
	}
	managerTokens := parseAuthEnvelope(t, managerRegister)
	managerMe := fetchMe(t, router, managerTokens.AccessToken)

	userBRegister := doJSONRequest(t, router, http.MethodPost, "/api/auth/register", map[string]any{
		"email":        userBEmail,
		"password":     userBPassword,
		"display_name": "User B",
	}, "")
	if userBRegister.Code != http.StatusCreated {
		t.Fatalf("register userB: expected 201, got %d, body=%s", userBRegister.Code, userBRegister.Body.String())
	}
	userBTokens := parseAuthEnvelope(t, userBRegister)
	userBMe := fetchMe(t, router, userBTokens.AccessToken)

	userCRegister := doJSONRequest(t, router, http.MethodPost, "/api/auth/register", map[string]any{
		"email":        userCEmail,
		"password":     userCPassword,
		"display_name": "User C",
	}, "")
	if userCRegister.Code != http.StatusCreated {
		t.Fatalf("register userC: expected 201, got %d, body=%s", userCRegister.Code, userCRegister.Body.String())
	}
	userCTokens := parseAuthEnvelope(t, userCRegister)

	if _, err := pool.Exec(context.Background(), "UPDATE users SET role='manager' WHERE email=$1", managerEmail); err != nil {
		t.Fatalf("promote managerA: %v", err)
	}

	managerLogin := doJSONRequest(t, router, http.MethodPost, "/api/auth/login", map[string]any{
		"email":    managerEmail,
		"password": managerPassword,
	}, "")
	if managerLogin.Code != http.StatusOK {
		t.Fatalf("login managerA: expected 200, got %d, body=%s", managerLogin.Code, managerLogin.Body.String())
	}
	managerTokens = parseAuthEnvelope(t, managerLogin)

	folderRec := doJSONRequest(t, router, http.MethodPost, "/api/nodes", map[string]any{
		"kind":  "folder",
		"title": "产品文档",
	}, managerTokens.AccessToken)
	if folderRec.Code != http.StatusCreated {
		t.Fatalf("create root folder: expected 201, got %d, body=%s", folderRec.Code, folderRec.Body.String())
	}
	folder := parseEnvelopeData[nodeItemResponse](t, folderRec)

	docRec := doJSONRequest(t, router, http.MethodPost, "/api/nodes", map[string]any{
		"parent_id": folder.ID,
		"kind":      "doc",
		"title":     "需求文档",
	}, managerTokens.AccessToken)
	if docRec.Code != http.StatusCreated {
		t.Fatalf("create child doc: expected 201, got %d, body=%s", docRec.Code, docRec.Body.String())
	}
	doc := parseEnvelopeData[nodeItemResponse](t, docRec)

	rootListRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes?parent=null", nil, managerTokens.AccessToken)
	if rootListRec.Code != http.StatusOK {
		t.Fatalf("list root nodes: expected 200, got %d, body=%s", rootListRec.Code, rootListRec.Body.String())
	}
	rootNodes := parseEnvelopeData[[]nodeItemResponse](t, rootListRec)
	if !hasNode(rootNodes, folder.ID) {
		t.Fatalf("expected root list to contain folder %s", folder.ID)
	}

	childListRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes?parent="+folder.ID.String(), nil, managerTokens.AccessToken)
	if childListRec.Code != http.StatusOK {
		t.Fatalf("list folder children: expected 200, got %d, body=%s", childListRec.Code, childListRec.Body.String())
	}
	childNodes := parseEnvelopeData[[]nodeItemResponse](t, childListRec)
	if !hasNode(childNodes, doc.ID) {
		t.Fatalf("expected child list to contain doc %s", doc.ID)
	}

	getDocRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes/"+doc.ID.String(), nil, managerTokens.AccessToken)
	if getDocRec.Code != http.StatusOK {
		t.Fatalf("get doc: expected 200, got %d, body=%s", getDocRec.Code, getDocRec.Body.String())
	}
	docDetail := parseEnvelopeData[nodeItemResponse](t, getDocRec)
	if docDetail.Title != "需求文档" || docDetail.Permission != "manage" {
		t.Fatalf("unexpected doc detail: %+v", docDetail)
	}

	renameRec := doJSONRequest(t, router, http.MethodPatch, "/api/nodes/"+doc.ID.String(), map[string]any{
		"title": "PRD v2",
	}, managerTokens.AccessToken)
	if renameRec.Code != http.StatusOK {
		t.Fatalf("rename doc: expected 200, got %d, body=%s", renameRec.Code, renameRec.Body.String())
	}
	doc = parseEnvelopeData[nodeItemResponse](t, renameRec)
	if doc.Title != "PRD v2" {
		t.Fatalf("expected renamed title PRD v2, got %s", doc.Title)
	}

	archiveRec := doJSONRequest(t, router, http.MethodPost, "/api/nodes", map[string]any{
		"kind":  "folder",
		"title": "归档",
	}, managerTokens.AccessToken)
	if archiveRec.Code != http.StatusCreated {
		t.Fatalf("create archive folder: expected 201, got %d, body=%s", archiveRec.Code, archiveRec.Body.String())
	}
	archiveFolder := parseEnvelopeData[nodeItemResponse](t, archiveRec)

	moveRec := doJSONRequest(t, router, http.MethodPatch, "/api/nodes/"+doc.ID.String(), map[string]any{
		"parent_id": archiveFolder.ID,
	}, managerTokens.AccessToken)
	if moveRec.Code != http.StatusOK {
		t.Fatalf("move doc: expected 200, got %d, body=%s", moveRec.Code, moveRec.Body.String())
	}
	doc = parseEnvelopeData[nodeItemResponse](t, moveRec)
	if doc.ParentID == nil || *doc.ParentID != archiveFolder.ID {
		t.Fatalf("expected moved parent %s, got %+v", archiveFolder.ID, doc.ParentID)
	}

	setPermsRec := doJSONRequest(t, router, http.MethodPut, "/api/nodes/"+folder.ID.String()+"/permissions", map[string]any{
		"permissions": []map[string]any{
			{
				"subject_type": "user",
				"subject_id":   userBMe.ID,
				"level":        "readable",
			},
		},
	}, managerTokens.AccessToken)
	if setPermsRec.Code != http.StatusOK {
		t.Fatalf("set folder permissions: expected 200, got %d, body=%s", setPermsRec.Code, setPermsRec.Body.String())
	}

	userBLogin := doJSONRequest(t, router, http.MethodPost, "/api/auth/login", map[string]any{
		"email":    userBEmail,
		"password": userBPassword,
	}, "")
	if userBLogin.Code != http.StatusOK {
		t.Fatalf("login userB: expected 200, got %d, body=%s", userBLogin.Code, userBLogin.Body.String())
	}
	userBTokens = parseAuthEnvelope(t, userBLogin)

	userBRootRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes?parent=null", nil, userBTokens.AccessToken)
	if userBRootRec.Code != http.StatusOK {
		t.Fatalf("userB list root: expected 200, got %d, body=%s", userBRootRec.Code, userBRootRec.Body.String())
	}
	userBRootNodes := parseEnvelopeData[[]nodeItemResponse](t, userBRootRec)
	if !hasNode(userBRootNodes, folder.ID) {
		t.Fatalf("expected userB to see folder %s", folder.ID)
	}

	userBCreateRec := doJSONRequest(t, router, http.MethodPost, "/api/nodes", map[string]any{
		"parent_id": folder.ID,
		"kind":      "doc",
		"title":     "被拒",
	}, userBTokens.AccessToken)
	if userBCreateRec.Code != http.StatusForbidden {
		t.Fatalf("userB create child: expected 403, got %d, body=%s", userBCreateRec.Code, userBCreateRec.Body.String())
	}

	userBPermRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes/"+folder.ID.String()+"/permissions", nil, userBTokens.AccessToken)
	if userBPermRec.Code != http.StatusForbidden {
		t.Fatalf("userB get permissions: expected 403, got %d, body=%s", userBPermRec.Code, userBPermRec.Body.String())
	}

	userCRootRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes?parent=null", nil, userCTokens.AccessToken)
	if userCRootRec.Code != http.StatusOK {
		t.Fatalf("userC list root: expected 200, got %d, body=%s", userCRootRec.Code, userCRootRec.Body.String())
	}
	userCRootNodes := parseEnvelopeData[[]nodeItemResponse](t, userCRootRec)
	if len(userCRootNodes) != 0 {
		t.Fatalf("expected userC root list to be empty, got %+v", userCRootNodes)
	}

	userCGetFolderRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes/"+folder.ID.String(), nil, userCTokens.AccessToken)
	if userCGetFolderRec.Code != http.StatusNotFound {
		t.Fatalf("userC get folder: expected 404, got %d, body=%s", userCGetFolderRec.Code, userCGetFolderRec.Body.String())
	}

	deleteDocRec := doJSONRequest(t, router, http.MethodDelete, "/api/nodes/"+doc.ID.String(), nil, managerTokens.AccessToken)
	if deleteDocRec.Code != http.StatusOK {
		t.Fatalf("delete doc: expected 200, got %d, body=%s", deleteDocRec.Code, deleteDocRec.Body.String())
	}

	getDeletedDocRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes/"+doc.ID.String(), nil, managerTokens.AccessToken)
	if getDeletedDocRec.Code != http.StatusNotFound {
		t.Fatalf("get deleted doc: expected 404, got %d, body=%s", getDeletedDocRec.Code, getDeletedDocRec.Body.String())
	}

	restoreDocRec := doJSONRequest(t, router, http.MethodPost, "/api/nodes/"+doc.ID.String()+"/restore", nil, managerTokens.AccessToken)
	if restoreDocRec.Code != http.StatusOK {
		t.Fatalf("restore doc: expected 200, got %d, body=%s", restoreDocRec.Code, restoreDocRec.Body.String())
	}

	getRestoredDocRec := doJSONRequest(t, router, http.MethodGet, "/api/nodes/"+doc.ID.String(), nil, managerTokens.AccessToken)
	if getRestoredDocRec.Code != http.StatusOK {
		t.Fatalf("get restored doc: expected 200, got %d, body=%s", getRestoredDocRec.Code, getRestoredDocRec.Body.String())
	}

	groupName := fmt.Sprintf("设计组-%d", suffix)
	groupRec := doJSONRequest(t, router, http.MethodPost, "/api/admin/groups", map[string]any{
		"name":      groupName,
		"leader_id": managerMe.ID,
	}, managerTokens.AccessToken)
	if groupRec.Code != http.StatusCreated {
		t.Fatalf("create group: expected 201, got %d, body=%s", groupRec.Code, groupRec.Body.String())
	}
	group := parseEnvelopeData[groupResponse](t, groupRec)

	groupListRec := doJSONRequest(t, router, http.MethodGet, "/api/admin/groups", nil, managerTokens.AccessToken)
	if groupListRec.Code != http.StatusOK {
		t.Fatalf("list groups: expected 200, got %d, body=%s", groupListRec.Code, groupListRec.Body.String())
	}
	groupList := parseEnvelopeData[[]groupListItemResponse](t, groupListRec)
	groupListItem, ok := findGroup(groupList, group.ID)
	if !ok {
		t.Fatalf("expected groups list to contain group %s", group.ID)
	}
	if groupListItem.Leader.ID != managerMe.ID || groupListItem.MemberCount < 1 {
		t.Fatalf("unexpected group list item: %+v", groupListItem)
	}

	updatedGroupName := fmt.Sprintf("设计部-%d", suffix)
	updateGroupRec := doJSONRequest(t, router, http.MethodPatch, "/api/admin/groups/"+group.ID.String(), map[string]any{
		"name": updatedGroupName,
	}, managerTokens.AccessToken)
	if updateGroupRec.Code != http.StatusOK {
		t.Fatalf("update group: expected 200, got %d, body=%s", updateGroupRec.Code, updateGroupRec.Body.String())
	}

	addMemberRec := doJSONRequest(t, router, http.MethodPost, "/api/admin/groups/"+group.ID.String()+"/members", map[string]any{
		"user_id": userBMe.ID,
	}, managerTokens.AccessToken)
	if addMemberRec.Code != http.StatusOK {
		t.Fatalf("add group member: expected 200, got %d, body=%s", addMemberRec.Code, addMemberRec.Body.String())
	}

	listMembersRec := doJSONRequest(t, router, http.MethodGet, "/api/admin/groups/"+group.ID.String()+"/members", nil, managerTokens.AccessToken)
	if listMembersRec.Code != http.StatusOK {
		t.Fatalf("list group members: expected 200, got %d, body=%s", listMembersRec.Code, listMembersRec.Body.String())
	}
	members := parseEnvelopeData[[]groupMemberResponse](t, listMembersRec)
	if !hasGroupMember(members, userBMe.ID) {
		t.Fatalf("expected members list to contain userB %s", userBMe.ID)
	}

	removeMemberRec := doJSONRequest(t, router, http.MethodDelete, "/api/admin/groups/"+group.ID.String()+"/members/"+userBMe.ID.String(), nil, managerTokens.AccessToken)
	if removeMemberRec.Code != http.StatusOK {
		t.Fatalf("remove group member: expected 200, got %d, body=%s", removeMemberRec.Code, removeMemberRec.Body.String())
	}

	deleteGroupRec := doJSONRequest(t, router, http.MethodDelete, "/api/admin/groups/"+group.ID.String(), nil, managerTokens.AccessToken)
	if deleteGroupRec.Code != http.StatusOK {
		t.Fatalf("delete group: expected 200, got %d, body=%s", deleteGroupRec.Code, deleteGroupRec.Body.String())
	}

	groupListAfterDeleteRec := doJSONRequest(t, router, http.MethodGet, "/api/admin/groups", nil, managerTokens.AccessToken)
	if groupListAfterDeleteRec.Code != http.StatusOK {
		t.Fatalf("list groups after delete: expected 200, got %d, body=%s", groupListAfterDeleteRec.Code, groupListAfterDeleteRec.Body.String())
	}
	groupListAfterDelete := parseEnvelopeData[[]groupListItemResponse](t, groupListAfterDeleteRec)
	if _, found := findGroup(groupListAfterDelete, group.ID); found {
		t.Fatalf("expected deleted group %s to be absent", group.ID)
	}

	usersRec := doJSONRequest(t, router, http.MethodGet, "/api/admin/users", nil, managerTokens.AccessToken)
	if usersRec.Code != http.StatusOK {
		t.Fatalf("list users: expected 200, got %d, body=%s", usersRec.Code, usersRec.Body.String())
	}
	users := parseEnvelopeData[[]meResponse](t, usersRec)
	userBListItem, ok := findUser(users, userBMe.ID)
	if !ok || userBListItem.Role != "member" {
		t.Fatalf("expected userB to be listed as member, got %+v", userBListItem)
	}

	promoteUserBRec := doJSONRequest(t, router, http.MethodPatch, "/api/admin/users/"+userBMe.ID.String()+"/role", map[string]any{
		"role": "manager",
	}, managerTokens.AccessToken)
	if promoteUserBRec.Code != http.StatusOK {
		t.Fatalf("promote userB: expected 200, got %d, body=%s", promoteUserBRec.Code, promoteUserBRec.Body.String())
	}

	changeSelfRec := doJSONRequest(t, router, http.MethodPatch, "/api/admin/users/"+managerMe.ID.String()+"/role", map[string]any{
		"role": "member",
	}, managerTokens.AccessToken)
	if changeSelfRec.Code != http.StatusBadRequest {
		t.Fatalf("change own role: expected 400, got %d, body=%s", changeSelfRec.Code, changeSelfRec.Body.String())
	}
	if parseErrorCode(t, changeSelfRec) != service.ErrCannotChangeSelf.Error() {
		t.Fatalf("expected cannot_change_own_role error, got %s", parseErrorCode(t, changeSelfRec))
	}

	demoteUserBRec := doJSONRequest(t, router, http.MethodPatch, "/api/admin/users/"+userBMe.ID.String()+"/role", map[string]any{
		"role": "member",
	}, managerTokens.AccessToken)
	if demoteUserBRec.Code != http.StatusOK {
		t.Fatalf("demote userB: expected 200, got %d, body=%s", demoteUserBRec.Code, demoteUserBRec.Body.String())
	}

	userBReloginRec := doJSONRequest(t, router, http.MethodPost, "/api/auth/login", map[string]any{
		"email":    userBEmail,
		"password": userBPassword,
	}, "")
	if userBReloginRec.Code != http.StatusOK {
		t.Fatalf("re-login userB after demotion: expected 200, got %d, body=%s", userBReloginRec.Code, userBReloginRec.Body.String())
	}
	userBMemberTokens := parseAuthEnvelope(t, userBReloginRec)

	userBAdminUsersRec := doJSONRequest(t, router, http.MethodGet, "/api/admin/users", nil, userBMemberTokens.AccessToken)
	if userBAdminUsersRec.Code != http.StatusForbidden {
		t.Fatalf("member list admin users: expected 403, got %d, body=%s", userBAdminUsersRec.Code, userBAdminUsersRec.Body.String())
	}

	userBCreateGroupRec := doJSONRequest(t, router, http.MethodPost, "/api/admin/groups", map[string]any{
		"name":      fmt.Sprintf("无权组-%d", suffix),
		"leader_id": userBMe.ID,
	}, userBMemberTokens.AccessToken)
	if userBCreateGroupRec.Code != http.StatusForbidden {
		t.Fatalf("member create group: expected 403, got %d, body=%s", userBCreateGroupRec.Code, userBCreateGroupRec.Body.String())
	}
}

func fetchMe(t *testing.T, router http.Handler, accessToken string) meResponse {
	t.Helper()

	rec := doJSONRequest(t, router, http.MethodGet, "/api/me", nil, accessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("get /api/me: expected 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	return parseEnvelopeData[meResponse](t, rec)
}

func parseEnvelopeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var env httpEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if !env.Success {
		t.Fatalf("expected success response, got error: %+v", env.Error)
	}

	var data T
	if err := json.Unmarshal(env.Data, &data); err != nil {
		t.Fatalf("unmarshal envelope data: %v", err)
	}

	return data
}

func parseErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()

	var env httpEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal error envelope: %v", err)
	}
	if env.Success || env.Error == nil {
		t.Fatalf("expected error response, got body=%s", rec.Body.String())
	}

	return env.Error.Code
}

func hasNode(nodes []nodeItemResponse, nodeID uuid.UUID) bool {
	for _, node := range nodes {
		if node.ID == nodeID {
			return true
		}
	}
	return false
}

func findGroup(groups []groupListItemResponse, groupID uuid.UUID) (groupListItemResponse, bool) {
	for _, group := range groups {
		if group.ID == groupID {
			return group, true
		}
	}
	return groupListItemResponse{}, false
}

func hasGroupMember(members []groupMemberResponse, userID uuid.UUID) bool {
	for _, member := range members {
		if member.UserID == userID {
			return true
		}
	}
	return false
}

func findUser(users []meResponse, userID uuid.UUID) (meResponse, bool) {
	for _, user := range users {
		if user.ID == userID {
			return user, true
		}
	}
	return meResponse{}, false
}
