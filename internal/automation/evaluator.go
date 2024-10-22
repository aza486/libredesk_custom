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
		e.lo.Debug("evaluating rule for conversation", "rule", rule, "conversation_uuid", conversation.UUID)

		// At max there can be only 2 groups.
		if len(rule.Groups) > 2 {
			e.lo.Warn("WARNING: more than 2 groups found for rules skipping evaluation")
			continue
		}

		var results []bool
		for _, group := range rule.Groups {
			result := e.evaluateGroup(group.Rules, group.LogicalOp, conversation)
			e.lo.Debug("evaluating group rules", "logical_op", group.LogicalOp, "result", result, "conversation_uuid", conversation.UUID)
			results = append(results, result)
		}

		if evaluateFinalResult(results, rule.GroupOperator) {
			e.lo.Debug("rule evaluation successfull executing actions", "conversation_uuid", conversation.UUID)
			// All group rule evaluations successful, execute the actions.
			for _, action := range rule.Actions {
				e.applyAction(action, conversation)
			}
		} else {
			e.lo.Debug("rule evaluation failed", "conversation_uuid", conversation.UUID)
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

func (e *Engine) evaluateRule(rule models.RuleDetail, conversation cmodels.Conversation) bool {
	var (
		valueToCompare string
		ruleValues     []string
		conditionMet   bool
	)

	// Extract the value from the conversation based on the rule's field
	switch rule.Field {
	case models.ConversationFieldSubject:
		valueToCompare = conversation.Subject
	case models.ConversationFieldContent:
		valueToCompare = conversation.LastMessage
	case models.ConversationFieldStatus:
		valueToCompare = conversation.Status.String
	case models.ConversationFieldPriority:
		valueToCompare = conversation.Priority.String
	case models.ConversationFieldAssignedTeam:
		if conversation.AssignedTeamID.Valid {
			valueToCompare = strconv.Itoa(conversation.AssignedTeamID.Int)
		}
	case models.ConversationFieldAssignedUser:
		if conversation.AssignedUserID.Valid {
			valueToCompare = strconv.Itoa(conversation.AssignedUserID.Int)
		}
	default:
		e.lo.Error("unrecognized rule field", "field", rule.Field)
		return false
	}

	// Case sensitivity handling
	if !rule.CaseSensitiveMatch {
		valueToCompare = strings.ToLower(valueToCompare)
		rule.Value = strings.ToLower(rule.Value)
	}

	// Split and trim values for Contains/NotContains operations
	if rule.Operator == models.RuleContains || rule.Operator == models.RuleNotContains {
		ruleValues = strings.Split(rule.Value, ",")
		for i := range ruleValues {
			ruleValues[i] = strings.TrimSpace(ruleValues[i])
			if !rule.CaseSensitiveMatch {
				ruleValues[i] = strings.ToLower(ruleValues[i])
			}
		}
	}

	e.lo.Debug("evaluating rule", "rule_field", rule.Field, "rule_operator", rule.Operator,
		"rule_value", rule.Value, "rule_values", ruleValues, "value_to_compare",
		valueToCompare, "conversation_uuid", conversation.UUID)

	// Compare with set operator
	switch rule.Operator {
	case models.RuleEquals:
		conditionMet = valueToCompare == rule.Value
	case models.RuleNotEqual:
		conditionMet = valueToCompare != rule.Value
	case models.RuleContains:
		for _, val := range ruleValues {
			if strings.Contains(valueToCompare, val) {
				conditionMet = true
				break
			}
		}
	case models.RuleNotContains:
		conditionMet = true
		for _, val := range ruleValues {
			if strings.Contains(valueToCompare, val) {
				conditionMet = false
				break
			}
		}
	case models.RuleSet:
		conditionMet = len(valueToCompare) > 0
	case models.RuleNotSet:
		conditionMet = len(valueToCompare) == 0
	default:
		e.lo.Error("unrecognized rule logical operator", "operator", rule.Operator)
		return false
	}
	e.lo.Debug("rule conditions met status", "met", conditionMet,
		"conversation_uuid", conversation.UUID)
	return conditionMet
}

// applyAction applies a specific action to the given conversation.
func (e *Engine) applyAction(action models.RuleAction, conversation cmodels.Conversation) error {
	switch action.Type {
	case models.ActionAssignTeam:
		e.lo.Debug("executing assign team action", "value", action.Action, "conversation_uuid", conversation.UUID)
		teamID, err := strconv.Atoi(action.Action)
		if err != nil {
			e.lo.Error("error converting string to int", "string", action.Action, "error", err)
			return err
		}
		if err := e.conversationStore.UpdateConversationTeamAssignee(conversation.UUID, teamID, e.systemUser); err != nil {
			return err
		}
	case models.ActionAssignUser:
		e.lo.Debug("executing assign user action", "value", action.Action, "conversation_uuid", conversation.UUID)
		agentID, err := strconv.Atoi(action.Action)
		if err != nil {
			e.lo.Error("error converting string to int", "string", action.Action, "error", err)
			return err
		}
		if err := e.conversationStore.UpdateConversationUserAssignee(conversation.UUID, agentID, e.systemUser); err != nil {
			return err
		}
	case models.ActionSetPriority:
		e.lo.Debug("executing set priority action", "value", action.Action, "conversation_uuid", conversation.UUID)
		if err := e.conversationStore.UpdateConversationPriority(conversation.UUID, []byte(action.Action), e.systemUser); err != nil {
			return err
		}
	case models.ActionSetStatus:
		e.lo.Debug("executing set status action", "value", action.Action, "conversation_uuid", conversation.UUID)
		if err := e.conversationStore.UpdateConversationStatus(conversation.UUID, []byte(action.Action), e.systemUser); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized rule action: %s", action.Type)
	}
	return nil
}
