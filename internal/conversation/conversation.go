// Package conversation provides functionality to manage conversations in the system.
package conversation

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/inbox"
	mmodels "github.com/abhinavxd/artemis/internal/media/models"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	tmodels "github.com/abhinavxd/artemis/internal/team/models"
	"github.com/abhinavxd/artemis/internal/template"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs                                  embed.FS
	ErrConversationNotFound              = errors.New("conversation not found")
	ConversationsListAllowedFilterFields = []string{"status_id", "priority_id", "assigned_team_id", "assigned_user_id"}
	ConversationStatusesFilterFields     = []string{"id", "name"}
)

const (
	conversationsListMaxPageSize = 50
)

// Manager handles the operations related to conversations
type Manager struct {
	q                          queries
	inboxStore                 inboxStore
	userStore                  userStore
	teamStore                  teamStore
	mediaStore                 mediaStore
	notifier                   *notifier.Service
	lo                         *logf.Logger
	db                         *sqlx.DB
	i18n                       *i18n.I18n
	automation                 *automation.Engine
	wsHub                      *ws.Hub
	template                   *template.Manager
	incomingMessageQueue       chan models.IncomingMessage
	outgoingMessageQueue       chan models.Message
	outgoingProcessingMessages sync.Map
	closed                     bool
	closedMu                   sync.RWMutex
	wg                         sync.WaitGroup
}

type teamStore interface {
	Get(int) (tmodels.Team, error)
	UserBelongsToTeam(userID, teamID int) (bool, error)
}

type userStore interface {
	Get(int) (umodels.User, error)
	CreateContact(user *umodels.User) error
}

type mediaStore interface {
	GetBlob(name string) ([]byte, error)
	Attach(id int, model string, modelID int) error
	GetByModel(id int, model string) ([]mmodels.Media, error)
	ContentIDExists(contentID string) (bool, error)
	UploadAndInsert(fileName, contentType, contentID, modelType string, modelID int, content io.ReadSeeker, fileSize int, disposition string, meta []byte) (mmodels.Media, error)
}

type inboxStore interface {
	Get(int) (inbox.Inbox, error)
}

// Opts holds the options for creating a new Manager.
type Opts struct {
	DB                       *sqlx.DB
	Lo                       *logf.Logger
	OutgoingMessageQueueSize int
	IncomingMessageQueueSize int
}

// New initializes a new conversation Manager.
func New(
	wsHub *ws.Hub,
	i18n *i18n.I18n,
	notifier *notifier.Service,
	inboxStore inboxStore,
	userStore userStore,
	teamStore teamStore,
	mediaStore mediaStore,
	automation *automation.Engine,
	template *template.Manager,
	opts Opts) (*Manager, error) {

	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	c := &Manager{
		q:                          q,
		wsHub:                      wsHub,
		i18n:                       i18n,
		notifier:                   notifier,
		inboxStore:                 inboxStore,
		userStore:                  userStore,
		teamStore:                  teamStore,
		mediaStore:                 mediaStore,
		automation:                 automation,
		template:                   template,
		db:                         opts.DB,
		lo:                         opts.Lo,
		incomingMessageQueue:       make(chan models.IncomingMessage, opts.IncomingMessageQueueSize),
		outgoingMessageQueue:       make(chan models.Message, opts.OutgoingMessageQueueSize),
		outgoingProcessingMessages: sync.Map{},
	}

	return c, nil
}

