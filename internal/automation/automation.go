// Package automation automatically evaluates and applies rules to conversations based on events like new conversations, updates, and time triggers,
// and performs some actions if they are true.
package automation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"sync"
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
	rules   []models.Rule
	rulesMu *sync.RWMutex

	q                   queries
	lo                  *logf.Logger
	conversationStore   ConversationStore
	systemUser          umodels.User
	newConversationQ    chan string
	updateConversationQ chan string
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type ConversationStore interface {
	GetConversation(uuid string) (cmodels.Conversation, error)
	GetRecentConversations(t time.Time) ([]cmodels.Conversation, error)
	UpdateConversationTeamAssignee(uuid string, teamID int, actor umodels.User) error
	UpdateConversationUserAssignee(uuid string, assigneeID int, actor umodels.User) error
	UpdateConversationStatus(uuid string, status []byte, actor umodels.User) error
	UpdateConversationPriority(uuid string, priority []byte, actor umodels.User) error
}

type queries struct {
	GetAll          *sqlx.Stmt `query:"get-all"`
	GetRule         *sqlx.Stmt `query:"get-rule"`
	InsertRule      *sqlx.Stmt `query:"insert-rule"`
	UpdateRule      *sqlx.Stmt `query:"update-rule"`
	DeleteRule      *sqlx.Stmt `query:"delete-rule"`
	ToggleRule      *sqlx.Stmt `query:"toggle-rule"`
	GetEnabledRules *sqlx.Stmt `query:"get-enabled-rules"`
}

// New initializes a new Engine.
func New(systemUser umodels.User, opt Opts) (*Engine, error) {
	var (
		q queries
		e = &Engine{
			lo:                  opt.Lo,
			newConversationQ:    make(chan string, 5000),
			updateConversationQ: make(chan string, 5000),
			rulesMu:             &sync.RWMutex{},
		}
	)
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	e.q = q
	e.rules = e.queryRules()
	return e, nil
}

// SetConversationStore sets the conversation store.
func (e *Engine) SetConversationStore(store ConversationStore) {
	e.conversationStore = store
}

// ReloadRules reloads automation rules.
func (e *Engine) ReloadRules() {
	e.rulesMu.Lock()
	defer e.rulesMu.Unlock()
	e.lo.Debug("reloading automation engine rules")
	e.rules = e.queryRules()
}

// Run starts the Engine to evaluate rules based on events.
func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Create separate semaphores for each channel.
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

// GetAllRules retrieves all rules of a specific type.
func (e *Engine) GetAllRules(typ []byte) ([]models.RuleRecord, error) {
	var rules = make([]models.RuleRecord, 0)
	if err := e.q.GetAll.Select(&rules, typ); err != nil {
		e.lo.Error("error fetching rules", "error", err)
		return rules, envelope.NewError(envelope.GeneralError, "Error fetching automation rules.", nil)
	}
	return rules, nil
}

// GetRule retrieves a rule by ID.
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

// ToggleRule toggles the active status of a rule by ID.
func (e *Engine) ToggleRule(id int) error {
	if _, err := e.q.ToggleRule.Exec(id); err != nil {
		e.lo.Error("error toggling rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error toggling automation rule.", nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// UpdateRule updates an existing rule.
func (e *Engine) UpdateRule(id int, rule models.RuleRecord) error {
	if _, err := e.q.UpdateRule.Exec(id, rule.Name, rule.Description, rule.Type, rule.Rules); err != nil {
		e.lo.Error("error updating rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating automation rule.", nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// CreateRule creates a new rule.
func (e *Engine) CreateRule(rule models.RuleRecord) error {
	if _, err := e.q.InsertRule.Exec(rule.Name, rule.Description, rule.Type, rule.Rules); err != nil {
		e.lo.Error("error creating rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating automation rule.", nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// DeleteRule deletes a rule by ID.
func (e *Engine) DeleteRule(id int) error {
	if _, err := e.q.DeleteRule.Exec(id); err != nil {
		e.lo.Error("error deleting rule", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting automation rule.", nil)
	}
	// Reload rules.
	e.ReloadRules()
	return nil
}

// handleNewConversation handles new conversation events.
func (e *Engine) handleNewConversation(conversationUUID string, semaphore chan struct{}) {
	defer func() { <-semaphore }()
	conversation, err := e.conversationStore.GetConversation(conversationUUID)
	if err != nil {
		return
	}
	rules := e.filterRulesByType(models.RuleTypeNewConversation)
	e.evalConversationRules(rules, conversation)
}

// handleUpdateConversation handles update conversation events.
func (e *Engine) handleUpdateConversation(conversationUUID string, semaphore chan struct{}) {
	defer func() { <-semaphore }()
	conversation, err := e.conversationStore.GetConversation(conversationUUID)
	if err != nil {
		e.lo.Error("error could not fetch conversations to evaluate update conversation rules", "conversation_uuid", conversationUUID)
		return
	}
	rules := e.filterRulesByType(models.RuleTypeConversationUpdate)
	e.evalConversationRules(rules, conversation)
}

// handleTimeTrigger handles time trigger events.
func (e *Engine) handleTimeTrigger(semaphore chan struct{}) {
	defer func() { <-semaphore }()

	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	conversations, err := e.conversationStore.GetRecentConversations(thirtyDaysAgo)
	if err != nil {
		return
	}
	rules := e.filterRulesByType(models.RuleTypeTimeTrigger)
	for _, conversation := range conversations {
		e.evalConversationRules(rules, conversation)
	}
}

// EvaluateNewConversationRules enqueues a new conversation for rule evaluation.
func (e *Engine) EvaluateNewConversationRules(conversationUUID string) {
	select {
	case e.newConversationQ <- conversationUUID:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateNewConversationRules: newConversationQ is full, unable to enqueue conversation")
	}
}

// EvaluateConversationUpdateRules enqueues an updated conversation for rule evaluation.
func (e *Engine) EvaluateConversationUpdateRules(conversationUUID string) {
	select {
	case e.updateConversationQ <- conversationUUID:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateConversationUpdateRules: updateConversationQ is full, unable to enqueue conversation")
	}
}

// queryRules fetches automation rules from the database.
func (e *Engine) queryRules() []models.Rule {
	var (
		rulesJSON []string
		rules     []models.Rule
	)
	err := e.q.GetEnabledRules.Select(&rulesJSON)
	if err != nil {
		e.lo.Error("error fetching automation rules", "error", err)
		return rules
	}

	e.lo.Debug("fetched rules from db", "count", len(rulesJSON))

	for _, ruleJSON := range rulesJSON {
		var rulesBatch []models.Rule
		if err := json.Unmarshal([]byte(ruleJSON), &rulesBatch); err != nil {
			e.lo.Error("error unmarshalling rule JSON", "error", err)
			continue
		}
		rules = append(rules, rulesBatch...)
	}
	return rules
}

// filterRulesByType filters rules by type.
func (e *Engine) filterRulesByType(ruleType string) []models.Rule {
	e.rulesMu.RLock()
	defer e.rulesMu.RUnlock()

	var filteredRules []models.Rule
	for _, rule := range e.rules {
		if rule.Type == ruleType {
			filteredRules = append(filteredRules, rule)
		}
	}
	return filteredRules
}
