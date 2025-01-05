// package models contains the model definitions for the SLA package.
package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// SLAPolicy represents an SLA policy.
type SLAPolicy struct {
	ID                int       `db:"id" json:"id"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
	Name              string    `db:"name" json:"name"`
	Description       string    `db:"description" json:"description"`
	FirstResponseTime string    `db:"first_response_time" json:"first_response_time"`
	ResolutionTime    string    `db:"resolution_time" json:"resolution_time"`
	EveryResponseTime string    `db:"every_response_time" json:"every_response_time"`
}

// ConversationSLA represents an SLA policy applied to a conversation.
type ConversationSLA struct {
	ID                         int       `db:"id"`
	CreatedAt                  time.Time `db:"created_at"`
	ConversationID             int       `db:"conversation_id"`
	ConversationCreatedAt      time.Time `db:"conversation_created_at"`
	ConversationFirstReplyAt   null.Time `db:"conversation_first_reply_at"`
	ConversationLastMessageAt  null.Time `db:"conversation_last_message_at"`
	ConversationResolvedAt     null.Time `db:"conversation_resolved_at"`
	ConversationAssignedTeamID null.Int  `db:"conversation_assigned_team_id"`
	SLAPolicyID                int       `db:"sla_policy_id"`
	SLAType                    string    `db:"sla_type"`
	DueAt                      null.Time `db:"due_at"`
	BreachedAt                 null.Time `db:"breached_at"`
}
