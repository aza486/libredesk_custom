package message

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/automation"
	cmodels "github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/inbox"
	mmodels "github.com/abhinavxd/artemis/internal/media/models"
	"github.com/abhinavxd/artemis/internal/message/models"
	"github.com/abhinavxd/artemis/internal/stringutil"
	tmodels "github.com/abhinavxd/artemis/internal/team/models"
	"github.com/abhinavxd/artemis/internal/template"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	ErrConversationNotFound = errors.New("conversation not found")
)

const (
	TypeIncoming = "incoming"
	TypeOutgoing = "outgoing"
	TypeActivity = "activity"

	SenderTypeUser    = "user"
	SenderTypeContact = "contact"

	StatusPending   = "pending"
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusRead      = "read"
	StatusFailed    = "failed"
	StatusReceived  = "received"

	ActivityStatusChange       = "status_change"
	ActivityPriorityChange     = "priority_change"
	ActivityAssignedUserChange = "assigned_user_change"
	ActivityAssignedTeamChange = "assigned_team_change"
	ActivitySelfAssign         = "self_assign"
	ActivityTagChange          = "tag_change"

	ContentTypeText = "text"
	ContentTypeHTML = "html"

	maxLastMessageLen  = 45
	maxMessagesPerPage = 30
)

// Manager handles message-related operations.
type Manager struct {
	q                          queries
	db                         *sqlx.DB
	lo                         *logf.Logger
	contactStore               contactStore
	inboxStore                 inboxStore
	conversationStore          conversationStore
	userStore                  userStore
	teamStore                  teamStore
	mediaStore                 mediaStore
	automation                 *automation.Engine
	wsHub                      *ws.Hub
	template                   *template.Manager
	incomingMessageQueue       chan models.IncomingMessage
	outgoingMessageQueue       chan models.Message
	outgoingProcessingMessages sync.Map
}

// Opts contains options for initializing the message Manager.
type Opts struct {
	DB                   *sqlx.DB
	Lo                   *logf.Logger
	IncomingMsgQueueSize int
	OutgoingMsgQueueSize int
}

type teamStore interface {
	GetTeam(int) (tmodels.Team, error)
}

type userStore interface {
	Get(int, string) (umodels.User, error)
}

type contactStore interface {
	Upsert(cmodels.Contact) (int, error)
}

type conversationStore interface {
	UpdateFirstReplyAt(string, int, time.Time) error
	UpdateLastMessage(int, string, string, time.Time) error
	Create(int, int, []byte) (int, string, error)
	GetUUID(int) (string, error)
}

type mediaStore interface {
	GetBlob(name string) ([]byte, error)
	AttachToModel(id int, model string, modelID int) error
	GetModelMedia(id int, model string) ([]mmodels.Media, error)
	UploadAndInsert(fileName, contentType string, content io.ReadSeeker, fileSize int, meta []byte) (mmodels.Media, error)
}

type inboxStore interface {
	Get(int) (inbox.Inbox, error)
}

// queries contains prepared SQL queries.
type queries struct {
	GetMessage           *sqlx.Stmt `query:"get-message"`
	GetMessages          string     `query:"get-messages"`
	GetToAddress         *sqlx.Stmt `query:"get-to-address"`
	GetInReplyTo         *sqlx.Stmt `query:"get-in-reply-to"`
	GetPendingMessages   *sqlx.Stmt `query:"get-pending-messages"`
	InsertMessage        *sqlx.Stmt `query:"insert-message"`
	UpdateMessageContent *sqlx.Stmt `query:"update-message-content"`
	UpdateMessageStatus  *sqlx.Stmt `query:"update-message-status"`
	MessageExists        *sqlx.Stmt `query:"message-exists"`
}

