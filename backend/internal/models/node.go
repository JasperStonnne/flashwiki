package models

import (
	"time"

	"github.com/google/uuid"
)

type Node struct {
	ID        uuid.UUID  `db:"id"`
	ParentID  *uuid.UUID `db:"parent_id"`
	Kind      string     `db:"kind"` // "folder" | "doc"
	Title     string     `db:"title"`
	OwnerID   uuid.UUID  `db:"owner_id"`
	DeletedAt *time.Time `db:"deleted_at"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type NodePermission struct {
	ID          uuid.UUID `db:"id"`
	NodeID      uuid.UUID `db:"node_id"`
	SubjectType string    `db:"subject_type"` // "user" | "group"
	SubjectID   uuid.UUID `db:"subject_id"`
	Level       string    `db:"level"` // "manage" | "edit" | "readable" | "none"
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

// NodeWithPermission 用于目录树查询结果，附带当前用户的解析权限
type NodeWithPermission struct {
	Node
	Permission  string `json:"permission"`
	HasChildren bool   `json:"has_children"`
}

// AncestorRow 用于 WITH RECURSIVE 查询结果，供权限解析算法使用
type AncestorRow struct {
	NodeID      uuid.UUID  `db:"node_id"`
	Depth       int        `db:"depth"`
	SubjectType *string    `db:"subject_type"` // nullable（LEFT JOIN 无权限行时为 null）
	SubjectID   *uuid.UUID `db:"subject_id"`
	Level       *string    `db:"level"`
}
