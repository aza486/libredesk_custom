package automation

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/artemis/internal/automation/models"
	"github.com/abhinavxd/artemis/internal/conversation"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type queries struct {
	GetNewConversationRules *sqlx.Stmt `query:"get-rules"`
	GetRuleActions          *sqlx.Stmt `query:"get-rule-actions"`
}

type Engine struct {
	q                queries
	lo               *logf.Logger
	convMgr          *conversation.Manager
	newConversationQ chan cmodels.Conversation
	rules            []models.Rule
	actions          []models.Action
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

func New(convMgr *conversation.Manager, opt Opts) (*Engine, error) {
	var (
		q queries
		e = &Engine{
			lo:               opt.Lo,
			convMgr:          convMgr,
			newConversationQ: make(chan cmodels.Conversation, 10000),
		}
	)

	if err := dbutils.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}

	// Fetch applicable rules & actions.
	if err := q.GetNewConversationRules.Select(&e.rules); err != nil {
		return nil, fmt.Errorf("fetching rules: %w", err)
	}
	if err := q.GetRuleActions.Select(&e.actions); err != nil {
		return nil, fmt.Errorf("fetching rule actions: %w", err)
	}
	e.q = q

	return e, nil
}

func (e *Engine) Serve() {
	for conv := range e.newConversationQ {
		e.processConversations(conv)
	}
}

func (e *Engine) ProcessConversation(c cmodels.Conversation) {
	e.newConversationQ <- c
}