// New creates and returns a new instance of the Manager.
func New(
	wsHub *ws.Hub,
	userStore userStore,
	teamStore teamStore,
	contactStore contactStore,
	mediaStore mediaStore,
	inboxStore inboxStore,
	conversationStore conversationStore,
	automation *automation.Engine,
	template *template.Manager,
	opts Opts,
) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:                          q,
		db:                         opts.DB,
		lo:                         opts.Lo,
		wsHub:                      wsHub,
		userStore:                  userStore,
		teamStore:                  teamStore,
		mediaStore:                 mediaStore,
		contactStore:               contactStore,
		inboxStore:                 inboxStore,
		template:                   template,
		conversationStore:          conversationStore,
		automation:                 automation,
		incomingMessageQueue:       make(chan models.IncomingMessage, opts.IncomingMsgQueueSize),
		outgoingMessageQueue:       make(chan models.Message, opts.OutgoingMsgQueueSize),
		outgoingProcessingMessages: sync.Map{},
	}, nil
}

// Run starts worker pool to process incoming and outgoing messages.
func (m *Manager) Run(ctx context.Context, dispatchConcurrency, readerConcurrency int, readInterval time.Duration) {
	// Spawn a worker goroutine pool to dispatch messages.
	for range dispatchConcurrency {
		go m.DispatchWorker(ctx)
	}

	// Spawn a worker goroutine pool to process incoming messages.
	for range readerConcurrency {
		go m.IncomingWorker(ctx)
	}

	// Scan pending messages from the DB on set interval.
	dbScanner := time.NewTicker(readInterval)
	defer dbScanner.Stop()

	for {
		select {

		case <-ctx.Done():
			return

		case <-dbScanner.C:
			var (
				pendingMessages = []models.Message{}
				messageIDs      = m.getOutgoingProcessingMessageIDs()
			)

			// Skip the currently processing msg ids.
			if err := m.q.GetPendingMessages.Select(&pendingMessages, pq.Array(messageIDs)); err != nil {
				m.lo.Error("error fetching pending messages from db", "error", err)
			}

			// Prepare and push the message to the outgoing queue.
			for _, message := range pendingMessages {
				// Get inbox.
				inb, err := m.inboxStore.Get(message.InboxID)
				if err != nil {
					m.lo.Error("error fetching inbox", "error", err, "inbox_id", message.InboxID)
					continue
				}

				switch inb.Channel() {
				case inbox.ChannelEmail:
					// Email channel requires the content to be rendered.
					message.Content, _, err = m.template.RenderDefault(map[string]string{
						"Content": message.Content,
					})
					if err != nil {
						m.lo.Error("error rendering default template", "error", err)
						m.UpdateStatus(message.UUID, StatusFailed)
						continue
					}
				default:
					m.lo.Warn("unknown message channel", "channel", inb.Channel())
					m.UpdateStatus(message.UUID, StatusFailed)
					continue
				}

				// Put the message ID in the processing map.
				m.outgoingProcessingMessages.Store(message.ID, message.ID)

				// Push the message to the outgoing message queue.
				m.outgoingMessageQueue <- message
			}
		}
	}
}

// DispatchWorker dispatches outgoing pending messages.
func (m *Manager) DispatchWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-m.outgoingMessageQueue:
			if !ok {
				return
			}

			// Get inbox.
			inbox, err := m.inboxStore.Get(message.InboxID)
			if err != nil {
				m.lo.Error("error fetching inbox", "error", err, "inbox_id", message.InboxID)
				m.outgoingProcessingMessages.Delete(message.ID)
				m.UpdateStatus(message.UUID, StatusFailed)
				continue
			}

			// Attach attachments to the message.
			if err := m.attachAttachments(&message); err != nil {
				m.lo.Error("error attaching attachments to message", "error", err, "id", message.ID)
				m.outgoingProcessingMessages.Delete(message.ID)
				m.UpdateStatus(message.UUID, StatusFailed)
				continue
			}

			// Get from, to addresses and inReplyTo.
			message.From = inbox.FromAddress()
			message.To, _ = m.GetToAddress(message.ConversationID, inbox.Channel())
			message.InReplyTo, _ = m.GetInReplyTo(message.ConversationID)

			// Send.
			err = inbox.Send(message)

			// Update status.
			var newStatus = StatusFailed
			if err != nil {
				newStatus = StatusFailed
				m.lo.Error("error sending message", "error", err, "inbox_id", message.InboxID)
			}
			m.UpdateStatus(message.UUID, newStatus)

			// Update first reply at.
			if newStatus == StatusSent {
				m.conversationStore.UpdateFirstReplyAt(message.ConversationUUID, message.ConversationID, message.CreatedAt)
			}

			// Broadcast update to all subscribers.
			m.wsHub.BroadcastMessagePropUpdate(message.ConversationUUID, message.UUID, "status" /*message field*/, newStatus)

			// Delete from processing map.
			m.outgoingProcessingMessages.Delete(message.ID)
		}
	}
}

