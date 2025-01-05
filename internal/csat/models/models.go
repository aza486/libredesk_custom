// package models has the models for the customer satisfaction survey responses.
package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// CSATResponse represents a customer satisfaction survey response.
type CSATResponse struct {
	ID                int         `db:"id"`
	UUID              string      `db:"uuid"`
	CreatedAt         time.Time   `db:"created_at"`
	UpdatedAt         time.Time   `db:"updated_at"`
	ConversationID    int         `db:"conversation_id"`
	AssignedAgentID   int         `db:"assigned_agent_id"`
	Score             int         `db:"rating"`
	Feedback          null.String `db:"feedback"`
	ResponseTimestamp null.Time   `db:"response_timestamp"`
}
