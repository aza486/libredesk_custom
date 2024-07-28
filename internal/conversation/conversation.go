// Package conversation provides functionality to manage conversations in the system.
package conversation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	"github.com/abhinavxd/artemis/internal/stringutil"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

const (
	maxConversationsPageSize = 50
)

// MessageStore interface defines methods to record changes in conversation assignees, priority, and status.
type MessageStore interface {
	RecordAssigneeUserChange(conversationUUID string, assigneeID int, actor umodels.User) error
	RecordAssigneeTeamChange(conversationUUID string, teamID int, actor umodels.User) error
	RecordPriorityChange(priority, conversationUUID string, actor umodels.User) error
	RecordStatusChange(status, conversationUUID string, actor umodels.User) error
}

// Manager handles the operations related to conversations.
type Manager struct {
	lo                  *logf.Logger
	db                  *sqlx.DB
	hub                 *ws.Hub
	i18n                *i18n.I18n
	notifier            notifier.Notifier
	messageStore        MessageStore
	q                   queries
	ReferenceNumPattern string
}

// Opts holds the options for creating a new Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetID                        *sqlx.Stmt `query:"get-id"`
	GetUUID                      *sqlx.Stmt `query:"get-uuid"`
	GetInboxID                   *sqlx.Stmt `query:"get-inbox-id"`
	GetConversation              *sqlx.Stmt `query:"get-conversation"`
	GetRecentConversations       *sqlx.Stmt `query:"get-recent-conversations"`
	GetUnassigned                *sqlx.Stmt `query:"get-unassigned"`
	GetConversations             string     `query:"get-conversations"`
	GetConversationsUUIDs        string     `query:"get-conversations-uuids"`
	GetConversationParticipants  *sqlx.Stmt `query:"get-conversation-participants"`
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

// New initializes a new Manager.
func New(hub *ws.Hub, i18n *i18n.I18n, notifier notifier.Notifier, opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	c := &Manager{
		q:        q,
		hub:      hub,
		i18n:     i18n,
		notifier: notifier,
		db:       opts.DB,
		lo:       opts.Lo,
	}
	return c, nil
}

// SetMessageStore sets the message store for the Manager.
func (c *Manager) SetMessageStore(store MessageStore) {
	c.messageStore = store
}

// Create creates a new conversation and returns its ID and UUID.
func (c *Manager) Create(contactID int, inboxID int, meta []byte) (int, string, error) {
	var (
		id        int
		uuid      string
		refNum, _ = stringutil.RandomNumericString(16)
	)
	if err := c.q.InsertConversation.QueryRow(refNum, contactID, models.StatusOpen, inboxID, meta).Scan(&id, &uuid); err != nil {
		c.lo.Error("error inserting new conversation into the DB", "error", err)
		return id, uuid, err
	}
	return id, uuid, nil
}

// Get retrieves a conversation by its UUID.
func (c *Manager) Get(uuid string) (models.Conversation, error) {
	var conversation models.Conversation
	if err := c.q.GetConversation.Get(&conversation, uuid); err != nil {
		if err == sql.ErrNoRows {
			return conversation, envelope.NewError(envelope.InputError, "Conversation not found.", nil)
		}
		c.lo.Error("error fetching conversation", "error", err)
		return conversation, envelope.NewError(envelope.InputError, "Error fetching conversation.", nil)
	}
	return conversation, nil
}

// GetRecentConversations retrieves conversations created after the specified time.
func (c *Manager) GetRecentConversations(time time.Time) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetRecentConversations.Select(&conversations, time); err != nil {
		if err == sql.ErrNoRows {
			c.lo.Error("conversations not found", "created_after", time)
			return conversations, err
		}
		c.lo.Error("error fetching conversation", "error", err)
		return conversations, err
	}
	return conversations, nil
}