// GetConversationMessages retrieves messages for a specific conversation.
func (m *Manager) GetConversationMessages(conversationUUID string, page, pageSize int) ([]models.Message, error) {
	var (
		messages = make([]models.Message, 0)
		qArgs    []interface{}
	)
	qArgs = append(qArgs, conversationUUID)
	query, qArgs, err := m.generateMessagesQuery(m.q.GetMessages, qArgs, page, pageSize)
	if err != nil {
		m.lo.Error("error generating messages query", "error", err)
		return messages, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}

	tx, err := m.db.BeginTxx(context.Background(), nil)
	defer tx.Commit()
	if err != nil {
		m.lo.Error("error preparing get messages query", "error", err)
		return messages, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}

	if err := tx.Select(&messages, query, qArgs...); err != nil {
		m.lo.Error("error fetching conversations", "error", err)
		return messages, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}

	return messages, nil
}

// Get retrieves a message by UUID.
func (m *Manager) Get(uuid string) (models.Message, error) {
	var message models.Message
	if err := m.q.GetMessage.Get(&message, uuid); err != nil {
		m.lo.Error("error fetching message", "uuid", uuid, "error", err)
		return message, envelope.NewError(envelope.GeneralError, "Error fetching message", nil)
	}
	return message, nil
}

// UpdateStatus updates the status of a message.
func (m *Manager) UpdateStatus(uuid string, status string) error {
	if _, err := m.q.UpdateMessageStatus.Exec(status, uuid); err != nil {
		m.lo.Error("error updating message status in DB", "error", err, "uuid", uuid)
		return err
	}
	return nil
}

// MarkAsPending updates message status to `Pending` so a message can be sent again.
func (m *Manager) MarkAsPending(uuid string) error {
	if err := m.UpdateStatus(uuid, StatusPending); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error retrying message", nil)
	}
	return nil
}

