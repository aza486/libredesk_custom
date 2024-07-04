package models

import (
	"encoding/json"
	"time"

	"github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/volatiletech/null/v9"
)

type Conversation struct {
	ID                 int         `db:"id" json:"-"`
	CreatedAt          time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at" json:"updated_at"`
	UUID               string      `db:"uuid" json:"uuid"`
	ClosedAt           null.Time   `db:"closed_at" json:"closed_at,omitempty"`
	ResolvedAt         null.Time   `db:"resolved_at" json:"resolved_at,omitempty"`
	ReferenceNumber    null.String `db:"reference_number" json:"reference_number,omitempty"`
	Priority           null.String `db:"priority" json:"priority"`
	Status             null.String `db:"status" json:"status"`
	FirstReplyAt       null.Time   `db:"first_reply_at" json:"first_reply_at"`
	AssignedUserID     null.Int    `db:"assigned_user_id" json:"-"`
	AssignedTeamID     null.Int    `db:"assigned_team_id" json:"-"`
	AssigneeLastSeenAt *time.Time  `db:"assignee_last_seen_at" json:"assignee_last_seen_at"`
	models.Contact
	// Psuedo fields.
	Subject            string           `db:"subject" json:"subject"`
	UnreadMessageCount int              `db:"unread_message_count" json:"unread_message_count"`
	InboxName          string           `db:"inbox_name" json:"inbox_name"`
	InboxChannel       string           `db:"inbox_channel" json:"inbox_channel"`
	Tags               *json.RawMessage `db:"tags" json:"tags"`
	ContactAvatarURL   *string          `db:"contact_avatar_url" json:"contact_avatar_url"`
	AssignedTeamUUID   *string          `db:"assigned_team_uuid" json:"assigned_team_uuid"`
	AssignedAgentUUID  *string          `db:"assigned_user_uuid" json:"assigned_user_uuid"`
	LastMessageAt      null.Time        `db:"last_message_at" json:"last_message_at"`
	LastMessage        string           `db:"last_message" json:"last_message"`
	FirstMessage       string           `json:"-"`
}

type ConversationParticipant struct {
	UUID      string  `db:"uuid" json:"uuid"`
	FirstName string  `db:"first_name" json:"first_name"`
	LastName  string  `db:"last_name" json:"last_name"`
	AvatarURL *string `db:"avatar_url" json:"avatar_url"`
}

type ConversationCounts struct {
	TotalAssigned         int `db:"total_assigned" json:"total_assigned"`
	UnresolvedCount       int `db:"unresolved_count" json:"unresolved_count"`
	AwaitingResponseCount int `db:"awaiting_response_count" json:"awaiting_response_count"`
	CreatedTodayCount     int `db:"created_today_count" json:"created_today_count"`
}

type NewConversationsStats struct {
	Date             time.Time `db:"date" json:"date"`
	NewConversations int       `db:"new_conversations" json:"new_conversations"`
}
