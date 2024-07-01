package models

const (
	ActionAssignTeam  = "assign_team"
	ActionAssignAgent = "assign_agent"
	ActionSetStatus   = "set_status"
	ActionSetPriority = "set_priority"

	OperatorAnd = "AND"
	OperatorOR  = "OR"

	RuleContains    = "contains"
	RuleNotContains = "not contains"
	RuleEquals      = "equals"
	RuleNotEqual    = "not equal"
	RuleSet         = "set"
	RuleNotSet      = "not set"

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
	Type   string `json:"action_type" db:"action_type"`
	Action string `json:"action" db:"action"`
}