// Insert inserts a message and attaches the attachments to the message.
func (m *Manager) Insert(message *models.Message) error {
	if message.Private {
		message.Status = StatusSent
	}

	if message.Meta == "" {
		message.Meta = "{}"
	}

	// Insert.
	if err := m.q.InsertMessage.QueryRow(message.Type, message.Status, message.ConversationID, message.ConversationUUID, message.Content, message.SenderID, message.SenderType,
		message.Private, message.ContentType, message.SourceID, message.InboxID, message.Meta).Scan(&message.ID, &message.UUID, &message.CreatedAt); err != nil {
		m.lo.Error("error inserting message in db", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error sending message", nil)
	}

	// Attach message to the media.
	for _, media := range message.Media {
		m.mediaStore.AttachToModel(media.ID, mmodels.ModelMessages, message.ID)
	}
	return nil
}

// GetToAddress retrieves the recipient addresses for a message.
func (m *Manager) GetToAddress(convID int, channel string) ([]string, error) {
	var addr []string
	if err := m.q.GetToAddress.Select(&addr, convID, channel); err != nil {
		m.lo.Error("error fetching `to` address for message", "error", err, "conversation_id", convID)
		return addr, err
	}
	return addr, nil
}

// GetInReplyTo retrieves the In-Reply-To header value for a message.
func (m *Manager) GetInReplyTo(convID int) (string, error) {
	var out string
	if err := m.q.GetInReplyTo.Get(&out, convID); err != nil {
		m.lo.Error("error fetching in reply to", "error", err, "conversation_id", convID)
		return out, err
	}
	return out, nil
}

// IncomingWorker processes incoming messages from the incoming message queue.
func (m *Manager) IncomingWorker(ctx context.Context) {
	for msg := range m.incomingMessageQueue {
		select {
		case <-ctx.Done():
			return
		default:
			if err := m.processIncoming(msg); err != nil {
				m.lo.Error("error processing incoming msg", "error", err)
			}
		}
	}
}

// RecordAssigneeUserChange records an activity for a user assignee change.
func (m *Manager) RecordAssigneeUserChange(conversationUUID string, assigneeID int, actor umodels.User) error {
	// Self assign.
	if assigneeID == actor.ID {
		return m.InsertActivity(ActivitySelfAssign, conversationUUID, actor.FullName(), actor)
	}

	// Assignment to another user.
	assignee, err := m.userStore.Get(assigneeID, "")
	if err != nil {
		return err
	}
	return m.InsertActivity(ActivityAssignedUserChange, conversationUUID, assignee.FullName(), actor)
}

// RecordAssigneeTeamChange records an activity for a team assignee change.
func (m *Manager) RecordAssigneeTeamChange(conversationUUID string, teamID int, actor umodels.User) error {
	team, err := m.teamStore.GetTeam(teamID)
	if err != nil {
		return err
	}
	return m.InsertActivity(ActivityAssignedTeamChange, conversationUUID, team.Name, actor)
}

// RecordPriorityChange records an activity for a priority change.
func (m *Manager) RecordPriorityChange(priority, conversationUUID string, actor umodels.User) error {
	return m.InsertActivity(ActivityPriorityChange, conversationUUID, priority, actor)
}

// RecordStatusChange records an activity for a status change.
func (m *Manager) RecordStatusChange(status, conversationUUID string, actor umodels.User) error {
	return m.InsertActivity(ActivityStatusChange, conversationUUID, status, actor)
}

// InsertActivity inserts an activity message.
func (m *Manager) InsertActivity(activityType, conversationUUID, newValue string, actor umodels.User) error {
	content, err := m.getActivityContent(activityType, newValue, actor.FullName())
	if err != nil {
		m.lo.Error("error could not generate activity content", "error", err)
		return err
	}

	message := models.Message{
		Type:             TypeActivity,
		Status:           StatusSent,
		Content:          content,
		ContentType:      ContentTypeText,
		ConversationUUID: conversationUUID,
		Private:          true,
		SenderID:         actor.ID,
		SenderType:       SenderTypeUser,
	}

	// Insert message in DB.
	m.Insert(&message)

	// Broadcast the new message to all subscribers.
	m.BroadcastNewConversationMessage(message, content)

	// Update the last message in conversation meta.
	m.conversationStore.UpdateLastMessage(0, conversationUUID, content, message.CreatedAt)
	return nil
}

// getActivityContent generates activity content based on the activity type.
func (m *Manager) getActivityContent(activityType, newValue, actorName string) (string, error) {
	var content = ""
	switch activityType {
	case ActivityAssignedUserChange:
		content = fmt.Sprintf("Assigned to %s by %s", newValue, actorName)
	case ActivityAssignedTeamChange:
		content = fmt.Sprintf("Assigned to %s team by %s", newValue, actorName)
	case ActivitySelfAssign:
		content = fmt.Sprintf("%s self-assigned this conversation", actorName)
	case ActivityPriorityChange:
		content = fmt.Sprintf("%s changed priority to %s", actorName, newValue)
	case ActivityStatusChange:
		content = fmt.Sprintf("%s marked the conversation as %s", actorName, newValue)
	case ActivityTagChange:
		content = fmt.Sprintf("%s added tags %s", actorName, newValue)
	default:
		return "", fmt.Errorf("invalid activity type %s", activityType)
	}
	return content, nil
}

// processIncoming processes an incoming message by upserting contact, conversation and message.
func (m *Manager) processIncoming(in models.IncomingMessage) error {
	var err error

	// Create contact.
	in.Message.SenderID, err = m.contactStore.Upsert(in.Contact)
	if err != nil {
		m.lo.Error("error upserting contact", "error", err)
		return err
	}

	// Conversation already exists?
	conversationID, err := m.findConversationID([]string{in.Message.SourceID.String})
	if err != nil && err != ErrConversationNotFound {
		return err
	}
	if conversationID > 0 {
		return nil
	}

	// Find or create new conversation.
	isNewConversation, err := m.findOrCreateConversation(&in.Message, in.InboxID, in.Message.SenderID)
	if err != nil {
		return err
	}

	// Insert message.
	if err = m.Insert(&in.Message); err != nil {
		return fmt.Errorf("inserting conversation message: %w", err)
	}

	// Upload attachments.
	if err := m.uploadAttachments(&in.Message); err != nil {
		return fmt.Errorf("uploading message attachments: %w", err)
	}

	// Send WS update to all subscribers.
	if in.Message.ConversationUUID != "" {
		var content string
		if isNewConversation {
			content = stringutil.Trim(in.Message.Subject, maxLastMessageLen)
		} else {
			content = stringutil.Trim(in.Message.Content, maxLastMessageLen)
		}
		m.BroadcastNewConversationMessage(in.Message, content)
		m.conversationStore.UpdateLastMessage(in.Message.ConversationID, in.Message.ConversationUUID, content, in.Message.CreatedAt)
	}

	// Evaluate automation rules for this conversation.
	if isNewConversation {
		m.automation.EvaluateNewConversationRules(in.Message.ConversationUUID)
	} else {
		m.automation.EvaluateConversationUpdateRules(in.Message.ConversationUUID)
	}
	return nil
}

// Exists checks if a message with the given messageID exists.
func (m *Manager) Exists(messageID string) (bool, error) {
	_, err := m.findConversationID([]string{messageID})
	if err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			return false, nil
		}
		m.lo.Error("error fetching message from db", "error", err)
		return false, err
	}
	return true, nil
}

