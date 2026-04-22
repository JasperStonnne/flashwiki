package models

import (
	"time"

	"github.com/google/uuid"
)

type Group struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	LeaderID  uuid.UUID `db:"leader_id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type GroupMember struct {
	GroupID  uuid.UUID `db:"group_id"`
	UserID   uuid.UUID `db:"user_id"`
	JoinedAt time.Time `db:"joined_at"`
}

// GroupWithDetails 用于管理后台列表，附带组长信息和成员数
type GroupWithDetails struct {
	Group
	LeaderName  string `json:"leader_name"`
	LeaderEmail string `json:"leader_email"`
	MemberCount int    `json:"member_count"`
}
