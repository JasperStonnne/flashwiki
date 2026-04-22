package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

const (
	permissionLevelNone     = "none"
	permissionLevelReadable = "readable"
	permissionLevelEdit     = "edit"
	permissionLevelManage   = "manage"
	subjectTypeUser         = "user"
	subjectTypeGroup        = "group"
)

var permissionLevelRank = map[string]int{
	permissionLevelNone:     0,
	permissionLevelReadable: 1,
	permissionLevelEdit:     2,
	permissionLevelManage:   3,
}

type PermissionService interface {
	ResolvePermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) (string, error)
	SetNodePermissions(ctx context.Context, nodeID uuid.UUID, perms []SetPermissionEntry) error
	GetNodePermissions(ctx context.Context, nodeID uuid.UUID) (*NodePermissionResult, error)
}

type SetPermissionEntry struct {
	SubjectType string
	SubjectID   uuid.UUID
	Level       string
}

type NodePermissionResult struct {
	NodeID        uuid.UUID
	Permissions   []models.NodePermission
	InheritedFrom *uuid.UUID
}

type permissionService struct {
	userRepo           repository.UserRepo
	nodeRepo           repository.NodeRepo
	nodePermissionRepo repository.NodePermissionRepo
	groupRepo          repository.GroupRepo
	pool               *pgxpool.Pool
}

type permissionSubjectKey struct {
	subjectType string
	subjectID   uuid.UUID
}

func NewPermissionService(
	userRepo repository.UserRepo,
	nodeRepo repository.NodeRepo,
	nodePermissionRepo repository.NodePermissionRepo,
	groupRepo repository.GroupRepo,
	pool *pgxpool.Pool,
) PermissionService {
	return &permissionService{
		userRepo:           userRepo,
		nodeRepo:           nodeRepo,
		nodePermissionRepo: nodePermissionRepo,
		groupRepo:          groupRepo,
		pool:               pool,
	}
}

func (s *permissionService) ResolvePermission(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) (string, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve permission: %w", err)
	}
	if user == nil {
		return permissionLevelNone, nil
	}
	if user.Role == "manager" {
		return permissionLevelManage, nil
	}

	ancestors, err := s.nodeRepo.GetAncestorsWithPermissions(ctx, nodeID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve permission: %w", err)
	}

	_, effectiveRows := findEffectiveNodeRows(ancestors)
	if len(effectiveRows) == 0 {
		return permissionLevelNone, nil
	}

	userLevel, matched, err := resolveUserPermission(effectiveRows, userID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve permission: %w", err)
	}
	if matched {
		return userLevel, nil
	}

	groupIDs, err := s.groupRepo.GetGroupIDsByUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve permission: %w", err)
	}

	groupLevel, matched, err := resolveGroupPermission(effectiveRows, groupIDs)
	if err != nil {
		return "", fmt.Errorf("failed to resolve permission: %w", err)
	}
	if matched {
		return groupLevel, nil
	}

	return permissionLevelNone, nil
}