// UpdateAssigneeLastSeen updates the last seen timestamp of an assignee.
func (c *Manager) UpdateAssigneeLastSeen(uuid string) error {
	if _, err := c.q.UpdateAssigneeLastSeen.Exec(uuid); err != nil {
		c.lo.Error("error updating conversation", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating assignee last seen.", nil)
	}
	// Broadcast the property update.
	c.hub.BroadcastConversationPropertyUpdate(uuid, "assignee_last_seen_at", time.Now().Format(time.RFC3339))
	return nil
}

// GetParticipants retrieves the participants of a conversation.
func (c *Manager) GetParticipants(uuid string) ([]models.ConversationParticipant, error) {
	conv := make([]models.ConversationParticipant, 0)
	if err := c.q.GetConversationParticipants.Select(&conv, uuid); err != nil {
		c.lo.Error("error fetching conversation", "error", err)
		return conv, envelope.NewError(envelope.GeneralError, "Error fetching conversation participants", nil)
	}
	return conv, nil
}

// AddParticipant adds a participant to a conversation.
func (c *Manager) AddParticipant(userID int, convUUID string) error {
	if _, err := c.q.InsertConverstionParticipant.Exec(userID, convUUID); err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return nil
		}
		return err
	}
	return nil
}

// UpdateMeta updates the metadata of a conversation.
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

// UpdateLastMessage updates the last message and its timestamp in a conversation.
func (c *Manager) UpdateLastMessage(conversationID int, conversationUUID, lastMessage string, lastMessageAt time.Time) error {
	return c.UpdateMeta(conversationID, conversationUUID, map[string]string{
		"last_message":    lastMessage,
		"last_message_at": lastMessageAt.Format(time.RFC3339),
	})
}

// UpdateFirstReplyAt updates the first reply timestamp in a conversation.
func (c *Manager) UpdateFirstReplyAt(conversationUUID string, conversationID int, at time.Time) error {
	if _, err := c.q.UpdateFirstReplyAt.Exec(conversationID, at); err != nil {
		c.lo.Error("error updating conversation first reply at", "error", err)
		return err
	}
	// Send ws update.
	c.hub.BroadcastConversationPropertyUpdate(conversationUUID, "first_reply_at", time.Now().Format(time.RFC3339))
	return nil
}

// GetUnassigned retrieves unassigned conversations.
func (c *Manager) GetUnassigned() ([]models.Conversation, error) {
	var conv []models.Conversation
	if err := c.q.GetUnassigned.Select(&conv); err != nil {
		if err != sql.ErrNoRows {
			return conv, fmt.Errorf("fetching unassigned conversations: %w", err)
		}
	}
	return conv, nil
}

// GetID retrieves the ID of a conversation by its UUID.
func (c *Manager) GetID(uuid string) (int, error) {
	var id int
	if err := c.q.GetID.QueryRow(uuid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, err
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return id, err
	}
	return id, nil
}

// GetUUID retrieves the UUID of a conversation by its ID.
func (c *Manager) GetUUID(id int) (string, error) {
	var uuid string
	if err := c.q.GetUUID.QueryRow(id).Scan(&uuid); err != nil {
		if err == sql.ErrNoRows {
			return uuid, err
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return uuid, err
	}
	return uuid, nil
}

// GetInboxID retrieves the inbox ID of a conversation by its UUID.
func (c *Manager) GetInboxID(uuid string) (int, error) {
	var id int
	if err := c.q.GetInboxID.QueryRow(uuid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, err
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return id, err
	}
	return id, nil
}

// GetAll retrieves all conversations with optional filtering, ordering, and pagination.
func (c *Manager) GetAll(order, orderBy, filter string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(0, models.AllConversations, order, orderBy, filter, page, pageSize)
}

// GetAssigned retrieves conversations assigned to a specific user with optional filtering, ordering, and pagination.
func (c *Manager) GetAssigned(userID int, order, orderBy, filter string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(userID, models.AssignedConversations, order, orderBy, filter, page, pageSize)
}

// GetTeamConversations retrieves conversations assigned to a team the user is part of with optional filtering, ordering, and pagination.
func (c *Manager) GetTeamConversations(userID int, order, orderBy, filter string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(userID, models.AssigneeTypeTeam, order, orderBy, filter, page, pageSize)
}

// GetConversations retrieves conversations based on user ID, type, and optional filtering, ordering, and pagination.
func (c *Manager) GetConversations(userID int, typ, order, orderBy, filter string, page, pageSize int) ([]models.Conversation, error) {
	var conversations = make([]models.Conversation, 0)

	if orderBy == "" {
		orderBy = "last_message_at"
	}
	if order == "" {
		order = "DESC"
	}

	query, qArgs, err := c.generateConversationsListQuery(userID, c.q.GetConversations, typ, order, orderBy, filter, page, pageSize)
	if err != nil {
		c.lo.Error("error generating conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.conversations}"), nil)
	}

	tx, err := c.db.BeginTxx(context.Background(), nil)
	defer tx.Rollback()
	if err != nil {
		c.lo.Error("error preparing get conversations query", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.conversations}"), nil)
	}

	if err := tx.Select(&conversations, query, qArgs...); err != nil {
		c.lo.Error("error fetching conversations", "error", err)
		return conversations, envelope.NewError(envelope.GeneralError, c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.conversations}"), nil)
	}
	return conversations, nil
}

