package models

import "encoding/json"

type Filter struct {
	ID      int             `db:"id" json:"id"`
	UserID  int             `db:"user_id" json:"user_id"`
	Page    string          `db:"page" json:"page"`
	Filters json.RawMessage `db:"filters" json:"filters"`
}
