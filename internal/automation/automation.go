package automation

import (
	"context"
	"embed"
	"encoding/json"

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
	q                 queries
	lo                *logf.Logger
	conversationStore ConversationStore
	messageStore      MessageStore
	rules             []models.Rule
	conversationQ     chan cmodels.Conversation
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type ConversationStore interface {
	UpdateTeamAssignee(uuid string, assigneeUUID []byte) error
	UpdateStatus(uuid string, status []byte) error
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
			lo:            opt.Lo,
			conversationQ: make(chan cmodels.Conversation, 10000),
		}
	)
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	e.q = q
	e.rules = e.getRules()
	return e, nil
}

func (e *Engine) ReloadRules() {
	e.rules = e.getRules()
}

func (e *Engine) SetMessageStore(messageStore MessageStore) {
	e.messageStore = messageStore
}

func (e *Engine) SetConversationStore(conversationStore ConversationStore) {
	e.conversationStore = conversationStore
}

func (e *Engine) Serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case conversation := <-e.conversationQ:
			e.processConversations(conversation)
		}
	}
}

func (e *Engine) EvaluateRules(c cmodels.Conversation) {
	select {
	case e.conversationQ <- c:
	default:
		// Queue is full.
		e.lo.Warn("EvaluateRules: conversationQ is full, unable to enqueue conversation")
	}
}

func (e *Engine) getRules() []models.Rule {
	var rulesJSON []string
	err := e.q.GetRules.Select(&rulesJSON)
	if err != nil {
		e.lo.Error("error fetching automation rules", "error", err)
		return nil
	}

	var rules []models.Rule
	for _, ruleJSON := range rulesJSON {
		var rulesBatch []models.Rule
		if err := json.Unmarshal([]byte(ruleJSON), &rulesBatch); err != nil {
			e.lo.Error("error unmarshalling rule JSON", "error", err)
			continue
		}
		rules = append(rules, rulesBatch...)
	}

	e.lo.Debug("fetched rules", "num", len(rules), "rules", rules)
	return rules
}