type queries struct {
	// Conversation queries.
	GetLatestReceivedMessageSourceID   *sqlx.Stmt `query:"get-latest-received-message-source-id"`
	GetToAddress                       *sqlx.Stmt `query:"get-to-address"`
	GetConversationID                  *sqlx.Stmt `query:"get-conversation-id"`
	GetConversationUUID                *sqlx.Stmt `query:"get-conversation-uuid"`
	GetConversation                    *sqlx.Stmt `query:"get-conversation"`
	GetConversationsCreatedAfter       *sqlx.Stmt `query:"get-conversations-created-after"`
	GetUnassignedConversations         *sqlx.Stmt `query:"get-unassigned-conversations"`
	GetConversations                   string     `query:"get-conversations"`
	GetConversationsListUUIDs          string     `query:"get-conversations-list-uuids"`
	GetConversationParticipants        *sqlx.Stmt `query:"get-conversation-participants"`
	UpdateConversationFirstReplyAt     *sqlx.Stmt `query:"update-conversation-first-reply-at"`
	UpdateConversationAssigneeLastSeen *sqlx.Stmt `query:"update-conversation-assignee-last-seen"`
	UpdateConversationAssignedUser     *sqlx.Stmt `query:"update-conversation-assigned-user"`
	UpdateConversationAssignedTeam     *sqlx.Stmt `query:"update-conversation-assigned-team"`
	UpdateConversationPriority         *sqlx.Stmt `query:"update-conversation-priority"`
	UpdateConversationStatus           *sqlx.Stmt `query:"update-conversation-status"`
	UpdateConversationLastMessage      *sqlx.Stmt `query:"update-conversation-last-message"`
	UpdateConversationMeta             *sqlx.Stmt `query:"update-conversation-meta"`
	InsertConverstionParticipant       *sqlx.Stmt `query:"insert-conversation-participant"`
	InsertConversation                 *sqlx.Stmt `query:"insert-conversation"`
	UpsertConversationTags             *sqlx.Stmt `query:"upsert-conversation-tags"`
	UnassignOpenConversations          *sqlx.Stmt `query:"unassign-open-conversations"`
	UnsnoozeAll                        *sqlx.Stmt `query:"unsnooze-all"`

	// Dashboard queries.
	GetDashboardCharts string `query:"get-dashboard-charts"`
	GetDashboardCounts string `query:"get-dashboard-counts"`

	// Message queries.
	GetMessage                         *sqlx.Stmt `query:"get-message"`
	GetMessages                        string     `query:"get-messages"`
	GetPendingMessages                 *sqlx.Stmt `query:"get-pending-messages"`
	GetConversationUUIDFromMessageUUID *sqlx.Stmt `query:"get-conversation-uuid-from-message-uuid"`
	InsertMessage                      *sqlx.Stmt `query:"insert-message"`
	UpdateMessageStatus                *sqlx.Stmt `query:"update-message-status"`
	MessageExistsBySourceID            *sqlx.Stmt `query:"message-exists-by-source-id"`
	GetConversationByMessageID         *sqlx.Stmt `query:"get-conversation-by-message-id"`
}

// CreateConversation creates a new conversation and returns its ID and UUID.
func (c *Manager) CreateConversation(contactID, contactChannelID, inboxID int, lastMessage string, lastMessageAt time.Time, subject string) (int, string, error) {
	var (
		id   int
		uuid string
	)
	if err := c.q.InsertConversation.QueryRow(contactID, contactChannelID, models.StatusOpen, inboxID, lastMessage, lastMessageAt, subject).Scan(&id, &uuid); err != nil {
		c.lo.Error("error inserting new conversation into the DB", "error", err)
		return id, uuid, err
	}
	return id, uuid, nil
}

// GetConversation retrieves a conversation by its UUID.
func (c *Manager) GetConversation(id int, uuid string) (models.Conversation, error) {
	var conversation models.Conversation
	if err := c.q.GetConversation.Get(&conversation, id, uuid); err != nil {
		if err == sql.ErrNoRows {
			return conversation, envelope.NewError(envelope.InputError, "Conversation not found", nil)
		}
		c.lo.Error("error fetching conversation", "error", err)
		return conversation, envelope.NewError(envelope.GeneralError, "Error fetching conversation", nil)
	}
	return conversation, nil
}

// GetConversationsCreatedAfter retrieves conversations created after the specified time.
func (c *Manager) GetConversationsCreatedAfter(time time.Time) ([]models.Conversation, error) {
	var conversations = make([]models.Conversation, 0)
	if err := c.q.GetConversationsCreatedAfter.Select(&conversations, time); err != nil {
		c.lo.Error("error fetching conversation", "error", err)
		return conversations, err
	}
	return conversations, nil
}