// GetConversationUUIDs retrieves the UUIDs of conversations based on user ID, type, and optional filtering, ordering, and pagination.
func (c *Manager) GetConversationUUIDs(userID, page, pageSize int, typ, filter string) ([]string, error) {
	var ids = make([]string, 0)

	query, qArgs, err := c.generateConversationsListQuery(userID, c.q.GetConversationsUUIDs, typ, "", "", filter, page, pageSize)
	if err != nil {
		c.lo.Error("error generating conversations query", "error", err)
		return ids, err
	}

	tx, err := c.db.BeginTxx(context.Background(), nil)
	defer tx.Rollback()
	if err != nil {
		c.lo.Error("error preparing get conversation ids query", "error", err)
		return ids, err
	}

	if err := tx.Select(&ids, query, qArgs...); err != nil {
		c.lo.Error("error fetching conversation uuids", "error", err)
		return ids, err
	}
	return ids, nil
}

// GetAssignedConversations retrieves conversations assigned to a specific user.
func (c *Manager) GetAssignedConversations(userID int) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := c.q.GetAssignedConversations.Select(&conversations, userID); err != nil {
		c.lo.Error("fetching conversation from DB", "error", err)
		return nil, fmt.Errorf("error fetching conversations")
	}
	return conversations, nil
}

// UpdateUserAssignee updates the assignee of a conversation to a specific user.
func (c *Manager) UpdateUserAssignee(uuid string, assigneeID int, actor umodels.User) error {
	if err := c.UpdateAssignee(uuid, assigneeID, models.AssigneeTypeUser); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error updating assignee", nil)
	}

	// Send notification to assignee.
	go c.notifier.SendAssignedConversationNotification([]int{assigneeID}, uuid)

	if err := c.messageStore.RecordAssigneeUserChange(uuid, assigneeID, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording assignee change", nil)
	}
	return nil
}

// UpdateTeamAssignee updates the assignee of a conversation to a specific team.
func (c *Manager) UpdateTeamAssignee(uuid string, teamID int, actor umodels.User) error {
	if err := c.UpdateAssignee(uuid, teamID, models.AssigneeTypeTeam); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error updating assignee", nil)
	}
	if err := c.messageStore.RecordAssigneeTeamChange(uuid, teamID, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording assignee change", nil)
	}
	return nil
}

