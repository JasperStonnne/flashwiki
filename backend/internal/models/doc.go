package models

import (
	"time"

	"github.com/google/uuid"
)

// DocState stores the latest compacted complete Y.Doc binary state for a document.
type DocState struct {
	NodeID          uuid.UUID `db:"node_id"`
	YDocState       []byte    `db:"ydoc_state"`
	Version         int64     `db:"version"`
	MarkdownPlain   string    `db:"markdown_plain"`
	LastCompactedAt time.Time `db:"last_compacted_at"`
}

// DocUpdate stores one pending Y.js binary update before compaction.
type DocUpdate struct {
	ID        int64     `db:"id"`
	NodeID    uuid.UUID `db:"node_id"`
	Update    []byte    `db:"update"`
	ClientID  string    `db:"client_id"`
	CreatedAt time.Time `db:"created_at"`
}
