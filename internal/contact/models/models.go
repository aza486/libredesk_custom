package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

type Contact struct {
	ID          int         `db:"id" json:"id,omitempty"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at,omitempty"`
	FirstName   string      `db:"first_name" json:"first_name"`
	LastName    string      `db:"last_name" json:"last_name"`
	Email       string      `db:"email" json:"email"`
	PhoneNumber null.String `db:"phone_number" json:"phone_number"`
	AvatarURL   null.String `db:"avatar_url" json:"avatar_url"`
	InboxID     int         `db:"inbox_id" json:"-"`
	Source      string      `db:"source" json:"-"`
	SourceID    string      `db:"source_id" json:"-"`
}