// EnqueueIncoming enqueues an incoming message for inserting.
func (m *Manager) EnqueueIncoming(message models.IncomingMessage) error {
	select {
	case m.incomingMessageQueue <- message:
		return nil
	default:
		m.lo.Error("incoming message queue is full")
		return errors.New("incoming message queue is full")
	}
}

// generateMessagesQuery generates the SQL query for fetching messages in a conversation
func (c *Manager) generateMessagesQuery(baseQuery string, qArgs []interface{}, page, pageSize int) (string, []interface{}, error) {
	// Set default values for page and page size if they are invalid
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > maxMessagesPerPage {
		pageSize = maxMessagesPerPage
	}

	// Calculate the offset
	offset := (page - 1) * pageSize

	// Append LIMIT and OFFSET to query arguments
	qArgs = append(qArgs, pageSize, offset)

	// Include LIMIT and OFFSET in the SQL query
	sqlQuery := fmt.Sprintf(baseQuery, fmt.Sprintf("LIMIT $%d OFFSET $%d", len(qArgs)-1, len(qArgs)))
	return sqlQuery, qArgs, nil
}

// uploadAttachments uploads attachments for a message.
func (m *Manager) uploadAttachments(message *models.Message) error {
	var (
		hasInline bool
	)

	for _, attachment := range message.Attachments {
		// Upload & insert the attachment.
		reader := bytes.NewReader(attachment.Content)
		media, err := m.mediaStore.UploadAndInsert(attachment.Name, attachment.ContentType, reader, attachment.Size, []byte("{}"))
		if err != nil {
			m.lo.Error("error uploading message media", "error", err)
			return err
		}

		// Inline? Replace cids with actual URL.
		if attachment.Disposition == mmodels.DispositionInline {
			hasInline = true
			message.Content = strings.ReplaceAll(message.Content, "cid:"+attachment.ContentID, media.URL)
		}

		// Attach message to media.
		if err := m.mediaStore.AttachToModel(media.ID, mmodels.ModelMessages, message.ID); err != nil {
			m.lo.Error("error attaching message to media", "error", err)
			return err
		}
	}

	// Update message content if the `cid:content_id` URLs have been replaced.
	if hasInline {
		if _, err := m.q.UpdateMessageContent.Exec(message.Content, message.ID); err != nil {
			m.lo.Error("error updating message content", "error", err)
			return err
		}
	}
	return nil
}