// UpdateConversationAssigneeLastSeen updates the last seen timestamp of assignee.
func (c *Manager) UpdateConversationAssigneeLastSeen(uuid string) error {
	if _, err := c.q.UpdateConversationAssigneeLastSeen.Exec(uuid); err != nil {
		c.lo.Error("error updating conversation", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating assignee last seen", nil)
	}

	// Broadcast the property update to all subscribers.
	c.BroadcastConversationPropertyUpdate(uuid, "assignee_last_seen_at", time.Now().Format(time.RFC3339))
	return nil
}

// GetConversationParticipants retrieves the participants of a conversation.
func (c *Manager) GetConversationParticipants(uuid string) ([]models.ConversationParticipant, error) {
	conv := make([]models.ConversationParticipant, 0)
	if err := c.q.GetConversationParticipants.Select(&conv, uuid); err != nil {
		c.lo.Error("error fetching conversation", "error", err)
		return conv, envelope.NewError(envelope.GeneralError, "Error fetching conversation participants", nil)
	}
	return conv, nil
}

// AddConversationParticipant adds a user as participant to a conversation.
func (c *Manager) AddConversationParticipant(userID int, conversationUUID string) error {
	if _, err := c.q.InsertConverstionParticipant.Exec(userID, conversationUUID); err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return nil
		}
		return err
	}
	return nil
}

// GetUnassignedConversations retrieves unassigned conversations.
func (c *Manager) GetUnassignedConversations() ([]models.Conversation, error) {
	var conv []models.Conversation
	if err := c.q.GetUnassignedConversations.Select(&conv); err != nil {
		if err != sql.ErrNoRows {
			c.lo.Error("error fetching conversations", "error", err)
			return conv, err
		}
	}
	return conv, nil
}

// GetConversationID retrieves the ID of a conversation by its UUID.
func (c *Manager) GetConversationID(uuid string) (int, error) {
	var id int
	if err := c.q.GetConversationID.QueryRow(uuid).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return id, err
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return id, err
	}
	return id, nil
}

// GetConversationUUID retrieves the UUID of a conversation by its ID.
func (c *Manager) GetConversationUUID(id int) (string, error) {
	var uuid string
	if err := c.q.GetConversationUUID.QueryRow(id).Scan(&uuid); err != nil {
		if err == sql.ErrNoRows {
			return uuid, err
		}
		c.lo.Error("fetching conversation from DB", "error", err)
		return uuid, err
	}
	return uuid, nil
}

// GetAllConversationsList retrieves all conversations with optional filtering, ordering, and pagination.
func (c *Manager) GetAllConversationsList(order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(0, models.AllConversations, order, orderBy, filters, page, pageSize)
}

// GetAssignedConversationsList retrieves conversations assigned to a specific user with optional filtering, ordering, and pagination.
func (c *Manager) GetAssignedConversationsList(userID int, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(userID, models.AssignedConversations, order, orderBy, filters, page, pageSize)
}

// GetUnassignedConversationsList retrieves conversations assigned to a team the user is part of with optional filtering, ordering, and pagination.
func (c *Manager) GetUnassignedConversationsList(order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(0, models.UnassignedConversations, order, orderBy, filters, page, pageSize)
}

// GetTeamUnassignedConversationsList retrieves conversations assigned to a team with optional filtering, ordering, and pagination.
func (c *Manager) GetTeamUnassignedConversationsList(teamID int, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	return c.GetConversations(teamID, models.TeamUnassignedConversations, order, orderBy, filters, page, pageSize)
}

func (c *Manager) GetViewConversationsList(userID int, typ, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	fmt.Println("Applying view filters", filters)
	switch typ {
	case models.AssignedConversations:
		return c.GetAssignedConversationsList(userID, order, orderBy, filters, page, pageSize)
	case models.UnassignedConversations:
		return c.GetUnassignedConversationsList(order, orderBy, filters, page, pageSize)
	case models.AllConversations:
		return c.GetAllConversationsList(order, orderBy, filters, page, pageSize)
	default:
		return nil, envelope.NewError(envelope.InputError, fmt.Sprintf("Invalid conversation type: %s", typ), nil)
	}
}

