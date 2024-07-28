package models

import (
	"time"
)

const (
	ModelMessages = "messages"

	DispositionInline = "inline"
)

// Media represents an uploaded object.
type Media struct {
	ID          int       `db:"id" json:"id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UUID        string    `db:"uuid" json:"uuid"`
	Filename    string    `db:"filename" json:"filename"`
	ContentType string    `db:"content_type" json:"content_type"`
	Model       string    `db:"model" json:"model"`
	ModelID     string    `db:"model_id" json:"model_id"`
	Size        int       `db:"size" json:"size"`
	Store       string    `db:"store" json:"store"`
	URL         string    `json:"url"`
	ContentID   string    `json:"-"`
	Disposition string    `json:"-"`
	Content     []byte    `json:"-"`
}
