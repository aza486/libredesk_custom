package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// SLAPolicy represents a service level agreement policy definition
type SLAPolicy struct {
	ID                int       `db:"id" json:"id"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time `db:"updated_at" json:"updated_at"`
	Name              string    `db:"name" json:"name"`
	Description       string    `db:"description" json:"description"`
	FirstResponseTime string    `db:"first_response_time" json:"first_response_time"`
	EveryResponseTime string    `db:"every_response_time" json:"every_response_time"`
	ResolutionTime    string    `db:"resolution_time" json:"resolution_time"`
}

// AppliedSLA represents an SLA policy applied to a conversation with its deadlines and breach status
type AppliedSLA struct {
	ID                      int       `db:"id"`
	CreatedAt               time.Time `db:"created_at"`
	ConversationID          int       `db:"conversation_id"`
	SLAPolicyID             int       `db:"sla_policy_id"`
	FirstResponseDeadlineAt time.Time `db:"first_response_deadline_at"`
	ResolutionDeadlineAt    time.Time `db:"resolution_deadline_at"`
	FirstResponseBreachedAt null.Time `db:"first_response_breached_at"`
	ResolutionBreachedAt    null.Time `db:"resolution_breached_at"`
	FirstResponseMetAt      null.Time `db:"first_response_met_at"`
	ResolutionMetAt         null.Time `db:"resolution_met_at"`

	ConversationFirstResponseAt null.Time `db:"conversation_first_response_at"`
	ConversationResolvedAt      null.Time `db:"conversation_resolved_at"`
}
