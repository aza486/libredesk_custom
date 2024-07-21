package models

import "encoding/json"

// Inbox represents a inbox record in DB.
type Inbox struct {
	ID       int             `db:"id" json:"id"`
	Name     string          `db:"name" json:"name"`
	Channel  string          `db:"channel" json:"channel"`
	Disabled bool            `db:"disabled" json:"disabled"`
	From     string          `db:"from" json:"from"`
	Config   json.RawMessage `db:"config" json:"config"`
}
