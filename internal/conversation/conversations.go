package conversation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/stringutil"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	StatusOpen       = "Open"
	StatusResolved   = "Resolved"
	StatusProcessing = "Processing"
	StatusSpam       = "Spam"

	PriorityLow    = "Low"
	PriortiyMedium = "Medium"
	PriorityHigh   = "High"

	statuses = []string{
		StatusOpen,
		StatusResolved,
		StatusProcessing,
		StatusSpam,
	}
	priorities = []string{
		PriorityLow,
		PriortiyMedium,
		PriorityHigh,
	}

	preDefinedFilters = map[string]string{
		"status_open":       " c.status = 'Open'",
		"status_processing": " c.status = 'Processing'",
		"status_spam":       " c.status = 'Spam'",
		"status_resolved":   " c.status = 'Resolved'",
		"status_all":        " 1=1  ",
	}

	assigneeTypeTeam = "team"
	assigneeTypeUser = "user"
)

type Manager struct {
	lo                  *logf.Logger
	db                  *sqlx.DB
	hub                 *ws.Hub
	i18n                *i18n.I18n
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
	GetRecentConversations       *sqlx.Stmt `query:"get-recent-conversations"`
	GetUnassigned                *sqlx.Stmt `query:"get-unassigned"`
	GetConversationParticipants  *sqlx.Stmt `query:"get-conversation-participants"`
	GetConversations             string     `query:"get-conversations"`
	GetConversationsUUIDs        string     `query:"get-conversations-uuids"`
	GetAssignedConversations     *sqlx.Stmt `query:"get-assigned-conversations"`
	GetAssigneeStats             *sqlx.Stmt `query:"get-assignee-stats"`
	GetNewConversationsStats     *sqlx.Stmt `query:"get-new-conversations-stats"`
	InsertConverstionParticipant *sqlx.Stmt `query:"insert-conversation-participant"`
	InsertConversation           *sqlx.Stmt `query:"insert-conversation"`
	UpdateFirstReplyAt           *sqlx.Stmt `query:"update-first-reply-at"`
	UpdateAssigneeLastSeen       *sqlx.Stmt `query:"update-assignee-last-seen"`
	UpdateAssignedUser           *sqlx.Stmt `query:"update-assigned-user"`
	UpdateAssignedTeam           *sqlx.Stmt `query:"update-assigned-team"`
	UpdatePriority               *sqlx.Stmt `query:"update-priority"`
	UpdateStatus                 *sqlx.Stmt `query:"update-status"`
	UpdateMeta                   *sqlx.Stmt `query:"update-meta"`
	AddTag                       *sqlx.Stmt `query:"add-tag"`
	DeleteTags                   *sqlx.Stmt `query:"delete-tags"`
}

func New(hub *ws.Hub, i18n *i18n.I18n, opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	c := &Manager{
		q:                   q,
		hub:                 hub,
		i18n:                i18n,
		db:                  opts.DB,
		lo:                  opts.Lo,
		ReferenceNumPattern: opts.ReferenceNumPattern,
	}
	return c, nil
}

func (c *Manager) Create(contactID int, inboxID int, meta []byte) (int, string, error) {
	var (
		id        int
		uuid      string
		refNum, _ = c.generateRefNum(c.ReferenceNumPattern)
	)
	if err := c.q.InsertConversation.QueryRow(refNum, contactID, StatusOpen, inboxID, meta).Scan(&id, &uuid); err != nil {
		c.lo.Error("error inserting new conversation into the DB", "error", err)
		return id, uuid, err
	}
	return id, uuid, nil
}

func (c *Manager) Get(uuid string) (models.Conversation, error) {
	var conversation models.Conversation
	if err := c.q.GetConversation.Get(&conversation, uuid); err != nil {
		if err == sql.ErrNoRows {
			c.lo.Error("conversation not found", "uuid", uuid)
			return conversation, fmt.Errorf("conversation not found")
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return conversation, fmt.Errorf("error fetching conversation")
	}
	return conversation, nil
}

func (c *Manager) GetRecentConversations(time time.Time) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetRecentConversations.Select(&conversations, time); err != nil {
		if err == sql.ErrNoRows {
			c.lo.Error("conversations not found", "created_after", time)
			return conversations, fmt.Errorf("conversation not found")
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return conversations, fmt.Errorf("error fetching conversation")
	}
	return conversations, nil
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

func (c *Manager) UpdateMeta(conversationID int, conversationUUID string, meta map[string]string) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		c.lo.Error("error marshalling meta", "error", err)
		return err
	}
	if _, err := c.q.UpdateMeta.Exec(conversationID, conversationUUID, metaJSON); err != nil {
		c.lo.Error("error updating conversation meta", "error", "error")
		return err
	}
	return nil
}

