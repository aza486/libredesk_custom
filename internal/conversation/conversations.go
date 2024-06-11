package conversation

import (
	"database/sql"
	"embed"
	"fmt"
	"slices"

	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/abhinavxd/artemis/internal/stringutils"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	StatusOpen = "Open"

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

type Manager struct {
	lo                  *logf.Logger
	q                   queries
	ReferenceNumPattern string
}

type Opts struct {
	DB                  *sqlx.DB
	Lo                  *logf.Logger
	ReferenceNumPattern string
}

type queries struct {
	GetID                        *sqlx.Stmt `query:"get-id"`
	GetUUID                      *sqlx.Stmt `query:"get-uuid"`
	GetInboxID                   *sqlx.Stmt `query:"get-inbox-id"`
	GetConversation              *sqlx.Stmt `query:"get-conversation"`
	GetUnassigned                *sqlx.Stmt `query:"get-unassigned"`
	GetConversationParticipants  *sqlx.Stmt `query:"get-conversation-participants"`
	GetConversations             *sqlx.Stmt `query:"get-conversations"`
	GetAssignedConversations     *sqlx.Stmt `query:"get-assigned-conversations"`
	InsertConverstionParticipant *sqlx.Stmt `query:"insert-conversation-participant"`
	InsertConversation           *sqlx.Stmt `query:"insert-conversation"`
	UpdateAssigneeLastSeen       *sqlx.Stmt `query:"update-assignee-last-seen"`
	UpdateAssignedUser           *sqlx.Stmt `query:"update-assigned-user"`
	UpdateAssignedTeam           *sqlx.Stmt `query:"update-assigned-team"`
	UpdatePriority               *sqlx.Stmt `query:"update-priority"`
	UpdateStatus                 *sqlx.Stmt `query:"update-status"`
}

func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	c := &Manager{
		q:                   q,
		lo:                  opts.Lo,
		ReferenceNumPattern: opts.ReferenceNumPattern,
	}

	return c, nil
}

func (c *Manager) Create(contactID int, inboxID int, meta string) (int, error) {
	var (
		id        int
		refNum, _ = c.generateRefNum(c.ReferenceNumPattern)
	)
	if err := c.q.InsertConversation.QueryRow(refNum, contactID, StatusOpen, inboxID, meta).Scan(&id); err != nil {
		c.lo.Error("inserting new conversation into the DB", "error", err)
		return id, err
	}
	return id, nil
}

func (c *Manager) Get(uuid string) (models.Conversation, error) {
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

func (c *Manager) UpdateAssigneeLastSeen(uuid string) error {
	if _, err := c.q.UpdateAssigneeLastSeen.Exec(uuid); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return fmt.Errorf("error updating conversation last seen: %w", err)
	}
	return nil
}

func (c *Manager) GetParticipants(uuid string) ([]models.ConversationParticipant, error) {
	conv := make([]models.ConversationParticipant, 0)
	if err := c.q.GetConversationParticipants.Select(&conv, uuid); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return conv, fmt.Errorf("error fetching conversation")
	}
	return conv, nil
}

func (c *Manager) AddParticipant(userID int, convUUID string) error {
	if _, err := c.q.InsertConverstionParticipant.Exec(userID, convUUID); err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return nil
		}
		return err
	}
	return nil
}

func (c *Manager) GetUnassigned() ([]models.Conversation, error) {
	var conv []models.Conversation
	if err := c.q.GetUnassigned.Get(&conv); err != nil {
		if err != sql.ErrNoRows {
			return conv, fmt.Errorf("conversation not found")
		}
	}
	return conv, nil
}

func (c *Manager) GetID(uuid string) (int, error) {
	var id int
	if err := c.q.GetID.QueryRow(uuid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, fmt.Errorf("conversation not found: %w", err)
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return id, fmt.Errorf("error fetching conversation: %w", err)
	}
	return id, nil
}

func (c *Manager) GetUUID(id int) (string, error) {
	var uuid string
	if err := c.q.GetUUID.QueryRow(id).Scan(&uuid); err != nil {
		if err == sql.ErrNoRows {
			return uuid, fmt.Errorf("conversation not found: %w", err)
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return uuid, fmt.Errorf("error fetching conversation: %w", err)
	}
	return uuid, nil
}

func (c *Manager) GetInboxID(uuid string) (int, error) {
	var id int
	if err := c.q.GetInboxID.QueryRow(uuid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, fmt.Errorf("conversation not found: %w", err)
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return id, fmt.Errorf("error fetching conversation: %w", err)
	}
	return id, nil
}

func (c *Manager) GetConversations() ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetConversations.Select(&conversations); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return nil, fmt.Errorf("error fetching conversations")
	}
	return conversations, nil
}

func (c *Manager) GetAssignedConversations(userID int) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetAssignedConversations.Select(&conversations, userID); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return nil, fmt.Errorf("error fetching conversations")
	}
	return conversations, nil
}

func (c *Manager) UpdateAssignee(uuid string, assigneeUUID []byte, assigneeType string) error {
	if assigneeType == "agent" {
		if _, err := c.q.UpdateAssignedUser.Exec(uuid, assigneeUUID); err != nil {
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

func (c *Manager) UpdatePriority(uuid string, priority []byte) error {
	if !slices.Contains(priorities, string(priority)) {
		return fmt.Errorf("invalid `priority` value %s", priority)
	}
	if _, err := c.q.UpdatePriority.Exec(uuid, priority); err != nil {
		c.lo.Error("updating conversation priority", "error", err)
		return fmt.Errorf("error updating priority")
	}
	return nil
}

func (c *Manager) UpdateStatus(uuid string, status []byte) error {
	if !slices.Contains(statuses, string(status)) {
		return fmt.Errorf("invalid `status` value %s", status)
	}
	if _, err := c.q.UpdateStatus.Exec(uuid, status); err != nil {
		c.lo.Error("updating conversation status", "error", err)
		return fmt.Errorf("error updating status")
	}
	return nil
}

func (c *Manager) generateRefNum(pattern string) (string, error) {
	if len(pattern) <= 5 {
		pattern = "01234567890"
	}
	randomNumbers, err := stringutils.RandomNumericString(len(pattern))
	if err != nil {
		return "", err
	}
	result := []byte(pattern)
	randomIndex := 0
	for i := range result {
		if result[i] == '#' {
			result[i] = randomNumbers[randomIndex]
			randomIndex++
		}
	}
	return string(result), nil
}
