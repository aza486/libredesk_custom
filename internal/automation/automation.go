package automation

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
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
	messageStore        MessageStore
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
	UpdateTeamAssignee(uuid string, assigneeUUID []byte) error
	UpdateUserAssignee(uuid string, assigneeUUID []byte) error
	UpdateStatus(uuid string, status []byte) error
	UpdatePriority(uuid string, priority []byte) error
}

type MessageStore interface {
	RecordAssigneeTeamChange(convUUID, value, actorUUID string) error
	RecordStatusChange(updatedValue, convUUID, actorUUID string) error
}

type queries struct {
	GetRules *sqlx.Stmt `query:"get-rules"`
}

func New(opt Opts) (*Engine, error) {
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

func (e *Engine) SetMessageStore(messageStore MessageStore) {
	e.messageStore = messageStore
}

func (e *Engine) SetConversationStore(conversationStore ConversationStore) {
	e.conversationStore = conversationStore
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