// UpdateAssignee updates the assignee of a conversation.
func (c *Manager) UpdateAssignee(uuid string, assigneeID int, assigneeType string) error {
	switch assigneeType {
	case models.AssigneeTypeUser:
		if _, err := c.q.UpdateAssignedUser.Exec(uuid, assigneeID); err != nil {
			c.lo.Error("error updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}
		c.hub.BroadcastConversationPropertyUpdate(uuid, "assigned_user_id", strconv.Itoa(assigneeID))
	case models.AssigneeTypeTeam:
		if _, err := c.q.UpdateAssignedTeam.Exec(uuid, assigneeID); err != nil {
			c.lo.Error("error updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}
		c.hub.BroadcastConversationPropertyUpdate(uuid, "assigned_team_id", strconv.Itoa(assigneeID))
	default:
		return errors.New("invalid assignee type")
	}
	return nil
}

// UpdatePriority updates the priority of a conversation.
func (c *Manager) UpdatePriority(uuid string, priority []byte, actor umodels.User) error {
	var priorityStr = string(priority)
	if !slices.Contains(models.ValidPriorities, priorityStr) {
		return envelope.NewError(envelope.GeneralError, "Invalid `priority` value", nil)
	}
	if _, err := c.q.UpdatePriority.Exec(uuid, priority); err != nil {
		c.lo.Error("error updating conversation priority", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating priority", nil)
	}
	if err := c.messageStore.RecordPriorityChange(priorityStr, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording priority change", nil)
	}
	return nil
}

// UpdateStatus updates the status of a conversation.
func (c *Manager) UpdateStatus(uuid string, status []byte, actor umodels.User) error {
	var statusStr = string(status)
	if !slices.Contains(models.ValidStatuses, statusStr) {
		return envelope.NewError(envelope.GeneralError, "Invalid `status` value", nil)
	}
	if _, err := c.q.UpdateStatus.Exec(uuid, status); err != nil {
		c.lo.Error("error updating conversation status", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating status", nil)
	}
	if err := c.messageStore.RecordStatusChange(statusStr, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording status change", nil)
	}
	return nil
}

// GetAssigneeStats retrieves the statistics of conversations assigned to a specific user.
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

// GetNewConversationsStats retrieves the statistics of new conversations.
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

// UpsertTags updates the tags associated with a conversation.
func (t *Manager) UpsertTags(uuid string, tagIDs []int) error {
	if _, err := t.q.DeleteTags.Exec(uuid, pq.Array(tagIDs)); err != nil {
		t.lo.Error("error deleting conversation tags", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error adding tags", nil)
	}
	for _, tagID := range tagIDs {
		if _, err := t.q.AddTag.Exec(uuid, tagID); err != nil {
			t.lo.Error("error adding tags to conversation", "error", err)
			return envelope.NewError(envelope.GeneralError, "Error adding tags", nil)
		}
	}
	return nil
}

// generateConversationsListQuery generates the SQL query to list conversations with optional filtering, ordering, and pagination.
func (c *Manager) generateConversationsListQuery(userID int, baseQuery, typ, order, orderBy, filter string, page, pageSize int) (string, []interface{}, error) {
	var (
		qArgs []interface{}
		cond  string
	)

	// Set condition based on the type.
	switch typ {
	case models.AssignedConversations:
		cond = "AND c.assigned_user_id = $1"
		qArgs = append(qArgs, userID)
	case models.TeamConversations:
		cond = "AND c.assigned_user_id IS NULL AND c.assigned_team_id IN (SELECT team_id FROM team_members WHERE user_id = $1)"
		qArgs = append(qArgs, userID)
	case models.AllConversations:
		// No conditions.
	default:
		return "", nil, errors.New("invalid type of conversation")
	}

	if filterClause, ok := models.ValidFilters[filter]; ok {
		cond += " AND " + filterClause
	}

	// Ensure orderBy & order is valid.
	var orderByClause = ""
	if slices.Contains(models.ValidOrderBy, orderBy) {
		orderByClause = fmt.Sprintf(" ORDER BY %s ", orderBy)
	}
	if orderByClause > "" && slices.Contains(models.ValidOrder, order) {
		orderByClause += order
	}

	// Calculate offset based on page number and page size.
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > maxConversationsPageSize {
		pageSize = maxConversationsPageSize
	}
	offset := (page - 1) * pageSize
	qArgs = append(qArgs, pageSize, offset)

	// Include LIMIT, OFFSET, and ORDER BY in the SQL query.
	sqlQuery := fmt.Sprintf("%s %s LIMIT $%d OFFSET $%d", fmt.Sprintf(baseQuery, cond), orderByClause, len(qArgs)-1, len(qArgs))

	return sqlQuery, qArgs, nil
}