// GetConversations retrieves conversations list based on user ID, type, and optional filtering, ordering, and pagination.
func (c *Manager) GetConversations(userID int, listType, order, orderBy, filters string, page, pageSize int) ([]models.Conversation, error) {
	var conversations = make([]models.Conversation, 0)

	// Make the query.
	query, qArgs, err := c.makeConversationsListQuery(userID, c.q.GetConversations, listType, order, orderBy, page, pageSize, filters)
	if err != nil {
		c.lo.Error("error making conversations query", "error", err)
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

// GetConversationsListUUIDs retrieves the UUIDs of conversations list, used to subscribe to conversations.
func (c *Manager) GetConversationsListUUIDs(userID, teamID, page, pageSize int, typ string) ([]string, error) {
	var (
		ids = make([]string, 0)
		id  = userID
	)

	if typ == models.TeamUnassignedConversations {
		id = teamID
		if teamID == 0 {
			return ids, fmt.Errorf("team ID is required for team unassigned conversations")
		}
		exists, err := c.teamStore.UserBelongsToTeam(userID, teamID)
		if err != nil {
			return ids, fmt.Errorf("fetching team members: %w", err)
		}
		if !exists {
			return ids, fmt.Errorf("user does not belong to team")
		}
	}

	query, qArgs, err := c.makeConversationsListQuery(id, c.q.GetConversationsListUUIDs, typ, "", "", page, pageSize, "")
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

// UpdateConversationMeta updates the metadata of a conversation.
func (c *Manager) UpdateConversationMeta(conversationID int, conversationUUID string, meta map[string]string) error {
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		c.lo.Error("error marshalling conversation meta", "meta", meta, "error", err)
		return err
	}
	if _, err := c.q.UpdateConversationMeta.Exec(conversationID, conversationUUID, metaJSON); err != nil {
		c.lo.Error("error updating conversation meta", "error", "error")
		return err
	}
	return nil
}

// UpdateConversationLastMessage updates the last message details for a conversation.
func (c *Manager) UpdateConversationLastMessage(convesationID int, conversationUUID, lastMessage string, lastMessageAt time.Time) error {
	if _, err := c.q.UpdateConversationLastMessage.Exec(convesationID, conversationUUID, lastMessage, lastMessageAt); err != nil {
		c.lo.Error("error updating conversation last message", "error", err)
		return err
	}
	return nil
}

// UpdateConversationFirstReplyAt updates the first reply timestamp for a conversation.
func (c *Manager) UpdateConversationFirstReplyAt(conversationUUID string, conversationID int, at time.Time) error {
	res, err := c.q.UpdateConversationFirstReplyAt.Exec(conversationID, at)
	if err != nil {
		c.lo.Error("error updating conversation first reply at", "error", err)
		return err
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		c.BroadcastConversationPropertyUpdate(conversationUUID, "first_reply_at", at.Format(time.RFC3339))
	}
	return nil
}

// UpdateConversationUserAssignee sets the assignee of a conversation to a specifc user.
func (c *Manager) UpdateConversationUserAssignee(uuid string, assigneeID int, actor umodels.User) error {
	if err := c.UpdateAssignee(uuid, assigneeID, models.AssigneeTypeUser); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error updating assignee", nil)
	}

	conversation, err := c.GetConversation(0, uuid)
	if err != nil {
		return err
	}

	// Send email to assignee.
	if err := c.SendAssignedConversationEmail([]int{assigneeID}, conversation.Subject.String, uuid); err != nil {
		c.lo.Error("error sending assigned conversation email", "error", err)
	}

	if err := c.RecordAssigneeUserChange(uuid, assigneeID, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording assignee change", nil)
	}
	return nil
}

// UpdateConversationTeamAssignee sets the assignee of a conversation to a specific team and sets the assigned user id to NULL.
func (c *Manager) UpdateConversationTeamAssignee(uuid string, teamID int, actor umodels.User) error {
	if err := c.UpdateAssignee(uuid, teamID, models.AssigneeTypeTeam); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error updating assignee", nil)
	}
	if err := c.RecordAssigneeTeamChange(uuid, teamID, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording assignee change", nil)
	}
	return nil
}

