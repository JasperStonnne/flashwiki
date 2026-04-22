package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

const (
	nodeKindFolder = "folder"
	nodeKindDoc    = "doc"

	maxNodeTitleLen = 200
	restoreWindow   = 30 * 24 * time.Hour
)

var (
	ErrNotFound        = errors.New("not_found")
	ErrForbidden       = errors.New("forbidden")
	ErrInvalidInput    = errors.New("invalid_input")
	ErrParentNotFound  = errors.New("parent_not_found")
	ErrParentNotFolder = errors.New("parent_not_folder")
	ErrInvalidKind     = errors.New("invalid_kind")
	ErrTitleTooLong    = errors.New("title_too_long")
	ErrCircularMove    = errors.New("circular_move")
	ErrMoveToSelf      = errors.New("move_to_self")
	ErrNotDeleted      = errors.New("not_deleted")
	ErrRestoreExpired  = errors.New("restore_expired")
	ErrParentDeleted   = errors.New("parent_deleted")
)

type NodeService interface {
	CreateNode(ctx context.Context, userID uuid.UUID, req CreateNodeRequest) (*models.Node, error)
	GetNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) (*models.Node, string, error)
	ListChildren(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]models.NodeWithPermission, error)
	RenameNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, newTitle string) (*models.Node, error)
	MoveNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID, newParentID *uuid.UUID) (*models.Node, error)
	SoftDeleteNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) error
	RestoreNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) error
}

type CreateNodeRequest struct {
	ParentID *uuid.UUID
	Kind     string
	Title    string
}

type nodeService struct {
	nodeRepo          repository.NodeRepo
	userRepo          repository.UserRepo
	permissionService PermissionService
	pool              *pgxpool.Pool
	now               func() time.Time
}

func NewNodeService(
	nodeRepo repository.NodeRepo,
	userRepo repository.UserRepo,
	permissionService PermissionService,
	pool *pgxpool.Pool,
) NodeService {
	return &nodeService{
		nodeRepo:          nodeRepo,
		userRepo:          userRepo,
		permissionService: permissionService,
		pool:              pool,
		now:               time.Now,
	}
}

func (s *nodeService) CreateNode(ctx context.Context, userID uuid.UUID, req CreateNodeRequest) (*models.Node, error) {
	title, err := validateNodeTitle(req.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}
	if err := validateNodeKind(req.Kind); err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	if req.ParentID == nil {
		if err := s.ensureManager(ctx, userID); err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
	} else {
		parent, err := s.requireActiveNode(ctx, *req.ParentID, ErrParentNotFound)
		if err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
		if parent.Kind != nodeKindFolder {
			return nil, fmt.Errorf("failed to create node: %w", ErrParentNotFolder)
		}
		if err := s.requirePermissionAtLeast(ctx, userID, parent.ID, permissionLevelEdit); err != nil {
			return nil, fmt.Errorf("failed to create node: %w", err)
		}
	}

	node := &models.Node{
		ParentID: req.ParentID,
		Kind:     req.Kind,
		Title:    title,
		OwnerID:  userID,
	}
	if err := s.nodeRepo.Create(ctx, node); err != nil {
		return nil, fmt.Errorf("failed to create node: %w", err)
	}

	return node, nil
}

func (s *nodeService) GetNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) (*models.Node, string, error) {
	node, err := s.requireActiveNode(ctx, nodeID, ErrNotFound)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get node: %w", err)
	}

	level, err := s.permissionService.ResolvePermission(ctx, userID, nodeID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get node: %w", err)
	}
	if level == permissionLevelNone {
		return nil, "", fmt.Errorf("failed to get node: %w", ErrNotFound)
	}

	return node, level, nil
}

func (s *nodeService) ListChildren(
	ctx context.Context,
	userID uuid.UUID,
	parentID *uuid.UUID,
) ([]models.NodeWithPermission, error) {
	nodes, err := s.nodeRepo.ListByParent(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list node children: %w", err)
	}

	result := make([]models.NodeWithPermission, 0, len(nodes))
	for _, node := range nodes {
		level, err := s.permissionService.ResolvePermission(ctx, userID, node.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list node children: %w", err)
		}
		if level == permissionLevelNone {
			continue
		}

		hasChildren, err := s.nodeRepo.HasChildren(ctx, node.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list node children: %w", err)
		}

		result = append(result, models.NodeWithPermission{
			Node:        *node,
			Permission:  level,
			HasChildren: hasChildren,
		})
	}

	return result, nil
}

func (s *nodeService) RenameNode(
	ctx context.Context,
	userID uuid.UUID,
	nodeID uuid.UUID,
	newTitle string,
) (*models.Node, error) {
	node, err := s.requireActiveNode(ctx, nodeID, ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("failed to rename node: %w", err)
	}
	if err := s.requirePermissionAtLeast(ctx, userID, nodeID, permissionLevelEdit); err != nil {
		return nil, fmt.Errorf("failed to rename node: %w", err)
	}

	title, err := validateNodeTitle(newTitle)
	if err != nil {
		return nil, fmt.Errorf("failed to rename node: %w", err)
	}

	node.Title = title
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to rename node: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to rename node: %w", err)
	}

	return node, nil
}

