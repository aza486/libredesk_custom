package automation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
)

func (e *Engine) evalConversationRules(rules []models.Rule, conversation cmodels.Conversation) {
	e.lo.Debug("num rules", "rules", len(rules))
	for _, rule := range rules {
		e.lo.Debug("eval rule", "groups", len(rule.Groups), "rule", rule)
		// At max there can be only 2 groups.
		if len(rule.Groups) > 2 {
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
			e.lo.Debug("rule fully evalauted, executing actions")
			// All group rule evaluations successful, execute the actions.
			for _, action := range rule.Actions {
				e.executeActions(conversation, action)
			}
		} else {
			e.lo.Debug("rule evaluation failed, NOT executing actions")
		}
	}
}

// evaluateFinalResult
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

// evaluateGroup
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
		conditionMet = bool(len(valueToCompare) > 0)
	case models.RuleNotSet:
		conditionMet = !bool(len(valueToCompare) > 0)
	default:
		e.lo.Error("rule logical operator not recognized", "operator", rule.Operator)
		return false
	}
	return conditionMet
}

func (e *Engine) executeActions(conversation cmodels.Conversation, action models.RuleAction) {
	err := e.applyAction(action, conversation)
	if err != nil {
		e.lo.Error("error executing rule action", "action", action.Action, "error", err)
	}
}

func (e *Engine) applyAction(action models.RuleAction, conversation cmodels.Conversation) error {
	switch action.Type {
	case models.ActionAssignTeam:
		teamID, err := strconv.Atoi(action.Action)
		if err != nil {
			e.lo.Error("error converting string to int", "string", action.Action, "error", err)
			return err
		}
		if err := e.conversationStore.UpdateTeamAssignee(conversation.UUID, teamID, e.systemUser); err != nil {
			return err
		}
	case models.ActionAssignUser:
		agentID, err := strconv.Atoi(action.Action)
		if err != nil {
			e.lo.Error("error converting string to int", "string", action.Action, "error", err)
			return err
		}
		if err := e.conversationStore.UpdateUserAssignee(conversation.UUID, agentID, e.systemUser); err != nil {
			return err
		}
	case models.ActionSetPriority:
		if err := e.conversationStore.UpdatePriority(conversation.UUID, []byte(action.Action), e.systemUser); err != nil {
			return err
		}
	case models.ActionSetStatus:
		if err := e.conversationStore.UpdateStatus(conversation.UUID, []byte(action.Action), e.systemUser); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized rule action: %s", action.Type)
	}
	return nil
}
