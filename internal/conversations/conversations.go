package conversations

import (
	"database/sql"
	"embed"
	"fmt"
	"slices"

	"github.com/abhinavxd/artemis/internal/conversations/models"
	"github.com/abhinavxd/artemis/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	statuses = []string{
		"Open",
		"Resolved",
		"Processing",
		"Spam",
	}
	priorities = []string{
		"Low",
		"Medium",
		"High",
	}
)

type Conversations struct {
	lo *logf.Logger
	q  queries
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetConversation     *sqlx.Stmt `query:"get-conversation"`
	GetConversations    *sqlx.Stmt `query:"get-conversations"`
	GetMessages         *sqlx.Stmt `query:"get-messages"`
	UpdateAssignedAgent *sqlx.Stmt `query:"update-assigned-agent"`
	UpdateAssignedTeam  *sqlx.Stmt `query:"update-assigned-team"`
	UpdatePriority      *sqlx.Stmt `query:"update-priority"`
	UpdateStatus        *sqlx.Stmt `query:"update-status"`
}

func New(opts Opts) (*Conversations, error) {
	c := &Conversations{
		q:  queries{},
		lo: opts.Lo,
	}
	if err := utils.ScanSQLFile("queries.sql", &c.q, opts.DB, efs); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Conversations) GetConversation(uuid string) (models.Conversation, error) {
	var conv models.Conversation
	if err := c.q.GetConversation.Get(&conv, uuid); err != nil {
		if err == sql.ErrNoRows {
			return conv, fmt.Errorf("conversation not found")
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return conv, fmt.Errorf("error fetching conversation")
	}
	return conv, nil
}

func (c *Conversations) GetConversations() ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetConversations.Select(&conversations); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return nil, fmt.Errorf("error fetching conversations")
	}
	return conversations, nil
}

func (c *Conversations) GetMessages(uuid string) ([]models.Message, error) {
	var messages []models.Message
	if err := c.q.GetMessages.Select(&messages, uuid); err != nil {
		c.lo.Error("fetching messages from DB", "conversation_uuid", uuid, "error", err)
		return nil, fmt.Errorf("error fetching messages")
	}
	return messages, nil
}

func (c *Conversations) UpdateAssignee(uuid string, assigneeUUID []byte, assigneeType string) error {
	if assigneeType == "agent" {
		if _, err := c.q.UpdateAssignedAgent.Exec(uuid, assigneeUUID); err != nil {
			c.lo.Error("updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}
	}
	if assigneeType == "team" {
		if _, err := c.q.UpdateAssignedTeam.Exec(uuid, assigneeUUID); err != nil {
			c.lo.Error("updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}
	}
	return nil
}

func (c *Conversations) UpdatePriority(uuid string, priority []byte) error {
	if !slices.Contains(priorities, string(priority)) {
		return fmt.Errorf("invalid `priority` value %s", priority)
	}
	if _, err := c.q.UpdatePriority.Exec(uuid, priority); err != nil {
		c.lo.Error("updating conversation priority", "error", err)
		return fmt.Errorf("error updating priority")
	}
	return nil
}

func (c *Conversations) UpdateStatus(uuid string, status []byte) error {
	if !slices.Contains(statuses, string(status)) {
		return fmt.Errorf("invalid `priority` value %s", status)
	}
	if _, err := c.q.UpdateStatus.Exec(uuid, status); err != nil {
		c.lo.Error("updating conversation status", "error", err)
		return fmt.Errorf("error updating status")
	}
	return nil
}
