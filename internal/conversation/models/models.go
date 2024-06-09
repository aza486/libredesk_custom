package models

import (
	"encoding/json"
	"time"

	"github.com/abhinavxd/artemis/internal/contact/models"
)

type Conversation struct {
	ID                 int64      `db:"id" json:"-"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at" json:"updated_at"`
	UUID               string     `db:"uuid" json:"uuid"`
	ClosedAt           *time.Time `db:"closed_at" json:"closed_at,omitempty"`
	ResolvedAt         *time.Time `db:"resolved_at" json:"resolved_at,omitempty"`
	ReferenceNumber    *string    `db:"reference_number" json:"reference_number,omitempty"`
	Priority           *string    `db:"priority" json:"priority"`
	Status             *string    `db:"status" json:"status"`
	AssigneeLastSeenAt *time.Time `db:"assignee_last_seen_at" json:"assignee_last_seen_at"`

	models.Contact

	// Fields not in schema.
	UnreadMessageCount int              `db:"unread_message_count" json:"unread_message_count"`
	InboxName          string           `db:"inbox_name" json:"inbox_name"`
	InboxChannel       string           `db:"inbox_channel" json:"inbox_channel"`
	Tags               *json.RawMessage `db:"tags" json:"tags"`
	ContactAvatarURL   *string          `db:"contact_avatar_url" json:"contact_avatar_url"`
	AssignedTeamUUID   *string          `db:"assigned_team_uuid" json:"assigned_team_uuid"`
	AssignedAgentUUID  *string          `db:"assigned_user_uuid" json:"assigned_user_uuid"`
	LastMessageAt      *time.Time       `db:"last_message_at" json:"last_message_at,omitempty"`
	LastMessage        string           `db:"last_message" json:"last_message,omitempty"`
}

type ConversationParticipant struct {
	UUID      string  `db:"uuid" json:"uuid"`
	FirstName string  `db:"first_name" json:"first_name"`
	LastName  string  `db:"last_name" json:"last_name"`
	AvatarURL *string `db:"avatar_url" json:"avatar_url"`
}
