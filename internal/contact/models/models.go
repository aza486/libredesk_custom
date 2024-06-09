package models

import "time"

type Contact struct {
	ID          int64     `db:"id" json:"id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	FirstName   string    `db:"first_name" json:"first_name"`
	LastName    string    `db:"last_name" json:"last_name"`
	Email       string    `db:"email" json:"email"`
	PhoneNumber *string   `db:"phone_number" json:"phone_number"`
	AvatarURL   *string   `db:"avatar_url" json:"avatar_url"`
	InboxID     int       `db:"inbox_id" json:"inbox_id"`
	Source      string    `db:"source" json:"source"`
	SourceID    string    `db:"source_id" json:"source_id"`
}