func (s *nodeService) MoveNode(
	ctx context.Context,
	userID uuid.UUID,
	nodeID uuid.UUID,
	newParentID *uuid.UUID,
) (*models.Node, error) {
	node, err := s.requireActiveNode(ctx, nodeID, ErrNotFound)
	if err != nil {
		return nil, fmt.Errorf("failed to move node: %w", err)
	}
	if err := s.requirePermissionAtLeast(ctx, userID, nodeID, permissionLevelManage); err != nil {
		return nil, fmt.Errorf("failed to move node: %w", err)
	}

	if newParentID != nil {
		if *newParentID == nodeID {
			return nil, fmt.Errorf("failed to move node: %w", ErrMoveToSelf)
		}
		if err := s.validateMoveTarget(ctx, userID, nodeID, *newParentID); err != nil {
			return nil, fmt.Errorf("failed to move node: %w", err)
		}
	}

	node.ParentID = newParentID
	if err := s.nodeRepo.Update(ctx, node); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to move node: %w", ErrNotFound)
		}
		return nil, fmt.Errorf("failed to move node: %w", err)
	}

	return node, nil
}

func (s *nodeService) SoftDeleteNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) error {
	if _, err := s.requireActiveNode(ctx, nodeID, ErrNotFound); err != nil {
		return fmt.Errorf("failed to soft delete node: %w", err)
	}
	if err := s.requirePermissionAtLeast(ctx, userID, nodeID, permissionLevelManage); err != nil {
		return fmt.Errorf("failed to soft delete node: %w", err)
	}

	if err := s.nodeRepo.SoftDelete(ctx, nodeID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to soft delete node: %w", ErrNotFound)
		}
		return fmt.Errorf("failed to soft delete node: %w", err)
	}

	return nil
}

func (s *nodeService) RestoreNode(ctx context.Context, userID uuid.UUID, nodeID uuid.UUID) error {
	node, err := s.nodeRepo.FindByID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("failed to restore node: %w", err)
	}
	if node == nil {
		return fmt.Errorf("failed to restore node: %w", ErrNotFound)
	}
	if node.DeletedAt == nil {
		return fmt.Errorf("failed to restore node: %w", ErrNotDeleted)
	}
	if s.now().UTC().Sub(node.DeletedAt.UTC()) > restoreWindow {
		return fmt.Errorf("failed to restore node: %w", ErrRestoreExpired)
	}

	if err := s.ensureParentActiveForRestore(ctx, node.ParentID); err != nil {
		return fmt.Errorf("failed to restore node: %w", err)
	}
	if err := s.ensureRestoreActorAllowed(ctx, userID, node.OwnerID); err != nil {
		return fmt.Errorf("failed to restore node: %w", err)
	}

	if err := s.nodeRepo.Restore(ctx, nodeID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to restore node: %w", ErrNotFound)
		}
		return fmt.Errorf("failed to restore node: %w", err)
	}

	return nil
}

func (s *nodeService) ensureManager(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.Role != "manager" {
		return ErrForbidden
	}

	return nil
}

func (s *nodeService) requireActiveNode(ctx context.Context, nodeID uuid.UUID, notFoundErr error) (*models.Node, error) {
	node, err := s.nodeRepo.FindByID(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if node == nil || node.DeletedAt != nil {
		return nil, notFoundErr
	}

	return node, nil
}

func (s *nodeService) requirePermissionAtLeast(
	ctx context.Context,
	userID uuid.UUID,
	nodeID uuid.UUID,
	required string,
) error {
	level, err := s.permissionService.ResolvePermission(ctx, userID, nodeID)
	if err != nil {
		return err
	}
	if !permissionAtLeast(level, required) {
		return ErrForbidden
	}

	return nil
}

func (s *nodeService) validateMoveTarget(
	ctx context.Context,
	userID uuid.UUID,
	nodeID uuid.UUID,
	newParentID uuid.UUID,
) error {
	parent, err := s.requireActiveNode(ctx, newParentID, ErrParentNotFound)
	if err != nil {
		return err
	}
	if parent.Kind != nodeKindFolder {
		return ErrParentNotFolder
	}
	if err := s.requirePermissionAtLeast(ctx, userID, newParentID, permissionLevelEdit); err != nil {
		return err
	}

	descendantIDs, err := s.nodeRepo.GetDescendantIDs(ctx, nodeID)
	if err != nil {
		return err
	}
	for _, descendantID := range descendantIDs {
		if descendantID == newParentID {
			return ErrCircularMove
		}
	}

	return nil
}

func (s *nodeService) ensureParentActiveForRestore(ctx context.Context, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}

	parent, err := s.nodeRepo.FindByID(ctx, *parentID)
	if err != nil {
		return err
	}
	if parent == nil || parent.DeletedAt != nil {
		return ErrParentDeleted
	}

	return nil
}

func (s *nodeService) ensureRestoreActorAllowed(ctx context.Context, userID, ownerID uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	// 已删除节点的继承链可能断裂，因此恢复权限简化为 Manager 或节点 owner。
	if user != nil && user.Role == "manager" {
		return nil
	}
	if ownerID == userID {
		return nil
	}

	return ErrForbidden
}

func validateNodeKind(kind string) error {
	if kind != nodeKindFolder && kind != nodeKindDoc {
		return ErrInvalidKind
	}

	return nil
}

func validateNodeTitle(title string) (string, error) {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return "", ErrInvalidInput
	}
	if utf8.RuneCountInString(trimmed) > maxNodeTitleLen {
		return "", ErrTitleTooLong
	}

	return trimmed, nil
}

func permissionAtLeast(level, required string) bool {
	levelRank, ok := permissionLevelRank[level]
	if !ok {
		return false
	}
	requiredRank, ok := permissionLevelRank[required]
	if !ok {
		return false
	}

	return levelRank >= requiredRank
}
