package models

const (
	ActionAssignTeam  = "assign_team"
	ActionAssignAgent = "assign_agent"

	RuleTypeNewConversation = "new_conversation"
)

type Rule struct {
	ID        int    `db:"id"`
	Type      string `db:"type"`
	Field     string `db:"field"`
	Operator  string `db:"operator"`
	Value     string `db:"value"`
	GroupID   int    `db:"group_id"`
	LogicalOp string `db:"logical_op"`
}

type Action struct {
	RuleID int    `db:"rule_id"`
	Type   string `db:"action_type"`
	Action string `db:"action"`
}
