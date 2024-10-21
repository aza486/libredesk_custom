package models

import (
	"net/textproto"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	cmodels "github.com/abhinavxd/artemis/internal/contact/models"
	mmodels "github.com/abhinavxd/artemis/internal/media/models"
	"github.com/volatiletech/null/v9"
)

var (
	StatusOpen = "Open"

	AssigneeTypeTeam = "team"
	AssigneeTypeUser = "user"

	AllConversations        = "all"
	AssignedConversations   = "assigned"
	UnassignedConversations = "unassigned"
)

type Conversation struct {
	ID                 int         `db:"id" json:"id"`
	CreatedAt          time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at" json:"updated_at"`
	UUID               string      `db:"uuid" json:"uuid"`
	ClosedAt           null.Time   `db:"closed_at" json:"closed_at,omitempty"`
	ResolvedAt         null.Time   `db:"resolved_at" json:"resolved_at,omitempty"`
	ReferenceNumber    null.String `db:"reference_number" json:"reference_number,omitempty"`
	Priority           null.String `db:"priority" json:"priority"`
	Status             null.String `db:"status" json:"status"`
	FirstReplyAt       null.Time   `db:"first_reply_at" json:"first_reply_at"`
	AssignedUserID     null.Int    `db:"assigned_user_id" json:"assigned_user_id"`
	AssignedTeamID     null.Int    `db:"assigned_team_id" json:"assigned_team_id"`
	AssigneeLastSeenAt null.Time   `db:"assignee_last_seen_at" json:"assignee_last_seen_at"`
	cmodels.Contact
	// Psuedo fields.
	Subject            string      `db:"subject" json:"subject"`
	UnreadMessageCount int         `db:"unread_message_count" json:"unread_message_count"`
	InboxName          string      `db:"inbox_name" json:"inbox_name"`
	InboxChannel       string      `db:"inbox_channel" json:"inbox_channel"`
	Tags               null.JSON   `db:"tags" json:"tags"`
	ContactAvatarURL   null.String `db:"contact_avatar_url" json:"contact_avatar_url"`
	LastMessageAt      null.Time   `db:"last_message_at" json:"last_message_at"`
	LastMessage        string      `db:"last_message" json:"last_message"`
	FirstMessage       string      `json:"-"`
}

type ConversationParticipant struct {
	ID        string      `db:"id" json:"id"`
	FirstName string      `db:"first_name" json:"first_name"`
	LastName  string      `db:"last_name" json:"last_name"`
	AvatarURL null.String `db:"avatar_url" json:"avatar_url"`
}

type ConversationCounts struct {
	TotalAssigned         int `db:"total_assigned" json:"total_assigned"`
	UnresolvedCount       int `db:"unresolved_count" json:"unresolved_count"`
	AwaitingResponseCount int `db:"awaiting_response_count" json:"awaiting_response_count"`
	CreatedTodayCount     int `db:"created_today_count" json:"created_today_count"`
}

type NewConversationsStats struct {
	Date             string `db:"date" json:"date"`
	NewConversations int    `db:"new_conversations" json:"new_conversations"`
}

// Message represents a message in a conversation
// TODO: Maybe diffentiate conversation message and a outgoing message.
type Message struct {
	ID             int                    `db:"id" json:"id"`
	CreatedAt      time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time              `db:"updated_at" json:"updated_at"`
	UUID           string                 `db:"uuid" json:"uuid"`
	Type           string                 `db:"type" json:"type"`
	Status         string                 `db:"status" json:"status"`
	ConversationID int                    `db:"conversation_id" json:"conversation_id"`
	Content        string                 `db:"content" json:"content"`
	ContentType    string                 `db:"content_type" json:"content_type"`
	Private        bool                   `db:"private" json:"private"`
	SourceID       null.String            `db:"source_id" json:"-"`
	SenderID       int                    `db:"sender_id" json:"sender_id"`
	SenderType     string                 `db:"sender_type" json:"sender_type"`
	InboxID        int                    `db:"inbox_id" json:"-"`
	Meta           string                 `db:"meta" json:"meta"`
	Attachments    attachment.Attachments `db:"attachments" json:"attachments"`
	// Psuedo fields.
	ConversationUUID string               `db:"conversation_uuid" json:"-"`
	From             string               `db:"from"  json:"-"`
	To               []string             `db:"from"  json:"-"`
	AltContent       string               `db:"alt_content" json:"-"`
	Subject          string               `db:"subject" json:"-"`
	Channel          string               `db:"channel" json:"-"`
	References       []string             `json:"-"`
	InReplyTo        string               `json:"-"`
	Headers          textproto.MIMEHeader `json:"-"`
	Media            []mmodels.Media      `db:"-" json:"-"`
}

// IncomingMessage links a message with the contact information and inbox id.
type IncomingMessage struct {
	Message Message
	Contact cmodels.Contact
	InboxID int
}

type Status struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}

type Priority struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}
