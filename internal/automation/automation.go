package automation

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/artemis/internal/automation/models"
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
	q             queries
	lo            *logf.Logger
	convUpdater   ConversationUpdater
	msgRecorder   MessageRecorder
	conversationQ chan cmodels.Conversation
	rules         []models.Rule
	actions       []models.Action
}

type ConversationUpdater interface {
	UpdateAssignee(uuid string, assigneeUUID []byte, assigneeType string) error
	UpdateStatus(uuid string, status []byte) error
}

type MessageRecorder interface {
	RecordAssigneeUserChange(updatedValue, convUUID, actorUUID string) error
	RecordStatusChange(updatedValue, convUUID, actorUUID string) error
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

func New(opt Opts) (*Engine, error) {
	var (
		q queries
		e = &Engine{
			lo:            opt.Lo,
			conversationQ: make(chan cmodels.Conversation, 10000),
		}
	)

	if err := dbutils.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}

	// Fetch rules and actions from the DB.
	if err := q.GetNewConversationRules.Select(&e.rules); err != nil {
		return nil, fmt.Errorf("fetching rules: %w", err)
	}
	if err := q.GetRuleActions.Select(&e.actions); err != nil {
		return nil, fmt.Errorf("fetching rule actions: %w", err)
	}
	e.q = q

	return e, nil
}


func (e *Engine) SetMsgRecorder(msgRecorder MessageRecorder) {
	e.msgRecorder = msgRecorder
}

func (e *Engine) SetConvUpdater(convUpdater ConversationUpdater) {
	e.convUpdater = convUpdater
}

func (e *Engine) Serve() {
	for conv := range e.conversationQ {
		e.processConversations(conv)
	}
}

func (e *Engine) EvaluateRules(c cmodels.Conversation) {
	e.conversationQ <- c
}