func (c *Manager) UpdateLastMessage(conversationID int, conversationUUID, lastMessage string, lastMessageAt time.Time) error {
	return c.UpdateMeta(conversationID, conversationUUID, map[string]string{
		"last_message":    lastMessage,
		"last_message_at": lastMessageAt.Format(time.RFC3339),
	})
}

func (c *Manager) UpdateFirstReplyAt(conversationUUID string, conversationID int, at time.Time) error {
	if _, err := c.q.UpdateFirstReplyAt.Exec(conversationID, at); err != nil {
		c.lo.Error("error updating conversation first reply at", "error", err)
		return err
	}
	// Send ws update.
	c.hub.BroadcastConversationPropertyUpdate(conversationUUID, "first_reply_at", time.Now().Format(time.RFC3339))
	return nil
}

func (c *Manager) GetUnassigned() ([]models.Conversation, error) {
	var conv []models.Conversation
	if err := c.q.GetUnassigned.Select(&conv); err != nil {
		if err != sql.ErrNoRows {
			return conv, fmt.Errorf("fetching unassigned converastions: %w", err)
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

func (c *Manager) GetConversations(userID int, typ, order, orderBy, predefinedFilter string, page, pageSize int) ([]models.Conversation, error) {
	var (
		conversations []models.Conversation
		qArgs         []interface{}
		cond          string
		validOrderBy  = map[string]bool{"created_at": true, "priority": true, "status": true, "last_message_at": true}
		validOrder    = []string{"ASC", "DESC"}
	)

	switch typ {
	case "assigned":
		cond = "AND c.assigned_user_id = $1"
		qArgs = append(qArgs, userID)
	case "unassigned":
		cond = "AND c.assigned_user_id IS NULL AND c.assigned_team_id IN (SELECT team_id FROM team_members WHERE user_id = $1)"
		qArgs = append(qArgs, userID)
	case "all":
	default:
		return conversations, errors.New("invalid type")
	}

	if filterClause, ok := preDefinedFilters[predefinedFilter]; ok {
		cond += " AND " + filterClause
	}

	// Ensure orderBy is valid.
	orderByClause := ""
	if _, ok := validOrderBy[orderBy]; ok {
		orderByClause = fmt.Sprintf(" ORDER BY %s", orderBy)
	} else {
		orderByClause = " ORDER BY last_message_at"
	}

	if slices.Contains(validOrder, order) {
		orderByClause += " " + order
	} else {
		orderByClause += " DESC "
	}

	// Calculate offset based on page number and page size.
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 20 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	qArgs = append(qArgs, pageSize, offset)

	tx, err := c.db.BeginTxx(context.Background(), nil)
	defer tx.Rollback()
	if err != nil {
		c.lo.Error("error preparing get conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.conversations}"), nil)
	}

	// Include LIMIT, OFFSET, and ORDER BY in the SQL query.
	sqlQuery := fmt.Sprintf("%s %s LIMIT $%d OFFSET $%d", fmt.Sprintf(c.q.GetConversations, cond), orderByClause, len(qArgs)-1, len(qArgs))
	if err := tx.Select(&conversations, sqlQuery, qArgs...); err != nil {
		c.lo.Error("error fetching conversations", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.conversations}"), nil)
	}

	return conversations, nil
}

func (c *Manager) GetConversationUUIDs(userID, page, pageSize int, typ, predefinedFilter string) ([]string, error) {
	var (
		conversationUUIDs []string
		qArgs             []interface{}
		cond              string
	)
	switch typ {
	case "assigned":
		cond = "AND c.assigned_user_id = $1"
		qArgs = append(qArgs, userID)
	case "unassigned":
		cond = "AND c.assigned_user_id IS NULL AND c.assigned_team_id IN (SELECT team_id FROM team_members WHERE user_id = $1)"
		qArgs = append(qArgs, userID)
	case "all":
	default:
		return conversationUUIDs, errors.New("invalid type")
	}

	if filterClause, ok := preDefinedFilters[predefinedFilter]; ok {
		cond += " AND " + filterClause
	}

	tx, err := c.db.BeginTxx(context.Background(), nil)
	defer tx.Rollback()
	if err != nil {
		c.lo.Error("Error preparing get conversation ids query", "error", err)
		return conversationUUIDs, err
	}

	// Calculate offset based on page number and page size.
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 20 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	qArgs = append(qArgs, pageSize, offset)

	// Include LIMIT, OFFSET, and ORDER BY in the SQL query.
	sqlQuery := fmt.Sprintf("%s LIMIT $%d OFFSET $%d", fmt.Sprintf(c.q.GetConversationsUUIDs, cond), len(qArgs)-1, len(qArgs))
	if err := tx.Select(&conversationUUIDs, sqlQuery, qArgs...); err != nil {
		c.lo.Error("Error fetching conversations", "error", err)
		return conversationUUIDs, err
	}
	return conversationUUIDs, nil
}

func (c *Manager) GetAssignedConversations(userID int) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetAssignedConversations.Select(&conversations, userID); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return nil, fmt.Errorf("error fetching conversations")
	}
	return conversations, nil
}