func (s *permissionService) SetNodePermissions(ctx context.Context, nodeID uuid.UUID, perms []SetPermissionEntry) error {
	if err := s.validateSetPermissionEntries(ctx, perms); err != nil {
		return fmt.Errorf("failed to set node permissions: %w", err)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to set node permissions: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := lockNodeForPermissionUpdate(ctx, tx, nodeID); err != nil {
		return fmt.Errorf("failed to set node permissions: %w", err)
	}

	modelPerms := toNodePermissions(nodeID, perms)
	if err := s.nodePermissionRepo.ReplacePermissions(ctx, tx, nodeID, modelPerms); err != nil {
		return fmt.Errorf("failed to set node permissions: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to set node permissions: %w", err)
	}

	return nil
}

func (s *permissionService) GetNodePermissions(ctx context.Context, nodeID uuid.UUID) (*NodePermissionResult, error) {
	perms, err := s.nodePermissionRepo.ListByNode(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node permissions: %w", err)
	}

	result := &NodePermissionResult{
		NodeID:      nodeID,
		Permissions: perms,
	}
	if len(perms) > 0 {
		return result, nil
	}

	ancestors, err := s.nodeRepo.GetAncestorsWithPermissions(ctx, nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get node permissions: %w", err)
	}

	effectiveNodeID, _ := findEffectiveNodeRows(ancestors)
	result.InheritedFrom = effectiveNodeID

	return result, nil
}

func (s *permissionService) validateSetPermissionEntries(ctx context.Context, perms []SetPermissionEntry) error {
	seenSubjects := make(map[permissionSubjectKey]struct{}, len(perms))
	seenUsers := make(map[uuid.UUID]struct{})
	seenGroups := make(map[uuid.UUID]struct{})

	for _, perm := range perms {
		if !isValidSubjectType(perm.SubjectType) {
			return fmt.Errorf("validation failed: invalid subject_type %q", perm.SubjectType)
		}
		if !isValidPermissionLevel(perm.Level) {
			return fmt.Errorf("validation failed: invalid level %q", perm.Level)
		}

		key := permissionSubjectKey{
			subjectType: perm.SubjectType,
			subjectID:   perm.SubjectID,
		}
		if _, exists := seenSubjects[key]; exists {
			return errors.New("validation failed: duplicate subject entry")
		}
		seenSubjects[key] = struct{}{}

		if err := s.ensureSubjectExists(ctx, perm, seenUsers, seenGroups); err != nil {
			return err
		}
	}

	return nil
}

func (s *permissionService) ensureSubjectExists(
	ctx context.Context,
	perm SetPermissionEntry,
	seenUsers map[uuid.UUID]struct{},
	seenGroups map[uuid.UUID]struct{},
) error {
	if perm.SubjectType == subjectTypeUser {
		if _, exists := seenUsers[perm.SubjectID]; exists {
			return nil
		}

		user, err := s.userRepo.FindByID(ctx, perm.SubjectID)
		if err != nil {
			return fmt.Errorf("failed to find user by id: %w", err)
		}
		if user == nil {
			return fmt.Errorf("validation failed: user %s not found", perm.SubjectID)
		}

		seenUsers[perm.SubjectID] = struct{}{}
		return nil
	}

	if _, exists := seenGroups[perm.SubjectID]; exists {
		return nil
	}

	group, err := s.groupRepo.FindByID(ctx, perm.SubjectID)
	if err != nil {
		return fmt.Errorf("failed to find group by id: %w", err)
	}
	if group == nil {
		return fmt.Errorf("validation failed: group %s not found", perm.SubjectID)
	}

	seenGroups[perm.SubjectID] = struct{}{}
	return nil
}

func lockNodeForPermissionUpdate(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID) error {
	const query = `
SELECT id
FROM nodes
WHERE id = $1
FOR UPDATE
`

	var lockedNodeID uuid.UUID
	if err := tx.QueryRow(ctx, query, nodeID).Scan(&lockedNodeID); err != nil {
		return err
	}

	return nil
}

func toNodePermissions(nodeID uuid.UUID, perms []SetPermissionEntry) []models.NodePermission {
	result := make([]models.NodePermission, 0, len(perms))
	for _, perm := range perms {
		result = append(result, models.NodePermission{
			NodeID:      nodeID,
			SubjectType: perm.SubjectType,
			SubjectID:   perm.SubjectID,
			Level:       perm.Level,
		})
	}

	return result
}

func resolveUserPermission(rows []models.AncestorRow, userID uuid.UUID) (string, bool, error) {
	return resolveMatchedPermissionLevel(rows, func(row models.AncestorRow) bool {
		return row.SubjectType != nil &&
			*row.SubjectType == subjectTypeUser &&
			row.SubjectID != nil &&
			*row.SubjectID == userID
	})
}

func resolveGroupPermission(rows []models.AncestorRow, groupIDs []uuid.UUID) (string, bool, error) {
	groupSet := make(map[uuid.UUID]struct{}, len(groupIDs))
	for _, groupID := range groupIDs {
		groupSet[groupID] = struct{}{}
	}

	return resolveMatchedPermissionLevel(rows, func(row models.AncestorRow) bool {
		if row.SubjectType == nil || *row.SubjectType != subjectTypeGroup || row.SubjectID == nil {
			return false
		}
		_, ok := groupSet[*row.SubjectID]
		return ok
	})
}

func resolveMatchedPermissionLevel(
	rows []models.AncestorRow,
	match func(row models.AncestorRow) bool,
) (string, bool, error) {
	maxRank := -1
	maxLevel := ""
	matched := false

	for _, row := range rows {
		if !match(row) {
			continue
		}
		if row.Level == nil {
			continue
		}

		level := *row.Level
		if level == permissionLevelNone {
			return permissionLevelNone, true, nil
		}

		rank, ok := permissionLevelRank[level]
		if !ok {
			return "", false, fmt.Errorf("validation failed: invalid level %q", level)
		}

		matched = true
		if rank > maxRank {
			maxRank = rank
			maxLevel = level
		}
	}

	if !matched {
		return "", false, nil
	}

	return maxLevel, true, nil
}

func findEffectiveNodeRows(ancestors []models.AncestorRow) (*uuid.UUID, []models.AncestorRow) {
	rowsByDepth := make(map[int][]models.AncestorRow)
	nodeByDepth := make(map[int]uuid.UUID)
	orderedDepths := make([]int, 0)
	seenDepth := make(map[int]struct{})

	for _, row := range ancestors {
		if _, ok := seenDepth[row.Depth]; !ok {
			seenDepth[row.Depth] = struct{}{}
			orderedDepths = append(orderedDepths, row.Depth)
			nodeByDepth[row.Depth] = row.NodeID
		}

		if row.SubjectType != nil {
			rowsByDepth[row.Depth] = append(rowsByDepth[row.Depth], row)
		}
	}

	for _, depth := range orderedDepths {
		rows := rowsByDepth[depth]
		if len(rows) == 0 {
			continue
		}

		nodeID := nodeByDepth[depth]
		return &nodeID, rows
	}

	return nil, nil
}

func isValidSubjectType(subjectType string) bool {
	return subjectType == subjectTypeUser || subjectType == subjectTypeGroup
}

func isValidPermissionLevel(level string) bool {
	_, ok := permissionLevelRank[level]
	return ok
}
