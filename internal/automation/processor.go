package automation

import (
	"fmt"
	"strings"

	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/systeminfo"
)

func (e *Engine) processConversations(conversation cmodels.Conversation) {
	e.lo.Debug("num rules", "rules", len(e.rules))
	for _, rule := range e.rules {
		e.lo.Debug("eval rule", "groups", len(rule.Groups), "rule", rule)
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
		conversationValue string
		conditionMet      bool
	)

	// Extract the value from the conversation based on the rule's field
	switch rule.Field {
	case "subject":
		conversationValue = conversation.Subject
	case "content":
		conversationValue = conversation.FirstMessage
	case "status":
		conversationValue = conversation.Status.String
	case "priority":
		conversationValue = conversation.Priority.String
	default:
		e.lo.Error("rule field not recognized", "field", rule.Field)
		return false
	}

	// Lower case the value.
	conversationValue = strings.ToLower(conversationValue)

	// Compare the conversation value with the rule's value based on the operator
	switch rule.Operator {
	case "equals":
		conditionMet = conversationValue == rule.Value
	case "not equal":
		conditionMet = conversationValue != rule.Value
	case "contains":
		conditionMet = strings.Contains(conversationValue, rule.Value)
	case "startsWith":
		conditionMet = strings.HasPrefix(conversationValue, rule.Value)
	case "endsWith":
		conditionMet = strings.HasSuffix(conversationValue, rule.Value)
	default:
		e.lo.Error("logical operator not recognized for evaluating rules", "operator", rule.Operator)
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
		if err := e.conversationStore.UpdateTeamAssignee(conversation.UUID, []byte(action.Action)); err != nil {
			return err
		}
		if err := e.messageStore.RecordAssigneeTeamChange(conversation.UUID, action.Action, systeminfo.SystemUserUUID); err != nil {
			return err
		}
	case models.ActionAssignAgent:
		if err := e.conversationStore.UpdateStatus(conversation.UUID, []byte(action.Action)); err != nil {
			return err
		}
		if err := e.messageStore.RecordStatusChange(action.Action, conversation.UUID, systeminfo.SystemUserUUID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized rule action: %s", action.Type)
	}
	return nil
}
