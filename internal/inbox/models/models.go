package models

import (
	"encoding/json"
	"time"
)

// Inbox represents a inbox record in DB.
type Inbox struct {
	ID        int             `db:"id" json:"id"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"`
	Name      string          `db:"name" json:"name"`
	Channel   string          `db:"channel" json:"channel"`
	Disabled  bool            `db:"disabled" json:"disabled"`
	From      string          `db:"from" json:"from"`
	Config    json.RawMessage `db:"config" json:"config"`
}
