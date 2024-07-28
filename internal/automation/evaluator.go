package automation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
)

// evalConversationRules evaluates a list of rules against a given conversation.
// If all the groups of a rule pass their evaluations based on the defined logical operations,
// the corresponding actions are executed.
func (e *Engine) evalConversationRules(rules []models.Rule, conversation cmodels.Conversation) {
	for _, rule := range rules {
		// At max there can be only 2 groups.
		if len(rule.Groups) > 2 {
			e.lo.Warn("more than 2 groups found for rules")
			continue
		}

		var results []bool

		for _, group := range rule.Groups {
			e.lo.Debug("evaluating group rule", "logical_op", group.LogicalOp)
			result := e.evaluateGroup(group.Rules, group.LogicalOp, conversation)
			e.lo.Debug("group evaluation status", "status", result)
			results = append(results, result)
		}

		if evaluateFinalResult(results, rule.GroupOperator) {
			e.lo.Debug("rule fully evaluated, executing actions")
			// All group rule evaluations successful, execute the actions.
			for _, action := range rule.Actions {
				e.applyAction(action, conversation)
			}
		} else {
			e.lo.Debug("rule evaluation failed, NOT executing actions")
		}
	}
}

// evaluateFinalResult computes the final result of multiple group evaluations
// based on the specified logical operator (AND/OR).
func evaluateFinalResult(results []bool, operator string) bool {
	if operator == models.OperatorAnd {
		for _, result := range results {
			if !result {
				return false
			}
		}
		return true
	}
	if operator == models.OperatorOR {
		for _, result := range results {
			if result {
				return true
			}
		}
		return false
	}
	return false
}

// evaluateGroup evaluates a set of rules within a group against a given conversation
// based on the specified logical operator (AND/OR).
func (e *Engine) evaluateGroup(rules []models.RuleDetail, operator string, conversation cmodels.Conversation) bool {
	switch operator {
	case models.OperatorAnd:
		// All conditions within the group must be true
		for _, rule := range rules {
			if !e.evaluateRule(rule, conversation) {
				return false
			}
		}
		return true
	case models.OperatorOR:
		// At least one condition within the group must be true
		for _, rule := range rules {
			if e.evaluateRule(rule, conversation) {
				return true
			}
		}
		return false
	default:
		e.lo.Error("invalid group operator", "operator", operator)
	}
	return false
}

// evaluateRule evaluates a single rule against a given conversation.
func (e *Engine) evaluateRule(rule models.RuleDetail, conversation cmodels.Conversation) bool {
	var (
		valueToCompare string
		conditionMet   bool
	)

	// Extract the value from the conversation based on the rule's field
	switch rule.Field {
	case models.ConversationFieldSubject:
		valueToCompare = conversation.Subject
	case models.ConversationFieldContent:
		valueToCompare = conversation.FirstMessage
	case models.ConversationFieldStatus:
		valueToCompare = conversation.Status.String
	case models.ConversationFieldPriority:
		valueToCompare = conversation.Priority.String
	case models.ConversationFieldAssignedTeamID:
		if conversation.AssignedTeamID.Valid {
			valueToCompare = strconv.Itoa(conversation.AssignedTeamID.Int)
		}
	case models.ConversationFieldAssignedUserID:
		if conversation.AssignedUserID.Valid {
			valueToCompare = strconv.Itoa(conversation.AssignedUserID.Int)
		}
	default:
		e.lo.Error("rule field not recognized", "field", rule.Field)
		return false
	}

	if !rule.CaseSensitiveMatch {
		valueToCompare = strings.ToLower(valueToCompare)
		rule.Value = strings.ToLower(rule.Value)
	}

	e.lo.Debug("comparing values", "conversation_value", valueToCompare, "rule_value", rule.Value)

	switch rule.Operator {
	case models.RuleEquals:
		conditionMet = valueToCompare == rule.Value
	case models.RuleNotEqual:
		conditionMet = valueToCompare != rule.Value
	case models.RuleContains:
		conditionMet = strings.Contains(valueToCompare, rule.Value)
	case models.RuleNotContains:
		conditionMet = !strings.Contains(valueToCompare, rule.Value)
	case models.RuleSet:
		conditionMet = len(valueToCompare) > 0
	case models.RuleNotSet:
		conditionMet = len(valueToCompare) == 0
	default:
		e.lo.Error("rule logical operator not recognized", "operator", rule.Operator)
		return false
	}
	return conditionMet
}

// applyAction applies a specific action to the given conversation.
func (e *Engine) applyAction(action models.RuleAction, conversation cmodels.Conversation) error {
	switch action.Type {
	case models.ActionAssignTeam:
		teamID, err := strconv.Atoi(action.Action)
		if err != nil {
			e.lo.Error("error converting string to int", "string", action.Action, "error", err)
			return err
		}
		if err := e.conversationStore.UpdateConversationTeamAssignee(conversation.UUID, teamID, e.systemUser); err != nil {
			return err
		}
	case models.ActionAssignUser:
		agentID, err := strconv.Atoi(action.Action)
		if err != nil {
			e.lo.Error("error converting string to int", "string", action.Action, "error", err)
			return err
		}
		if err := e.conversationStore.UpdateConversationUserAssignee(conversation.UUID, agentID, e.systemUser); err != nil {
			return err
		}
	case models.ActionSetPriority:
		if err := e.conversationStore.UpdateConversationPriority(conversation.UUID, []byte(action.Action), e.systemUser); err != nil {
			return err
		}
	case models.ActionSetStatus:
		if err := e.conversationStore.UpdateConversationStatus(conversation.UUID, []byte(action.Action), e.systemUser); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized rule action: %s", action.Type)
	}
	return nil
}
