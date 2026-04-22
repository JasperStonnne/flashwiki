package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
	"fpgwiki/backend/internal/repository"
)

const (
	minGroupNameLen = 1
	maxGroupNameLen = 50
)

var (
	ErrInvalidName        = errors.New("invalid_name")
	ErrLeaderNotFound     = errors.New("leader_not_found")
	ErrNameTaken          = errors.New("name_taken")
	ErrCannotRemoveLeader = errors.New("cannot_remove_leader")
	ErrMemberNotFound     = errors.New("member_not_found")
)

type GroupService interface {
	CreateGroup(ctx context.Context, name string, leaderID uuid.UUID) (*models.Group, error)
	GetGroup(ctx context.Context, id uuid.UUID) (*models.Group, error)
	ListGroups(ctx context.Context) ([]models.GroupWithDetails, error)
	UpdateGroup(ctx context.Context, id uuid.UUID, name *string, leaderID *uuid.UUID) (*models.Group, error)
	DeleteGroup(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, callerID uuid.UUID, groupID uuid.UUID, userID uuid.UUID) error
	RemoveMember(ctx context.Context, callerID uuid.UUID, groupID uuid.UUID, userID uuid.UUID) error
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]models.GroupMember, error)
}

type groupService struct {
	groupRepo          repository.GroupRepo
	userRepo           repository.UserRepo
	nodePermissionRepo repository.NodePermissionRepo
	pool               *pgxpool.Pool
}

func NewGroupService(
	groupRepo repository.GroupRepo,
	userRepo repository.UserRepo,
	nodePermissionRepo repository.NodePermissionRepo,
	pool *pgxpool.Pool,
) GroupService {
	return &groupService{
		groupRepo:          groupRepo,
		userRepo:           userRepo,
		nodePermissionRepo: nodePermissionRepo,
		pool:               pool,
	}
}

func (s *groupService) CreateGroup(ctx context.Context, name string, leaderID uuid.UUID) (*models.Group, error) {
	validName, err := validateGroupName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	if err := s.ensureUserExists(ctx, leaderID, ErrLeaderNotFound); err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	group := &models.Group{
		Name:     validName,
		LeaderID: leaderID,
	}

	if err := s.groupRepo.CreateGroup(ctx, group); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("failed to create group: %w", ErrNameTaken)
		}
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	return group, nil
}

func (s *groupService) GetGroup(ctx context.Context, id uuid.UUID) (*models.Group, error) {
	group, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("failed to get group: %w", ErrNotFound)
	}

	return group, nil
}

func (s *groupService) ListGroups(ctx context.Context) ([]models.GroupWithDetails, error) {
	groups, err := s.groupRepo.ListGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	return groups, nil
}

func (s *groupService) UpdateGroup(
	ctx context.Context,
	id uuid.UUID,
	name *string,
	leaderID *uuid.UUID,
) (*models.Group, error) {
	group, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("failed to update group: %w", ErrNotFound)
	}

	if name != nil {
		validName, err := validateGroupName(*name)
		if err != nil {
			return nil, fmt.Errorf("failed to update group: %w", err)
		}
		group.Name = validName
	}

	if leaderID != nil {
		if err := s.ensureUserExists(ctx, *leaderID, ErrLeaderNotFound); err != nil {
			return nil, fmt.Errorf("failed to update group: %w", err)
		}

		isMember, err := s.groupRepo.IsMember(ctx, id, *leaderID)
		if err != nil {
			return nil, fmt.Errorf("failed to update group: %w", err)
		}
		if !isMember {
			if err := s.groupRepo.AddMember(ctx, &models.GroupMember{GroupID: id, UserID: *leaderID}); err != nil {
				return nil, fmt.Errorf("failed to update group: %w", err)
			}
		}

		group.LeaderID = *leaderID
	}

	if err := s.groupRepo.UpdateGroup(ctx, group); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("failed to update group: %w", ErrNotFound)
		}
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("failed to update group: %w", ErrNameTaken)
		}
		return nil, fmt.Errorf("failed to update group: %w", err)
	}

	return group, nil
}

func (s *groupService) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.nodePermissionRepo.DeleteByGroupID(ctx, tx, id); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	if err := s.groupRepo.DeleteGroup(ctx, tx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to delete group: %w", ErrNotFound)
		}
		return fmt.Errorf("failed to delete group: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return nil
}

func (s *groupService) AddMember(ctx context.Context, callerID uuid.UUID, groupID uuid.UUID, userID uuid.UUID) error {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}
	if group == nil {
		return fmt.Errorf("failed to add group member: %w", ErrNotFound)
	}

	if err := s.ensureGroupManagerPermission(ctx, callerID, group); err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	if err := s.ensureUserExists(ctx, userID, ErrUserNotFound); err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	if err := s.groupRepo.AddMember(ctx, &models.GroupMember{
		GroupID: groupID,
		UserID:  userID,
	}); err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	return nil
}

func (s *groupService) RemoveMember(ctx context.Context, callerID uuid.UUID, groupID uuid.UUID, userID uuid.UUID) error {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}
	if group == nil {
		return fmt.Errorf("failed to remove group member: %w", ErrNotFound)
	}

	if err := s.ensureGroupManagerPermission(ctx, callerID, group); err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	if userID == group.LeaderID {
		return fmt.Errorf("failed to remove group member: %w", ErrCannotRemoveLeader)
	}

	if err := s.groupRepo.RemoveMember(ctx, groupID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to remove group member: %w", ErrMemberNotFound)
		}
		return fmt.Errorf("failed to remove group member: %w", err)
	}

	return nil
}

func (s *groupService) ListMembers(ctx context.Context, groupID uuid.UUID) ([]models.GroupMember, error) {
	members, err := s.groupRepo.ListMembers(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list group members: %w", err)
	}

	return members, nil
}

func (s *groupService) ensureGroupManagerPermission(
	ctx context.Context,
	callerID uuid.UUID,
	group *models.Group,
) error {
	caller, err := s.userRepo.FindByID(ctx, callerID)
	if err != nil {
		return err
	}
	if caller == nil {
		return ErrForbidden
	}
	if caller.Role == "manager" || group.LeaderID == callerID {
		return nil
	}

	return ErrForbidden
}

func (s *groupService) ensureUserExists(ctx context.Context, userID uuid.UUID, notFoundErr error) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return notFoundErr
	}

	return nil
}

func validateGroupName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	nameLen := utf8.RuneCountInString(trimmed)
	if nameLen < minGroupNameLen || nameLen > maxGroupNameLen {
		return "", ErrInvalidName
	}

	return trimmed, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	return pgErr.Code == "23505"
}