// findOrCreateConversation finds or creates a conversation for the given message.
func (m *Manager) findOrCreateConversation(in *models.Message, inboxID int, contactID int) (bool, error) {
	var (
		new              bool
		err              error
		conversationID   int
		conversationUUID string
	)

	// Search for existing conversation.
	sourceIDs := in.References
	if in.InReplyTo != "" {
		sourceIDs = append(sourceIDs, in.InReplyTo)
	}
	conversationID, err = m.findConversationID(sourceIDs)
	if err != nil && err != ErrConversationNotFound {
		return new, err
	}

	// Conversation not found, create one.
	if conversationID == 0 {
		new = true

		// Prepare meta.
		conversationMeta, err := json.Marshal(map[string]string{
			"subject":         in.Subject,
			"last_message":    stringutil.Trim(in.Content, maxLastMessageLen),
			"last_message_at": time.Now().Format(time.RFC3339),
		})
		if err != nil {
			return false, err
		}
		conversationID, conversationUUID, err = m.conversationStore.Create(contactID, inboxID, conversationMeta)
		if err != nil || conversationID == 0 {
			return new, err
		}
		in.ConversationID = conversationID
		in.ConversationUUID = conversationUUID
		return new, nil
	}

	// Set UUID if not available.
	if conversationUUID == "" {
		conversationUUID, err = m.conversationStore.GetUUID(conversationID)
		if err != nil {
			return new, err
		}
	}
	in.ConversationID = conversationID
	in.ConversationUUID = conversationUUID
	return new, nil
}

// findConversationID finds the conversation ID from the message source ID.
func (m *Manager) findConversationID(sourceIDs []string) (int, error) {
	if len(sourceIDs) == 0 {
		return 0, ErrConversationNotFound
	}
	var conversationID int
	if err := m.q.MessageExists.QueryRow(pq.Array(sourceIDs)).Scan(&conversationID); err != nil {
		if err == sql.ErrNoRows {
			return conversationID, ErrConversationNotFound
		}
		m.lo.Error("error fetching msg from DB", "error", err)
		return conversationID, err
	}
	return conversationID, nil
}

// attachAttachments attaches attachment blobs to message.
func (m *Manager) attachAttachments(message *models.Message) error {
	var attachments attachment.Attachments

	// Get all media for this message.
	medias, err := m.mediaStore.GetModelMedia(message.ID, mmodels.ModelMessages)
	if err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return err
	}

	// Fetch blobs.
	for _, media := range medias {
		blob, err := m.mediaStore.GetBlob(media.Filename)
		if err != nil {
			m.lo.Error("error fetching media blob", "error", err)
			return err
		}
		attachment := attachment.Attachment{
			Name:    media.Filename,
			Content: blob,
			Header:  attachment.MakeHeader(media.ContentType, media.Filename, "base64"),
		}
		attachments = append(attachments, attachment)
	}

	// Attach attachments.
	message.Attachments = attachments

	return nil
}

// getOutgoingProcessingMessageIDs returns the IDs of outgoing messages currently being processed.
func (m *Manager) getOutgoingProcessingMessageIDs() []int {
	var out = make([]int, 0)
	m.outgoingProcessingMessages.Range(func(key, _ any) bool {
		if k, ok := key.(int); ok {
			out = append(out, k)
		}
		return true
	})
	return out
}

// BroadcastNewConversationMessage broadcasts a new conversation message to subscribers.
func (m *Manager) BroadcastNewConversationMessage(message models.Message, content string) {
	m.wsHub.BroadcastNewConversationMessage(message.ConversationUUID, content, message.UUID, time.Now().Format(time.RFC3339), message.Private)
}
