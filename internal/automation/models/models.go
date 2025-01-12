package models

import (
	"encoding/json"
	"time"

	"github.com/lib/pq"
)

const (
	ActionAssignTeam      = "assign_team"
	ActionAssignUser      = "assign_user"
	ActionSetStatus       = "set_status"
	ActionSetPriority     = "set_priority"
	ActionSendPrivateNote = "send_private_note"
	ActionReply           = "send_reply"
	ActionSetSLA          = "set_sla"

	OperatorAnd = "AND"
	OperatorOR  = "OR"

	RuleOperatorContains    = "contains"
	RuleOperatorNotContains = "not contains"
	RuleOperatorEquals      = "equals"
	RuleOperatorNotEqual    = "not equals"
	RuleOperatorSet         = "set"
	RuleOperatorNotSet      = "not set"
	RuleOperatorGreaterThan = "greater than"

	RuleTypeNewConversation    = "new_conversation"
	RuleTypeConversationUpdate = "conversation_update"
	RuleTypeTimeTrigger        = "time_trigger"

	ConversationSubject            = "subject"
	ConversationContent            = "content"
	ConversationStatus             = "status"
	ConversationPriority           = "priority"
	ConversationAssignedUser       = "assigned_user"
	ConversationAssignedTeam       = "assigned_team"
	ConversationHoursSinceCreated  = "hours_since_created"
	ConversationHoursSinceResolved = "hours_since_resolved"
	ConversationInbox              = "inbox"

	EventConversationUserAssigned    = "conversation.user.assigned"
	EventConversationTeamAssigned    = "conversation.team.assigned"
	EventConversationStatusChange    = "conversation.status.change"
	EventConversationPriorityChange  = "conversation.priority.change"
	EventConversationMessageOutgoing = "conversation.message.outgoing"
	EventConversationMessageIncoming = "conversation.message.incoming"

	ExecutionModeAll        = "all"
	ExecutionModeFirstMatch = "first_match"
)

// RuleRecord represents a rule record in the database
type RuleRecord struct {
	ID            int             `db:"id" json:"id"`
	CreatedAt     time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at" json:"updated_at"`
	Name          string          `db:"name" json:"name"`
	Description   string          `db:"description" json:"description"`
	Type          string          `db:"type" json:"type"`
	Events        pq.StringArray  `db:"events" json:"events"`
	Disabled      bool            `db:"disabled" json:"disabled"`
	Weight        int             `db:"weight" json:"weight"`
	ExecutionMode string          `db:"execution_mode" json:"execution_mode"`
	Rules         json.RawMessage `db:"rules" json:"rules"`
}

type Rule struct {
	Type          string       `json:"type"`
	ExecutionMode string       `json:"execution_mode"`
	Events        []string     `json:"event"`
	GroupOperator string       `json:"group_operator"`
	Groups        []RuleGroup  `json:"groups"`
	Actions       []RuleAction `json:"actions"`
}

type RuleGroup struct {
	LogicalOp string       `json:"logical_op" db:"logical_op"`
	Rules     []RuleDetail `json:"rules" db:"rules"`
}

type RuleDetail struct {
	Field              string `json:"field" db:"field"`
	Operator           string `json:"operator" db:"operator"`
	Value              string `json:"value" db:"value"`
	CaseSensitiveMatch bool   `json:"case_sensitive_match" db:"case_sensitive_match"`
}

type RuleAction struct {
	Type   string `json:"type" db:"type"`
	Action string `json:"value" db:"value"`
}
