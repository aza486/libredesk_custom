package message

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/automation"
	cmodels "github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/message/models"
	tmodels "github.com/abhinavxd/artemis/internal/team/models"
	"github.com/abhinavxd/artemis/internal/template"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/k3a/html2text"
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

	maxLastMessageLen = 45
)

// Manager handles message-related operations.
type Manager struct {
	q                          queries
	lo                         *logf.Logger
	contactStore               contactStore
	inboxStore                 inboxStore
	conversationStore          conversationStore
	userStore                  userStore
	teamStore                  teamStore
	attachmentManager          *attachment.Manager
	automationEngine           *automation.Engine
	wsHub                      *ws.Hub
	templateManager            *template.Manager
	incomingMsgQ               chan models.IncomingMessage
	outgoingMessageQueue       chan models.Message
	outgoingProcessingMessages sync.Map
}

// Opts contains options for initializing the Manager.
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

type inboxStore interface {
	Get(int) (inbox.Inbox, error)
}

// queries contains prepared SQL queries.
type queries struct {
	GetMessage           *sqlx.Stmt `query:"get-message"`
	GetMessages          *sqlx.Stmt `query:"get-messages"`
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
	attachmentManager *attachment.Manager,
	inboxStore inboxStore,
	conversationStore conversationStore,
	automationEngine *automation.Engine,
	templateManager *template.Manager,
	opts Opts,
) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:                          q,
		lo:                         opts.Lo,
		wsHub:                      wsHub,
		userStore:                  userStore,
		teamStore:                  teamStore,
		contactStore:               contactStore,
		conversationStore:          conversationStore,
		attachmentManager:          attachmentManager,
		inboxStore:                 inboxStore,
		automationEngine:           automationEngine,
		templateManager:            templateManager,
		incomingMsgQ:               make(chan models.IncomingMessage, opts.IncomingMsgQueueSize),
		outgoingMessageQueue:       make(chan models.Message, opts.OutgoingMsgQueueSize),
		outgoingProcessingMessages: sync.Map{},
	}, nil
}

