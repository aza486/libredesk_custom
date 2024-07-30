package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

const (
	ModelMessages = "messages"

	DispositionInline = "inline"
)

// Media represents an uploaded object.
type Media struct {
	ID          int         `db:"id" json:"id"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
	UUID        string      `db:"uuid" json:"uuid"`
	Filename    string      `db:"filename" json:"filename"`
	ContentType string      `db:"content_type" json:"content_type"`
	Model       null.String `db:"model" json:"-"`
	ModelID     null.String `db:"model_id" json:"-"`
	Size        int         `db:"size" json:"size"`
	Store       string      `db:"store" json:"store"`
	URL         string      `json:"url"`
	ContentID   string      `json:"-"`
	Disposition string      `json:"-"`
	Content     []byte      `json:"-"`
}
