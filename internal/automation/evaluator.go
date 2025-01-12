package automation

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
)

// evalConversationRules evaluates a list of rules against a given conversation.
// If all the groups of a rule pass their evaluations based on the defined logical operations,
// the corresponding actions are executed.
func (e *Engine) evalConversationRules(rules []models.Rule, conversation cmodels.Conversation) {
	for _, rule := range rules {
		e.lo.Debug("evaluating rule for conversation", "rule", rule, "conversation_id", conversation.ID)

		// At max there can be only 2 groups.
		if len(rule.Groups) > 2 {
			e.lo.Warn("WARNING: more than 2 groups found for rules skipping evaluation")
			continue
		}

		var results []bool
		for _, group := range rule.Groups {
			if len(group.Rules) == 0 {
				results = append(results, true)
				continue
			}
			result := e.evaluateGroup(group.Rules, group.LogicalOp, conversation)
			e.lo.Debug("evaluating group rules", "logical_op", group.LogicalOp, "result", result, "conversation_uuid", conversation.UUID)
			results = append(results, result)
		}

		if evaluateFinalResult(results, rule.GroupOperator) {
			e.lo.Debug("rule evaluation successful executing actions", "conversation_uuid", conversation.UUID)
			for _, action := range rule.Actions {
				e.applyAction(action, conversation)
			}
			if rule.ExecutionMode == models.ExecutionModeFirstMatch {
				e.lo.Debug("first match rule execution mode, breaking out of rule evaluation", "conversation_uuid", conversation.UUID)
				break
			}
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

// evaluateRule determines if a conversation matches the specified rule's conditions.
func (e *Engine) evaluateRule(rule models.RuleDetail, conversation cmodels.Conversation) bool {
	var (
		valueToCompare string
		ruleValues     []string
		conditionMet   bool
	)

	// Extract the value from the conversation based on the rule's field
	switch rule.Field {
	case models.ConversationSubject:
		valueToCompare = conversation.Subject.String
	case models.ConversationContent:
		valueToCompare = conversation.LastMessage.String
	case models.ConversationStatus:
		valueToCompare = strconv.Itoa(conversation.StatusID.Int)
	case models.ConversationPriority:
		valueToCompare = strconv.Itoa(conversation.PriorityID.Int)
	case models.ConversationAssignedTeam:
		if conversation.AssignedTeamID.Valid {
			valueToCompare = strconv.Itoa(conversation.AssignedTeamID.Int)
		}
	case models.ConversationAssignedUser:
		if conversation.AssignedUserID.Valid {
			valueToCompare = strconv.Itoa(conversation.AssignedUserID.Int)
		}
	case models.ConversationHoursSinceCreated:
		valueToCompare = fmt.Sprintf("%.0f", (time.Since(conversation.CreatedAt).Hours()))
	case models.ConversationHoursSinceResolved:
		if !conversation.ResolvedAt.IsZero() {
			valueToCompare = fmt.Sprintf("%.0f", (time.Since(conversation.ResolvedAt.Time).Hours()))
		}
	case models.ConversationInbox:
		valueToCompare = strconv.Itoa(conversation.InboxID)
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
	if rule.Operator == models.RuleOperatorContains || rule.Operator == models.RuleOperatorNotContains {
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
	case models.RuleOperatorEquals:
		conditionMet = valueToCompare == rule.Value
	case models.RuleOperatorNotEqual:
		conditionMet = valueToCompare != rule.Value
	case models.RuleOperatorContains:
		// Split the value to compare into words
		words := strings.Fields(valueToCompare)
		wordMap := make(map[string]struct{}, len(words))
		for _, word := range words {
			wordMap[word] = struct{}{}
		}
		// Check if any of the rule values exist as complete words
		for _, val := range ruleValues {
			if _, exists := wordMap[val]; exists {
				conditionMet = true
				break
			}
		}
	case models.RuleOperatorNotContains:
		// Split the value to compare into words
		words := strings.Fields(valueToCompare)
		wordMap := make(map[string]struct{}, len(words))
		for _, word := range words {
			wordMap[word] = struct{}{}
		}

		// Check if none of the rule values exist as complete words
		conditionMet = true
		for _, val := range ruleValues {
			if _, exists := wordMap[val]; exists {
				conditionMet = false
				break
			}
		}
	case models.RuleOperatorSet:
		conditionMet = len(valueToCompare) > 0
	case models.RuleOperatorNotSet:
		conditionMet = len(valueToCompare) == 0
	case models.RuleOperatorGreaterThan:
		value1, _ := strconv.Atoi(valueToCompare)
		value2, _ := strconv.Atoi(rule.Value)
		conditionMet = value1 > value2
	default:
		e.lo.Error("error unrecognized rule logical operator", "operator", rule.Operator)
		return false
	}
	e.lo.Debug("conversation automation rule status", "has_met", conditionMet, "conversation_uuid", conversation.UUID)
	return conditionMet
}

// applyAction applies a specific action to the given conversation.
func (e *Engine) applyAction(action models.RuleAction, conversation cmodels.Conversation) error {
	switch action.Type {
	case models.ActionAssignTeam:
		e.lo.Debug("executing assign team action", "value", action.Action, "conversation_uuid", conversation.UUID)
		teamID, _ := strconv.Atoi(action.Action)
		if err := e.conversationStore.UpdateConversationTeamAssignee(conversation.UUID, teamID, e.systemUser); err != nil {
			return err
		}
	case models.ActionAssignUser:
		e.lo.Debug("executing assign user action", "value", action.Action, "conversation_uuid", conversation.UUID)
		agentID, _ := strconv.Atoi(action.Action)
		if err := e.conversationStore.UpdateConversationUserAssignee(conversation.UUID, agentID, e.systemUser); err != nil {
			return err
		}
	case models.ActionSetPriority:
		e.lo.Debug("executing set priority action", "value", action.Action, "conversation_uuid", conversation.UUID)
		priorityID, _ := strconv.Atoi(action.Action)
		if err := e.conversationStore.UpdateConversationPriority(conversation.UUID, priorityID, "", e.systemUser); err != nil {
			return err
		}
	case models.ActionSetStatus:
		e.lo.Debug("executing set status action", "value", action.Action, "conversation_uuid", conversation.UUID)
		statusID, _ := strconv.Atoi(action.Action)
		if err := e.conversationStore.UpdateConversationStatus(conversation.UUID, statusID, "", "", e.systemUser); err != nil {
			return err
		}
	case models.ActionSendPrivateNote:
		e.lo.Debug("executing send private note action", "value", action.Action, "conversation_uuid", conversation.UUID)
		if err := e.conversationStore.SendPrivateNote([]mmodels.Media{}, e.systemUser.ID, conversation.UUID, action.Action); err != nil {
			return err
		}
	case models.ActionReply:
		e.lo.Debug("executing reply action", "value", action.Action, "conversation_uuid", conversation.UUID)
		if err := e.conversationStore.SendReply([]mmodels.Media{}, e.systemUser.ID, conversation.UUID, action.Action, "" /**meta**/); err != nil {
			return err
		}
	case models.ActionSetSLA:
		e.lo.Debug("executing SLA action", "value", action.Action, "conversation_uuid", conversation.UUID)
		slaID, _ := strconv.Atoi(action.Action)
		if err := e.slaStore.ApplySLA(conversation.ID, slaID); err != nil {
			return err
		}
		if err := e.conversationStore.RecordSLASet(conversation.UUID, e.systemUser); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unrecognized rule action: %s", action.Type)
	}
	return nil
}
