package automation

import (
	"fmt"
	"strings"

	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/systeminfo"
)

func (e *Engine) processConversations(conv cmodels.Conversation) {
	var (
		groupRules    = make(map[int][]models.Rule)
		groupOperator = make(map[int]string)
	)

	// Group rules by RuleID and their logical operators.
	for _, rule := range e.rules {
		groupRules[rule.GroupID] = append(groupRules[rule.GroupID], rule)
		groupOperator[rule.GroupID] = rule.LogicalOp
	}

	fmt.Printf("%+v \n", e.actions)
	fmt.Printf("%+v \n", e.rules)

	// Evaluate rules grouped by RuleID
	for groupID, rules := range groupRules {
		e.lo.Debug("evaluating group rule", "group_id", groupID, "operator", groupOperator[groupID])
		if e.evaluateGroup(rules, groupOperator[groupID], conv) {
			for _, action := range e.actions {
				if action.RuleID == rules[0].ID {
					e.executeActions(conv)
				}
			}
		}
	}
}

// Helper function to evaluate a group of rules
func (e *Engine) evaluateGroup(rules []models.Rule, operator string, conv cmodels.Conversation) bool {
	switch operator {
	case "AND":
		// All conditions within the group must be true
		for _, rule := range rules {
			if !e.evaluateRule(rule, conv) {
				e.lo.Debug("rule evaluation was not success", "id", rule.ID)
				return false
			}
		}
		e.lo.Debug("all AND rules are success")
		return true
	case "OR":
		// At least one condition within the group must be true
		for _, rule := range rules {
			if e.evaluateRule(rule, conv) {
				e.lo.Debug("OR rules are success", "id", rule.ID)
				return true
			}
		}
		return false
	default:
		e.lo.Error("invalid group operator", "operator", operator)
	}
	return false
}

func (e *Engine) evaluateRule(rule models.Rule, conv cmodels.Conversation) bool {
	var (
		conversationValue string
		conditionMet      bool
	)

	// Extract the value from the conversation based on the rule's field
	switch rule.Field {
	case "subject":
		conversationValue = conv.Subject
	case "content":
		conversationValue = conv.FirstMessage
	case "status":
		conversationValue = conv.Status.String
	case "priority":
		conversationValue = conv.Priority.String
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
		e.lo.Debug("eval rule", "field", rule.Field, "conv_val", conversationValue, "rule_val", rule.Value)
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

func (e *Engine) executeActions(conv cmodels.Conversation) {
	for _, action := range e.actions {
		err := e.processAction(action, conv)
		if err != nil {
			e.lo.Error("error executing rule action", "action", action.Action, "error", err)
		}
	}
}

func (e *Engine) processAction(action models.Action, conv cmodels.Conversation) error {
	switch action.Type {
	case models.ActionAssignTeam:
		if err := e.convUpdater.UpdateAssignee(conv.UUID, []byte(action.Action), "team"); err != nil {
			return err
		}
		if err := e.msgRecorder.RecordAssigneeUserChange(action.Action, conv.UUID, systeminfo.SystemUserUUID); err != nil {
			return err
		}
	case models.ActionAssignAgent:
		if err := e.convUpdater.UpdateStatus(conv.UUID, []byte(action.Action)); err != nil {
			return err
		}
		if err := e.msgRecorder.RecordStatusChange(action.Action, conv.UUID, systeminfo.SystemUserUUID); err != nil {
			return err
		}
	default:
		return fmt.Errorf("rule action not recognized: %s", action.Type)
	}
	return nil
}
