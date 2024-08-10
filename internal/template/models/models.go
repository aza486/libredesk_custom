package models

import "time"

type Template struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Name      string    `db:"name" json:"name"`
	Body      string    `db:"body" json:"body"`
	IsDefault bool      `db:"is_default" json:"is_default"`
}