// UpdateAssignee updates the assignee of a conversation.
func (c *Manager) UpdateAssignee(uuid string, assigneeID int, assigneeType string) error {
	switch assigneeType {
	case models.AssigneeTypeUser:
		if _, err := c.q.UpdateConversationAssignedUser.Exec(uuid, assigneeID); err != nil {
			c.lo.Error("error updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}

		// Broadcast update to all subscribers.
		c.BroadcastConversationPropertyUpdate(uuid, "assigned_user_id", assigneeID)
	case models.AssigneeTypeTeam:
		if _, err := c.q.UpdateConversationAssignedTeam.Exec(uuid, assigneeID); err != nil {
			c.lo.Error("error updating conversation assignee", "error", err)
			return fmt.Errorf("error updating assignee")
		}

		// Broadcast update to all subscribers.
		c.BroadcastConversationPropertyUpdate(uuid, "assigned_team_id", assigneeID)
	default:
		return errors.New("invalid assignee type")
	}
	return nil
}

// UpdateConversationPriority updates the priority of a conversation.
func (c *Manager) UpdateConversationPriority(uuid string, priority []byte, actor umodels.User) error {
	var priorityStr = string(priority)
	if _, err := c.q.UpdateConversationPriority.Exec(uuid, priority); err != nil {
		c.lo.Error("error updating conversation priority", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating priority", nil)
	}
	if err := c.RecordPriorityChange(priorityStr, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording priority change", nil)
	}
	return nil
}

// UpdateConversationStatus updates the status of a conversation.
func (c *Manager) UpdateConversationStatus(uuid string, status []byte, snoozeDur []byte, actor umodels.User) error {
	var (
		statusStr  = string(status)
		snoozeDurS = string(snoozeDur)
	)

	if statusStr == models.StatusSnoozed && snoozeDurS == "" {
		return envelope.NewError(envelope.InputError, "Snooze duration is required", nil)
	}

	snoozeUntil := time.Time{}
	if statusStr == models.StatusSnoozed {
		duration, err := time.ParseDuration(snoozeDurS)
		if err != nil {
			return envelope.NewError(envelope.InputError, "Invalid snooze duration format", nil)
		}
		snoozeUntil = time.Now().Add(duration)
	}

	if _, err := c.q.UpdateConversationStatus.Exec(uuid, status, snoozeUntil); err != nil {
		c.lo.Error("error updating conversation status", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating status", nil)
	}
	if err := c.RecordStatusChange(statusStr, uuid, actor); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error recording status change", nil)
	}
	c.BroadcastConversationPropertyUpdate(uuid, "status", string(status))
	return nil
}

// GetDashboardCounts returns dashboard counts
// TODO: Rename to overview [reports/overview].
func (c *Manager) GetDashboardCounts(userID, teamID int) (json.RawMessage, error) {
	var counts = json.RawMessage{}
	tx, err := c.db.BeginTxx(context.Background(), nil)
	if err != nil {
		c.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching dashboard counts", nil)
	}
	defer tx.Rollback()

	var (
		cond  string
		qArgs []interface{}
	)
	if userID > 0 {
		cond = " AND assigned_user_id = $1"
		qArgs = append(qArgs, userID)
	} else if teamID > 0 {
		cond = " AND assigned_team_id = $1"
		qArgs = append(qArgs, teamID)
	}
	// TODO: Add date range filter support.
	cond += " AND c.created_at >= NOW() - INTERVAL '30 days'"

	query := fmt.Sprintf(c.q.GetDashboardCounts, cond)
	if err := tx.Get(&counts, query, qArgs...); err != nil {
		c.lo.Error("error fetching dashboard counts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching dashboard counts", nil)
	}

	if err := tx.Commit(); err != nil {
		c.lo.Error("error committing db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching dashboard counts", nil)
	}

	return counts, nil
}

// GetDashboardChart returns dashboard chart data
func (c *Manager) GetDashboardChart(userID, teamID int) (json.RawMessage, error) {
	var stats = json.RawMessage{}
	tx, err := c.db.BeginTxx(context.Background(), nil)
	if err != nil {
		c.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching dashboard charts", nil)
	}
	defer tx.Rollback()

	var (
		cond  string
		qArgs []interface{}
	)

	// TODO: Add date range filter support.
	if userID > 0 {
		cond = " AND assigned_user_id = $1"
		qArgs = append(qArgs, userID)
	} else if teamID > 0 {
		cond = " AND assigned_team_id = $1"
		qArgs = append(qArgs, teamID)
	}
	cond += " AND c.created_at >= NOW() - INTERVAL '30 days'"

	// Apply the same condition across queries.
	query := fmt.Sprintf(c.q.GetDashboardCharts, cond, cond)
	if err := tx.Get(&stats, query, qArgs...); err != nil {
		c.lo.Error("error fetching dashboard charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching dashboard charts", nil)
	}
	return stats, nil
}

// UpsertConversationTags upserts the tags associated with a conversation.
func (t *Manager) UpsertConversationTags(uuid string, tagIDs []int) error {
	if _, err := t.q.UpsertConversationTags.Exec(uuid, pq.Array(tagIDs)); err != nil {
		t.lo.Error("error upserting conversation tags", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error upserting tags", nil)
	}
	return nil
}

// makeConversationsListQuery prepares a SQL query string for conversations list
func (c *Manager) makeConversationsListQuery(userID int, baseQuery, listType, order, orderBy string, page, pageSize int, filtersJSON string) (string, []interface{}, error) {
	var qArgs []interface{}
	if orderBy == "" {
		orderBy = "last_message_at"
	}
	if order == "" {
		order = "DESC"
	}
	if filtersJSON == "" {
		filtersJSON = "[]"
	}
	if pageSize > conversationsListMaxPageSize {
		return "", nil, fmt.Errorf("page size exceeds maximum limit of %d", conversationsListMaxPageSize)
	}
	if pageSize < 1 {
		return "", nil, fmt.Errorf("page size must be greater than 0")
	}
	if page < 1 {
		return "", nil, fmt.Errorf("page must be greater than 0")
	}
	// Apply filters based on the type of conversation list.
	switch listType {
	// Conversations assigned to the current user.
	case models.AssignedConversations:
		baseQuery = fmt.Sprintf(baseQuery, "AND conversations.assigned_user_id = $1")
		qArgs = append(qArgs, userID)
	// Conversations that are unassigned.
	case models.UnassignedConversations:
		baseQuery = fmt.Sprintf(baseQuery, "AND conversations.assigned_user_id IS NULL AND conversations.assigned_team_id IS NULL")
	// All conversations without any specific filter.
	case models.AllConversations:
		baseQuery = fmt.Sprintf(baseQuery, "")
	// Conversations assigned to a team but not to a specific user.
	case models.TeamUnassignedConversations:
		baseQuery = fmt.Sprintf(baseQuery, "AND conversations.assigned_team_id = $1 AND conversations.assigned_user_id IS NULL")
		qArgs = append(qArgs, userID)
	default:
		return "", nil, fmt.Errorf("unknown conversation type: %s", listType)
	}

	// Build the paginated query.
	query, qArgs, err := dbutil.BuildPaginatedQuery(baseQuery, qArgs, dbutil.PaginationOptions{
		Order:    order,
		OrderBy:  orderBy,
		Page:     page,
		PageSize: pageSize,
	}, filtersJSON, dbutil.AllowedFields{
		"conversations":         ConversationsListAllowedFilterFields,
		"conversation_statuses": ConversationStatusesFilterFields,
	})
	if err != nil {
		c.lo.Error("error preparing query", "error", err)
		return "", nil, err
	}
	return query, qArgs, err
}

// GetToAddress retrieves the recipient addresses for a conversation and channel.
func (m *Manager) GetToAddress(conversationID int) ([]string, error) {
	var addr []string
	if err := m.q.GetToAddress.Select(&addr, conversationID); err != nil {
		m.lo.Error("error fetching `to` address for message", "error", err, "conversation_id", conversationID)
		return addr, err
	}
	return addr, nil
}

// GetLatestReceivedMessageSourceID returns the last received message source ID.
func (m *Manager) GetLatestReceivedMessageSourceID(conversationID int) (string, error) {
	var out string
	if err := m.q.GetLatestReceivedMessageSourceID.Get(&out, conversationID); err != nil {
		m.lo.Error("error fetching message source id", "error", err, "conversation_id", conversationID)
		return out, err
	}
	return out, nil
}

// SendAssignedConversationEmail sends a email for an assigned conversation to the passed user ids.
func (m *Manager) SendAssignedConversationEmail(userIDs []int, subject, conversationUUID string) error {
	content, subject, err := m.template.RenderNamedTemplate(template.TmplConversationAssigned,
		map[string]interface{}{
			"conversation": map[string]string{
				"subject": subject,
				"uuid":    conversationUUID,
			},
		})
	if err != nil {
		m.lo.Error("error rendering template", "template", template.TmplConversationAssigned, "conversation_uuid", conversationUUID, "error", err)
		return fmt.Errorf("rendering template: %w", err)
	}
	nm := notifier.Message{
		UserIDs:  userIDs,
		Subject:  subject,
		Content:  content,
		Provider: notifier.ProviderEmail,
	}
	if err := m.notifier.Send(nm); err != nil {
		m.lo.Error("error sending notification message", "error", err)
		return fmt.Errorf("sending notification message: %w", err)
	}
	return nil
}

// UnassignOpen unassigns all open conversations belonging to a user.
// i.e conversations without status `Closed` and `Resolved`.
func (m *Manager) UnassignOpen(userID int) error {
	if _, err := m.q.UnassignOpenConversations.Exec(userID); err != nil {
		m.lo.Error("error unassigning open conversations", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error unassigning open conversations", nil)
	}
	return nil
}