// GetConversationMessages retrieves messages for a specific conversation.
func (m *Manager) GetConversationMessages(uuid string) ([]models.Message, error) {
	var messages []models.Message
	if err := m.q.GetMessages.Select(&messages, uuid); err != nil {
		m.lo.Error("error fetching messages from DB", "conversation_uuid", uuid, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}
	return messages, nil
}

// GetMessage retrieves a message by UUID.
func (m *Manager) GetMessage(uuid string) ([]models.Message, error) {
	var messages []models.Message
	if err := m.q.GetMessage.Select(&messages, uuid); err != nil {
		m.lo.Error("error fetching message from DB", "message_uuid", uuid, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching message", nil)
	}
	return messages, nil
}

// UpdateMessageStatus updates the status of a message.
func (m *Manager) UpdateMessageStatus(uuid string, status string) error {
	if _, err := m.q.UpdateMessageStatus.Exec(status, uuid); err != nil {
		m.lo.Error("error updating message status in DB", "error", err, "uuid", uuid)
		return err
	}
	return nil
}

// RetryMessage retries sending a message by updating its status to pending.
func (m *Manager) RetryMessage(uuid string) error {
	if err := m.UpdateMessageStatus(uuid, StatusPending); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error retrying message", nil)
	}
	return nil
}

// RecordMessage inserts a message and attaches the attachments to the message.
func (m *Manager) RecordMessage(msg *models.Message) error {
	if msg.Private {
		msg.Status = StatusSent
	}

	if err := m.q.InsertMessage.QueryRow(msg.Type, msg.Status, msg.ConversationID, msg.ConversationUUID, msg.Content, msg.SenderID, msg.SenderType,
		msg.Private, msg.ContentType, msg.SourceID, msg.InboxID, msg.Meta).Scan(&msg.ID, &msg.UUID, &msg.CreatedAt); err != nil {
		m.lo.Error("error inserting message in db", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error sending message", nil)
	}

	// Attach the attachments.
	if err := m.attachmentManager.AttachMessage(msg.Attachments, msg.ID); err != nil {
		m.lo.Error("error attaching attachments to the message", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error sending message", nil)
	}

	return nil
}

// StartDispatcher starts a worker pool to dispatch pending outbound messages.
func (m *Manager) StartDispatcher(ctx context.Context, concurrency int, readInterval time.Duration) {
	// Spawn goroutine worker pool.
	for i := 0; i < concurrency; i++ {
		go m.DispatchWorker(ctx)
	}

	// Scan pending messages from the DB.
	dbScanner := time.NewTicker(readInterval)
	defer dbScanner.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dbScanner.C:
			var (
				pendingMsgs = []models.Message{}
				msgIDs      = m.getOutgoingProcessingMsgIDs()
			)

			// Skip the currently processing msg ids.
			if err := m.q.GetPendingMessages.Select(&pendingMsgs, pq.Array(msgIDs)); err != nil {
				m.lo.Error("error fetching pending messages from db", "error", err)
			}

			// Prepare and push the message to the outgoing queue.
			for _, msg := range pendingMsgs {
				var err error
				msg.Content, _, err = m.templateManager.RenderDefault(map[string]string{
					"Content": msg.Content,
				})
				if err != nil {
					m.lo.Error("error rendering message template", "error", err)
					m.UpdateMessageStatus(msg.UUID, StatusFailed)
					continue
				}
				m.outgoingProcessingMessages.Store(msg.ID, msg.ID)
				m.outgoingMessageQueue <- msg
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
				m.handleDispatchErrors(message, "error fetching inbox", err)
				continue
			}

			// Attach attachments.
			if err := m.attachAttachments(&message); err != nil {
				m.handleDispatchErrors(message, "error fetching inbox", err)
				continue
			}

			// Get from, to addresses and inReplyTo.
			message.From = inbox.FromAddress()
			message.To, _ = m.GetToAddress(message.ConversationID, inbox.Channel())
			message.InReplyTo, _ = m.GetInReplyTo(message.ConversationID)

			// Send.
			err = inbox.Send(message)

			var newStatus = StatusFailed
			if err != nil {
				newStatus = StatusFailed
				m.lo.Error("error sending message", "error", err, "inbox_id", message.InboxID)
			}

			// Update status.
			m.UpdateMessageStatus(message.UUID, newStatus)

			// Update first reply at.
			switch newStatus {
			case StatusSent:
				m.conversationStore.UpdateFirstReplyAt(message.ConversationUUID, message.ConversationID, message.CreatedAt)
			}

			// Broadcast update.
			m.wsHub.BroadcastMessagePropUpdate(message.ConversationUUID, message.UUID, "status" /*message field*/, newStatus)

			// Delete from processing map.
			m.outgoingProcessingMessages.Delete(message.ID)
		}
	}
}

// handleDispatchErrors logs an error, updates the message status to failed,
// and removes the message from the outgoing processing map.
func (m *Manager) handleDispatchErrors(message models.Message, logMessage string, err error) {
	m.lo.Error(logMessage, "error", err, "inbox_id", message.InboxID)
	m.outgoingProcessingMessages.Delete(message.ID)
	m.UpdateMessageStatus(message.UUID, StatusFailed)
}

// GetToAddress retrieves the recipient addresses for a message.
func (m *Manager) GetToAddress(convID int, channel string) ([]string, error) {
	var addr []string
	if err := m.q.GetToAddress.Select(&addr, convID, channel); err != nil {
		m.lo.Error("error fetching to address for msg", "error", err, "conversation_id", convID)
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

// StartDBInserts starts a worker pool to insert incoming messages into the database.
func (m *Manager) StartDBInserts(ctx context.Context, concurrency int) {
	for i := 0; i < concurrency; i++ {
		go m.InsertWorker(ctx)
	}
}

// InsertWorker processes incoming messages and inserts them into the database.
func (m *Manager) InsertWorker(ctx context.Context) {
	for msg := range m.incomingMsgQ {
		select {
		case <-ctx.Done():
			return
		default:
			if err := m.processIncomingMessage(msg); err != nil {
				m.lo.Error("error processing incoming msg", "error", err)
			}
		}
	}
}

// RecordAssigneeUserChange records an activity for a user assignee change.
func (m *Manager) RecordAssigneeUserChange(conversationUUID string, assigneeID int, actor umodels.User) error {
	// Self assign.
	if assigneeID == actor.ID {
		return m.RecordActivity(ActivitySelfAssign, conversationUUID, actor.FullName(), actor)
	}

	// Assignment to another user.
	assignee, err := m.userStore.Get(assigneeID, "")
	if err != nil {
		return err
	}
	return m.RecordActivity(ActivityAssignedUserChange, conversationUUID, assignee.FullName(), actor)
}

// RecordAssigneeTeamChange records an activity for a team assignee change.
func (m *Manager) RecordAssigneeTeamChange(conversationUUID string, teamID int, actor umodels.User) error {
	team, err := m.teamStore.GetTeam(teamID)
	if err != nil {
		return err
	}
	return m.RecordActivity(ActivityAssignedTeamChange, conversationUUID, team.Name, actor)
}

// RecordPriorityChange records an activity for a priority change.
func (m *Manager) RecordPriorityChange(priority, conversationUUID string, actor umodels.User) error {
	return m.RecordActivity(ActivityPriorityChange, conversationUUID, priority, actor)
}

// RecordStatusChange records an activity for a status change.
func (m *Manager) RecordStatusChange(status, conversationUUID string, actor umodels.User) error {
	return m.RecordActivity(ActivityStatusChange, conversationUUID, status, actor)
}

// RecordActivity records an activity message.
func (m *Manager) RecordActivity(activityType, conversationUUID, newValue string, actor umodels.User) error {
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

	// Record message in DB.
	m.RecordMessage(&message)

	// Broadcast the new message.
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

// processIncomingMessage processes an incoming message.
func (m *Manager) processIncomingMessage(in models.IncomingMessage) error {
	var (
		trimmedMsg       = m.TrimMsg(in.Message.Content)
		conversationMeta = map[string]string{
			"subject":         in.Message.Subject,
			"last_message":    trimmedMsg,
			"last_message_at": time.Now().Format(time.RFC3339),
		}
	)

	meta, err := json.Marshal(conversationMeta)
	if err != nil {
		m.lo.Error("error marshalling conversation meta", "error", err)
		return err
	}

	senderID, err := m.contactStore.Upsert(in.Contact)
	if err != nil {
		m.lo.Error("error upserting contact", "error", err)
		return err
	}
	in.Message.SenderID = senderID

	// Check if this message already exists.
	conversationID, err := m.findConversationID([]string{in.Message.SourceID.String})
	if err != nil && err != ErrConversationNotFound {
		return err
	}
	if conversationID > 0 {
		return nil
	}

	isNewConversation, err := m.findOrCreateConversation(&in.Message, in.InboxID, senderID, meta)
	if err != nil {
		return err
	}

	if err = m.RecordMessage(&in.Message); err != nil {
		return fmt.Errorf("inserting conversation message: %w", err)
	}

	if err := m.uploadAttachments(&in.Message); err != nil {
		return fmt.Errorf("uploading message attachments: %w", err)
	}

	// Send WS update.
	if in.Message.ConversationUUID != "" {
		var content string
		if isNewConversation {
			content = m.TrimMsg(in.Message.Subject)
		} else {
			content = m.TrimMsg(in.Message.Content)
		}
		m.BroadcastNewConversationMessage(in.Message, content)
		m.conversationStore.UpdateLastMessage(in.Message.ConversationID, in.Message.ConversationUUID, content, in.Message.CreatedAt)
	}

	// Evaluate automation rules for this conversation.
	if isNewConversation {
		m.automationEngine.EvaluateNewConversationRules(in.Message.ConversationUUID)
	} else {
		m.automationEngine.EvaluateConversationUpdateRules(in.Message.ConversationUUID)
	}

	return nil
}

// MessageExists checks if a message with the given messageID exists.
func (m *Manager) MessageExists(messageID string) (bool, error) {
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

// ProcessIncomingMessage enqueues an incoming message for processing.
func (m *Manager) ProcessIncomingMessage(message models.IncomingMessage) error {
	select {
	case m.incomingMsgQ <- message:
		return nil
	default:
		m.lo.Error("incoming message queue is full")
		return errors.New("incoming message queue is full")
	}
}

// TrimMsg trims and shortens a message content.
func (m *Manager) TrimMsg(msg string) string {
	plain := strings.Trim(strings.TrimSpace(html2text.HTML2Text(msg)), " \t\n\r\v\f")
	if len(plain) > maxLastMessageLen {
		plain = plain[:maxLastMessageLen]
		plain = plain + "..."
	}
	return plain
}

// uploadAttachments uploads attachments for a message.
func (m *Manager) uploadAttachments(in *models.Message) error {
	var (
		hasInline = false
		msgID     = in.ID
		msgUUID   = in.UUID
	)
	for _, att := range in.Attachments {
		reader := bytes.NewReader(att.Content)
		url, _, _, err := m.attachmentManager.Upload(msgUUID, att.Filename, att.ContentType, att.ContentDisposition, att.Size, reader)
		if err != nil {
			m.lo.Error("error uploading attachment", "message_uuid", msgUUID, "error", err)
			return errors.New("error uploading attachments for incoming message")
		}
		if att.ContentDisposition == attachment.DispositionInline {
			hasInline = true
			in.Content = strings.ReplaceAll(in.Content, "cid:"+att.ContentID, url)
		}
	}

	// Update the msg content if the `cid:content_id` URLs have been replaced.
	if hasInline {
		if _, err := m.q.UpdateMessageContent.Exec(in.Content, msgID); err != nil {
			m.lo.Error("error updating message content", "message_uuid", msgUUID)
			return fmt.Errorf("updating msg content: %w", err)
		}
	}
	return nil
}

// findOrCreateConversation finds or creates a conversation for the given message.
func (m *Manager) findOrCreateConversation(in *models.Message, inboxID int, contactID int, meta []byte) (bool, error) {
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
		conversationID, conversationUUID, err = m.conversationStore.Create(contactID, inboxID, meta)
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

// attachAttachments attaches files to a message.
func (m *Manager) attachAttachments(msg *models.Message) error {
	attachments, err := m.attachmentManager.GetMessageAttachments(msg.ID)
	if err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return err
	}

	// Fetch the blobs and attach the attachments to the message.
	for i, att := range attachments {
		attachments[i].Content, err = m.attachmentManager.Store.GetBlob(att.UUID)
		if err != nil {
			m.lo.Error("error fetching blob for attachment", "attachment_uuid", att.UUID, "message_id", msg.ID)
			return err
		}
		attachments[i].Header = attachment.MakeHeaders(att.Filename, "", att.ContentType, att.ContentDisposition)
	}
	msg.Attachments = attachments
	return nil
}

// getOutgoingProcessingMsgIDs returns the IDs of outgoing messages currently being processed.
func (m *Manager) getOutgoingProcessingMsgIDs() []int {
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
func (m *Manager) BroadcastNewConversationMessage(message models.Message, trimmedContent string) {
	m.wsHub.BroadcastNewConversationMessage(message.ConversationUUID, trimmedContent, message.UUID, time.Now().Format(time.RFC3339), message.Private)
}
