package models

import (
	"encoding/json"
	"time"
)

const (
	ActionAssignTeam  = "assign_team"
	ActionAssignUser  = "assign_user"
	ActionSetStatus   = "set_status"
	ActionSetPriority = "set_priority"

	OperatorAnd = "AND"
	OperatorOR  = "OR"

	RuleContains    = "contains"
	RuleNotContains = "not_contains"
	RuleEquals      = "equals"
	RuleNotEqual    = "not_equals"
	RuleSet         = "set"
	RuleNotSet      = "not_set"

	RuleTypeNewConversation    = "new_conversation"
	RuleTypeConversationUpdate = "conversation_update"
	RuleTypeTimeTrigger        = "time_trigger"

	ConversationFieldSubject        = "subject"
	ConversationFieldContent        = "content"
	ConversationFieldStatus         = "status"
	ConversationFieldPriority       = "priority"
	ConversationFieldAssignedUserID = "assigned_user_id"
	ConversationFieldAssignedTeamID = "assigned_team_id"
)

// RuleRecord represents a rule record in the database
type RuleRecord struct {
	ID          int             `db:"id" json:"id"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Type        string          `db:"type" json:"type"`
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
