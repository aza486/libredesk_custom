package models

import (
	"encoding/json"
	"time"
)

const (
	ActionAssignTeam      = "assign_team"
	ActionAssignUser      = "assign_user"
	ActionSetStatus       = "set_status"
	ActionSetPriority     = "set_priority"
	ActionSendPrivateNote = "send_private_note"
	ActionReply           = "reply"

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
)

// RuleRecord represents a rule record in the database
type RuleRecord struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Type        string          `db:"type" json:"type"`
	Disabled    bool            `db:"disabled" json:"disabled"`
	Rules       json.RawMessage `db:"rules" json:"rules"`
}

type Rule struct {
	Type          string       `json:"type" db:"type"`
	GroupOperator string       `json:"group_operator" db:"group_operator"`
	Groups        []RuleGroup  `json:"groups" db:"groups"`
	Actions       []RuleAction `json:"actions" db:"actions"`
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
