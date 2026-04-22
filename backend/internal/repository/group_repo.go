package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fpgwiki/backend/internal/models"
)

type GroupRepo interface {
	CreateGroup(ctx context.Context, group *models.Group) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Group, error)
	UpdateGroup(ctx context.Context, group *models.Group) error
	DeleteGroup(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	ListGroups(ctx context.Context) ([]models.GroupWithDetails, error)
	AddMember(ctx context.Context, member *models.GroupMember) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
	GetGroupIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	ListMembers(ctx context.Context, groupID uuid.UUID) ([]models.GroupMember, error)
	IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

type groupRepo struct {
	pool *pgxpool.Pool
}

func NewGroupRepo(pool *pgxpool.Pool) GroupRepo {
	return &groupRepo{pool: pool}
}

func (r *groupRepo) CreateGroup(ctx context.Context, group *models.Group) error {
	if group == nil {
		return fmt.Errorf("failed to create group: %w", errors.New("group is nil"))
	}
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	defer tx.Rollback(ctx)

	const createGroupQuery = `
INSERT INTO groups (id, name, leader_id)
VALUES ($1, $2, $3)
RETURNING created_at, updated_at
`
	if err := tx.QueryRow(ctx, createGroupQuery, group.ID, group.Name, group.LeaderID).Scan(
		&group.CreatedAt,
		&group.UpdatedAt,
	); err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	const addLeaderQuery = `
INSERT INTO group_members (group_id, user_id)
VALUES ($1, $2)
`
	if _, err := tx.Exec(ctx, addLeaderQuery, group.ID, group.LeaderID); err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}

	return nil
}

func (r *groupRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Group, error) {
	const query = `
SELECT id, name, leader_id, created_at, updated_at
FROM groups
WHERE id = $1
`

	group := &models.Group{}
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&group.ID,
		&group.Name,
		&group.LeaderID,
		&group.CreatedAt,
		&group.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find group by id: %w", err)
	}

	return group, nil
}

func (r *groupRepo) UpdateGroup(ctx context.Context, group *models.Group) error {
	if group == nil {
		return fmt.Errorf("failed to update group: %w", errors.New("group is nil"))
	}

	const query = `
UPDATE groups
SET name = $2, leader_id = $3, updated_at = now()
WHERE id = $1
RETURNING updated_at
`

	if err := r.pool.QueryRow(ctx, query, group.ID, group.Name, group.LeaderID).Scan(&group.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("failed to update group: %w", pgx.ErrNoRows)
		}
		return fmt.Errorf("failed to update group: %w", err)
	}

	return nil
}

func (r *groupRepo) DeleteGroup(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	const query = `
DELETE FROM groups
WHERE id = $1
`

	tag, err := tx.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to delete group: %w", pgx.ErrNoRows)
	}

	return nil
}

func (r *groupRepo) ListGroups(ctx context.Context) ([]models.GroupWithDetails, error) {
	const query = `
SELECT
  g.id,
  g.name,
  g.leader_id,
  g.created_at,
  g.updated_at,
  u.display_name AS leader_name,
  u.email AS leader_email,
  (
    SELECT COUNT(*)
    FROM group_members gm
    WHERE gm.group_id = g.id
  ) AS member_count
FROM groups g
JOIN users u ON u.id = g.leader_id
ORDER BY g.name ASC
`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}
	defer rows.Close()

	groups := make([]models.GroupWithDetails, 0)
	for rows.Next() {
		var group models.GroupWithDetails
		if err := rows.Scan(
			&group.ID,
			&group.Name,
			&group.LeaderID,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.LeaderName,
			&group.LeaderEmail,
			&group.MemberCount,
		); err != nil {
			return nil, fmt.Errorf("failed to list groups: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	return groups, nil
}

func (r *groupRepo) AddMember(ctx context.Context, member *models.GroupMember) error {
	if member == nil {
		return fmt.Errorf("failed to add group member: %w", errors.New("member is nil"))
	}

	const query = `
INSERT INTO group_members (group_id, user_id)
VALUES ($1, $2)
ON CONFLICT (group_id, user_id) DO NOTHING
`

	if _, err := r.pool.Exec(ctx, query, member.GroupID, member.UserID); err != nil {
		return fmt.Errorf("failed to add group member: %w", err)
	}

	return nil
}

func (r *groupRepo) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	const query = `
DELETE FROM group_members
WHERE group_id = $1 AND user_id = $2
`

	tag, err := r.pool.Exec(ctx, query, groupID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove group member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("failed to remove group member: %w", pgx.ErrNoRows)
	}

	return nil
}

func (r *groupRepo) GetGroupIDsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const query = `
SELECT group_id
FROM group_members
WHERE user_id = $1
`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group ids by user: %w", err)
	}
	defer rows.Close()

	groupIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var groupID uuid.UUID
		if err := rows.Scan(&groupID); err != nil {
			return nil, fmt.Errorf("failed to get group ids by user: %w", err)
		}
		groupIDs = append(groupIDs, groupID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to get group ids by user: %w", err)
	}

	return groupIDs, nil
}

func (r *groupRepo) ListMembers(ctx context.Context, groupID uuid.UUID) ([]models.GroupMember, error) {
	const query = `
SELECT group_id, user_id, joined_at
FROM group_members
WHERE group_id = $1
ORDER BY joined_at
`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list group members: %w", err)
	}
	defer rows.Close()

	members := make([]models.GroupMember, 0)
	for rows.Next() {
		var member models.GroupMember
		if err := rows.Scan(&member.GroupID, &member.UserID, &member.JoinedAt); err != nil {
			return nil, fmt.Errorf("failed to list group members: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list group members: %w", err)
	}

	return members, nil
}

func (r *groupRepo) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	const query = `
SELECT EXISTS(
  SELECT 1
  FROM group_members
  WHERE group_id = $1 AND user_id = $2
)
`

	var isMember bool
	if err := r.pool.QueryRow(ctx, query, groupID, userID).Scan(&isMember); err != nil {
		return false, fmt.Errorf("failed to check group membership: %w", err)
	}

	return isMember, nil
}
