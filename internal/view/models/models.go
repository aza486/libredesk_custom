package models

import (
	"encoding/json"
	"time"
)

type View struct {
	ID        int             `db:"id" json:"id"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
	Name      string          `db:"name" json:"name"`
	InboxType string          `db:"inbox_type" json:"inbox_type"`
	Filters   json.RawMessage `db:"filters" json:"filters"`
	UserID    int             `db:"user_id" json:"user_id"`
}
