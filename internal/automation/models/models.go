package models

const (
	ActionAssignTeam  = "assign_team"
	ActionAssignAgent = "assign_agent"
	OperatorAnd       = "AND"
	OperatorOR        = "OR"
)

type Rule struct {
	GroupOperator string       `json:"group_operator" db:"group_operator"`
	Groups        []RuleGroup  `json:"groups" db:"groups"`
	Actions       []RuleAction `json:"actions" db:"actions"`
}

type RuleGroup struct {
	LogicalOp string       `json:"logical_op" db:"logical_op"`
	Rules     []RuleDetail `json:"rules" db:"rules"`
}

type RuleDetail struct {
	Field    string `json:"field" db:"field"`
	Operator string `json:"operator" db:"operator"`
	Value    string `json:"value" db:"value"`
}

type RuleAction struct {
	Type   string `json:"action_type" db:"action_type"`
	Action string `json:"action" db:"action"`
}