func (c *Manager) UpdateTeamAssignee(uuid string, assigneeUUID []byte) error {
	return c.UpdateAssignee(uuid, assigneeUUID, assigneeTypeTeam)
}

func (c *Manager) UpdateUserAssignee(uuid string, assigneeUUID []byte) error {
	return c.UpdateAssignee(uuid, assigneeUUID, assigneeTypeUser)
}

func (c *Manager) UpdateAssignee(uuid string, assigneeUUID []byte, assigneeType string) error {
	switch assigneeType {
	case assigneeTypeUser:
		if _, err := c.q.UpdateAssignedUser.Exec(uuid, assigneeUUID); err != nil {
			c.lo.Error("updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}
		c.hub.BroadcastConversationPropertyUpdate(uuid, "assigned_user_uuid", string(assigneeUUID))
	case assigneeTypeTeam:
		if _, err := c.q.UpdateAssignedTeam.Exec(uuid, assigneeUUID); err != nil {
			c.lo.Error("updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}
		c.hub.BroadcastConversationPropertyUpdate(uuid, "assigned_team_uuid", string(assigneeUUID))
	default:
		return errors.New("invalid assignee type")
	}
	return nil
}

func (c *Manager) UpdatePriority(uuid string, priority []byte) error {
	var priorityStr = string(priority)
	if !slices.Contains(priorities, string(priorityStr)) {
		return fmt.Errorf("invalid `priority` value %s", priorityStr)
	}
	if _, err := c.q.UpdatePriority.Exec(uuid, priority); err != nil {
		c.lo.Error("updating conversation priority", "error", err)
		return fmt.Errorf("error updating priority")
	}
	c.hub.BroadcastConversationPropertyUpdate(uuid, "priority", priorityStr)
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
	c.hub.BroadcastConversationPropertyUpdate(uuid, "status", string(status))
	return nil
}

func (c *Manager) GetAssigneeStats(userID int) (models.ConversationCounts, error) {
	var counts = models.ConversationCounts{}
	if err := c.q.GetAssigneeStats.Get(&counts, userID); err != nil {
		if err == sql.ErrNoRows {
			return counts, err
		}
		c.lo.Error("error fetching assignee conversation stats", "user_id", userID, "error", err)
		return counts, err
	}
	return counts, nil
}

func (c *Manager) GetNewConversationsStats() ([]models.NewConversationsStats, error) {
	var stats []models.NewConversationsStats
	if err := c.q.GetNewConversationsStats.Select(&stats); err != nil {
		if err == sql.ErrNoRows {
			return stats, err
		}
		c.lo.Error("error fetching new conversation stats", "error", err)
		return stats, err
	}
	return stats, nil
}

func (t *Manager) AddTags(convUUID string, tagIDs []int) error {
	// Delete tags that have been removed.
	if _, err := t.q.DeleteTags.Exec(convUUID, pq.Array(tagIDs)); err != nil {
		t.lo.Error("error deleting conversation tags", "error", err)
	}

	// Add new tags.
	for _, tagID := range tagIDs {
		if _, err := t.q.AddTag.Exec(convUUID, tagID); err != nil {
			t.lo.Error("error adding tags to conversation", "error", err)
		}
	}
	return nil
}

func (c *Manager) generateRefNum(pattern string) (string, error) {
	if len(pattern) <= 5 {
		pattern = "01234567890"
	}
	randomNumbers, err := stringutil.RandomNumericString(len(pattern))
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
