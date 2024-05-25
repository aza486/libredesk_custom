package models

import (
	"encoding/json"
	"net/textproto"
	"time"
)

type Conversation struct {
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
	UUID            string     `db:"uuid" json:"uuid"`
	ClosedAt        *time.Time `db:"closed_at" json:"closed_at"`
	ResolvedAt      *time.Time `db:"resolved_at" json:"resolved_at"`
	ReferenceNumber *string    `db:"reference_number" json:"reference_number"`
	Priority        *string    `db:"priority" json:"priority"`
	Status          *string    `db:"status" json:"status"`

	// Fields not in schema.
	Tags                *json.RawMessage `db:"tags" json:"tags"`
	ContactFirstName    string           `db:"contact_first_name" json:"contact_first_name"`
	ContactLastName     string           `db:"contact_last_name" json:"contact_last_name"`
	ContactEmail        string           `db:"contact_email" json:"contact_email"`
	ConctactPhoneNumber string           `db:"contact_phone_number" json:"contact_phone_number"`
	ContactUUID         string           `db:"contact_uuid" json:"contact_uuid"`
	ContactAvatarURL    *string          `db:"contact_avatar_url" json:"contact_avatar_url"`
	AssignedTeamUUID    *string          `db:"assigned_team_uuid" json:"assigned_team_uuid"`
	AssignedAgentUUID   *string          `db:"assigned_agent_uuid" json:"assigned_agent_uuid"`
	LastMessage         *string          `db:"last_message" json:"last_message,omitempty"`
}

type Message struct {
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
	UUID        string    `db:"uuid" json:"uuid"`
	Type        string    `db:"type" json:"type"`
	Status      string    `db:"status" json:"status"`
	Content     string    `db:"content" json:"content"`
	Attachments []string  `db:"attachments" json:"attachments"`
	FirstName   string    `db:"first_name" json:"first_name"`
	LastName    *string   `db:"last_name" json:"last_name"`
	AvatarURL   *string   `db:"avatar_url" json:"avatar_url"`
}

type IncomingMessage struct {
	From        string
	To          []string
	Subject     string
	Content     string
	AltContent  string
	SourceID    string
	Source      string
	ContentType string
	Headers     textproto.MIMEHeader
	Attachments []Attachment
}

type Attachment struct {
	Name    string
	Header  textproto.MIMEHeader
	Content []byte
}
