package automation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Engine struct {
	q                   queries
	lo                  *logf.Logger
	conversationStore   ConversationStore
	systemUser          umodels.User
	rules               []models.Rule
	newConversationQ    chan string
	updateConversationQ chan string
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type ConversationStore interface {
	Get(uuid string) (cmodels.Conversation, error)
	GetRecentConversations(t time.Time) ([]cmodels.Conversation, error)
	UpdateTeamAssignee(uuid string, teamID int, actor umodels.User) error
	UpdateUserAssignee(uuid string, assigneeID int, actor umodels.User) error
	UpdateStatus(uuid string, status []byte, actor umodels.User) error
	UpdatePriority(uuid string, priority []byte, actor umodels.User) error
}

type UserStore interface {
	GetSystemUser() (umodels.User, error)
}

type queries struct {
	GetRules   *sqlx.Stmt `query:"get-rules"`
	GetAll     *sqlx.Stmt `query:"get-all"`
	GetRule    *sqlx.Stmt `query:"get-rule"`
	InsertRule *sqlx.Stmt `query:"insert-rule"`
	UpdateRule *sqlx.Stmt `query:"update-rule"`
	DeleteRule *sqlx.Stmt `query:"delete-rule"`
}

func New(systemUser umodels.User, opt Opts) (*Engine, error) {
	var (
		q queries
		e = &Engine{
			lo:                  opt.Lo,
			newConversationQ:    make(chan string, 5000),
			updateConversationQ: make(chan string, 5000),
		}
	)
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	e.q = q
	e.rules = e.queryRules()
	return e, nil
}

func (e *Engine) ReloadRules() {
	e.rules = e.queryRules()
}

func (e *Engine) SetConversationStore(store ConversationStore) {
	e.conversationStore = store
}

func (e *Engine) Serve(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Create separate semaphores for each channel
	maxWorkers := 10
	newConversationSemaphore := make(chan struct{}, maxWorkers)
	updateConversationSemaphore := make(chan struct{}, maxWorkers)
	timeTriggerSemaphore := make(chan struct{}, maxWorkers)

	for {
		select {
		case <-ctx.Done():
			return
		case conversationUUID := <-e.newConversationQ:
			newConversationSemaphore <- struct{}{}
			go e.handleNewConversation(conversationUUID, newConversationSemaphore)
		case conversationUUID := <-e.updateConversationQ:
			updateConversationSemaphore <- struct{}{}
			go e.handleUpdateConversation(conversationUUID, updateConversationSemaphore)
		case <-ticker.C:
			e.lo.Info("evaluating time triggers")
			timeTriggerSemaphore <- struct{}{}
			go e.handleTimeTrigger(timeTriggerSemaphore)
		}
	}
}

func (e *Engine) GetAllRules() ([]models.RuleRecord, error) {
	var rules = make([]models.RuleRecord, 0)
	if err := e.q.GetAll.Select(&rules); err != nil {
		e.lo.Error("error fetching rules", "error", err)
		return rules, envelope.NewError(envelope.GeneralError, "Error fetching automation rules.", nil)
	}
	return rules, nil
}

func (e *Engine) GetRule(id int) (models.RuleRecord, error) {
	var rule = models.RuleRecord{}
	if err := e.q.GetRule.Get(&rule, id); err != nil {
		if err == sql.ErrNoRows {
			return rule, envelope.NewError(envelope.InputError, "Rule not found.", nil)
		}
		e.lo.Error("error fetching rule", "error", err)
		return rule, envelope.NewError(envelope.GeneralError, "Error fetching automation rule.", nil)
	}
	return rule, nil
}

func (e *Engine) UpdateRule(id int, rule models.RuleRecord) error {
	if _, err := e.q.UpdateRule.Exec(id, rule.Name, rule.Description, rule.Type, rule.Rules); err != nil {
		e.lo.Error("error updating rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating automation rule.", nil)
	}
	return nil
}

func (e *Engine) CreateRule(rule models.RuleRecord) error {
	if _, err := e.q.InsertRule.Exec(rule.Name, rule.Description, rule.Type, rule.Rules); err != nil {
		e.lo.Error("error creating rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating automation rule.", nil)
	}
	return nil
}

func (e *Engine) DeleteRule(id int) error {
	if _, err := e.q.DeleteRule.Exec(id); err != nil {
		e.lo.Error("error deleting rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting automation rule.", nil)
	}
	return nil
}

func (e *Engine) handleNewConversation(conversationUUID string, semaphore chan struct{}) {
	defer func() { <-semaphore }()
	conversation, err := e.conversationStore.Get(conversationUUID)
	if err != nil {
		e.lo.Error("error could not fetch conversations to evaluate new conversation rules", "conversation_uuid", conversationUUID)
		return
	}
	rules := e.filterRulesByType(models.RuleTypeNewConversation)
	e.evalConversationRules(rules, conversation)
}

func (e *Engine) handleUpdateConversation(conversationUUID string, semaphore chan struct{}) {
	defer func() { <-semaphore }()
	conversation, err := e.conversationStore.Get(conversationUUID)
	if err != nil {
		e.lo.Error("error could not fetch conversations to evaluate update conversation rules", "conversation_uuid", conversationUUID)
		return
	}
	rules := e.filterRulesByType(models.RuleTypeConversationUpdate)
	e.evalConversationRules(rules, conversation)
}

func (e *Engine) handleTimeTrigger(semaphore chan struct{}) {
	defer func() { <-semaphore }()
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	conversations, err := e.conversationStore.GetRecentConversations(thirtyDaysAgo)
	if err != nil {
		e.lo.Error("error could not fetch conversations to evaluate time triggers")
		return
	}
	rules := e.filterRulesByType(models.RuleTypeTimeTrigger)
	for _, conversation := range conversations {
		e.evalConversationRules(rules, conversation)
	}
}

func (e *Engine) EvaluateNewConversationRules(conversationUUID string) {
	select {
	case e.newConversationQ <- conversationUUID:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateNewConversationRules: newConversationQ is full, unable to enqueue conversation")
	}
}

func (e *Engine) EvaluateConversationUpdateRules(conversationUUID string) {
	select {
	case e.updateConversationQ <- conversationUUID:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateConversationUpdateRules: updateConversationQ is full, unable to enqueue conversation")
	}
}

func (e *Engine) queryRules() []models.Rule {
	var (
		rulesJSON []string
		rules     []models.Rule
	)
	err := e.q.GetRules.Select(&rulesJSON)
	if err != nil {
		e.lo.Error("error fetching automation rules", "error", err)
		return nil
	}

	for _, ruleJSON := range rulesJSON {
		var rulesBatch []models.Rule
		if err := json.Unmarshal([]byte(ruleJSON), &rulesBatch); err != nil {
			e.lo.Error("error unmarshalling rule JSON", "error", err)
			continue
		}
		rules = append(rules, rulesBatch...)
	}

	e.lo.Debug("fetched rules", "num", len(rules), "rules", fmt.Sprintf("%+v", rules))
	return rules
}

func (e *Engine) filterRulesByType(ruleType string) []models.Rule {
	var filteredRules []models.Rule
	for _, rule := range e.rules {
		if rule.Type == ruleType {
			filteredRules = append(filteredRules, rule)
		}
	}
	return filteredRules
}
