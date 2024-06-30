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
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/message/models"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/template"
	"github.com/abhinavxd/artemis/internal/user"
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

type Manager struct {
	q                          queries
	lo                         *logf.Logger
	contactMgr                 *contact.Manager
	attachmentMgr              *attachment.Manager
	conversationMgr            *conversation.Manager
	inboxMgr                   *inbox.Manager
	userMgr                    *user.Manager
	teamMgr                    *team.Manager
	automationEngine           *automation.Engine
	wsHub                      *ws.Hub
	templateManager            *template.Manager
	incomingMsgQ               chan models.IncomingMessage
	outgoingMessageQueue       chan models.Message
	outgoingProcessingMessages sync.Map
}

type Opts struct {
	DB                   *sqlx.DB
	Lo                   *logf.Logger
	IncomingMsgQueueSize int
	OutgoingMsgQueueSize int
}

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

func New(
	wsHub *ws.Hub,
	userMgr *user.Manager,
	teamMgr *team.Manager,
	contactMgr *contact.Manager,
	attachmentMgr *attachment.Manager,
	inboxMgr *inbox.Manager,
	conversationMgr *conversation.Manager,
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
		userMgr:                    userMgr,
		teamMgr:                    teamMgr,
		contactMgr:                 contactMgr,
		attachmentMgr:              attachmentMgr,
		conversationMgr:            conversationMgr,
		inboxMgr:                   inboxMgr,
		automationEngine:           automationEngine,
		templateManager:            templateManager,
		incomingMsgQ:               make(chan models.IncomingMessage, opts.IncomingMsgQueueSize),
		outgoingMessageQueue:       make(chan models.Message, opts.OutgoingMsgQueueSize),
		outgoingProcessingMessages: sync.Map{},
	}, nil
}

func (m *Manager) GetConvMessages(uuid string) ([]models.Message, error) {
	var messages []models.Message
	if err := m.q.GetMessages.Select(&messages, uuid); err != nil {
		m.lo.Error("fetching messages from DB", "conversation_uuid", uuid, "error", err)
		return nil, fmt.Errorf("error fetching messages")
	}
	return messages, nil
}

func (m *Manager) GetMessage(uuid string) ([]models.Message, error) {
	var messages []models.Message
	if err := m.q.GetMessage.Select(&messages, uuid); err != nil {
		m.lo.Error("fetching messages from DB", "conversation_uuid", uuid, "error", err)
		return nil, fmt.Errorf("error fetching messages")
	}
	return messages, nil
}

func (m *Manager) UpdateMessageStatus(uuid string, status string) error {
	if _, err := m.q.UpdateMessageStatus.Exec(status, uuid); err != nil {
		m.lo.Error("error updating message status in DB", "error", err, "uuid", uuid)
		return err
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
		return fmt.Errorf("inserting message: %w", err)
	}

	// Attach the attachments.
	if err := m.attachmentMgr.AttachMessage(msg.Attachments, msg.ID); err != nil {
		m.lo.Error("error attaching attachments to the message", "error", err)
		return errors.New("error attaching attachments to the message")
	}

	return nil
}

// StartDispatcher is a blocking function that must be invoked with a goroutine.
// It scans DB per second for pending outbound messages and sends them using the inbox channel.
func (m *Manager) StartDispatcher(ctx context.Context, concurrency int, readInterval time.Duration) {
	// Spawn goroutine worker pool.
	for range concurrency {
		go m.DispatchWorker()
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

func (m *Manager) DispatchWorker() {
	for message := range m.outgoingMessageQueue {
		inbox, err := m.inboxMgr.GetInbox(message.InboxID)
		if err != nil {
			m.lo.Error("error fetching inbox", "error", err, "inbox_id", message.InboxID)
			m.outgoingProcessingMessages.Delete(message.ID)
			continue
		}

		message.From = inbox.FromAddress()

		if err := m.attachAttachments(&message); err != nil {
			m.lo.Error("error attaching attachments to message", "error", err)
			m.outgoingProcessingMessages.Delete(message.ID)
			continue
		}

		message.To, _ = m.GetToAddress(message.ConversationID, inbox.Channel())

		if inbox.Channel() == "email" {
			message.InReplyTo, _ = m.GetInReplyTo(message.ConversationID)
		}

		err = inbox.Send(message)

		var newStatus = StatusSent
		if err != nil {
			newStatus = StatusFailed
			m.lo.Error("error sending message", "error", err, "inbox_id", message.InboxID)
		}

		if _, err := m.q.UpdateMessageStatus.Exec(newStatus, message.UUID); err != nil {
			m.lo.Error("error updating message status in DB", "error", err, "inbox_id", message.InboxID)
		}

		switch newStatus {
		case StatusSent:
			m.conversationMgr.UpdateFirstReplyAt(message.ConversationUUID, message.ConversationID, message.CreatedAt)
		}

		// Broadcast message status update to the subscribers.
		m.wsHub.BroadcastMessagePropUpdate(message.ConversationUUID, message.UUID, "status" /*message field*/, newStatus)

		// Remove message from the processing list.
		m.outgoingProcessingMessages.Delete(message.ID)
	}
}

func (m *Manager) GetToAddress(convID int, channel string) ([]string, error) {
	var addr []string
	if err := m.q.GetToAddress.Select(&addr, convID, channel); err != nil {
		m.lo.Error("error fetching to address for msg", "error", err, "conversation_id", convID)
		return addr, err
	}
	return addr, nil
}

func (m *Manager) GetInReplyTo(convID int) (string, error) {
	var out string
	if err := m.q.GetInReplyTo.Get(&out, convID); err != nil {
		if err == sql.ErrNoRows {
			m.lo.Error("in reply to not found", "error", err, "conversation_id", convID)
			return out, nil
		}
		m.lo.Error("error fetching in reply to", "error", err, "conversation_id", convID)
		return out, err
	}
	return out, nil
}

// StartDBInserts is a blocking function that must be invoked with a goroutine,
// it spawns worker pools for inserting incoming messages.
func (m *Manager) StartDBInserts(ctx context.Context, concurrency int) {
	for range concurrency {
		go m.InsertWorker(ctx)
	}
}

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

func (m *Manager) RecordAssigneeUserChange(conversationUUID, assigneeUUID, actorUUID string) error {
	// Self assign.
	if assigneeUUID == actorUUID {
		return m.RecordActivity(ActivitySelfAssign, assigneeUUID, conversationUUID, actorUUID)
	}
	assignee, err := m.userMgr.GetUser(0, assigneeUUID)
	if err != nil {
		m.lo.Error("Error fetching user to record assignee change", "conversation_uuid", conversationUUID, "actor_uuid", actorUUID, "error", err)
		return err
	}
	return m.RecordActivity(ActivityAssignedUserChange, assignee.FullName() /*new_value*/, conversationUUID, actorUUID)
}

func (m *Manager) RecordAssigneeTeamChange(conversationUUID, value, actorUUID string) error {
	team, err := m.teamMgr.GetTeam(value)
	if err != nil {
		return err
	}
	return m.RecordActivity(ActivityAssignedTeamChange, team.Name /*new_value*/, conversationUUID, actorUUID)
}

func (m *Manager) RecordPriorityChange(updatedValue, conversationUUID, actorUUID string) error {
	return m.RecordActivity(ActivityPriorityChange, updatedValue, conversationUUID, actorUUID)
}

func (m *Manager) RecordStatusChange(updatedValue, conversationUUID, actorUUID string) error {
	return m.RecordActivity(ActivityStatusChange, updatedValue, conversationUUID, actorUUID)
}

func (m *Manager) RecordActivity(activityType, newValue, conversationUUID, actorUUID string) error {
	var (
		actor, err = m.userMgr.GetUser(0, actorUUID)
	)
	if err != nil {
		m.lo.Error("Error fetching user for recording message activity", "error", err)
		return err
	}

	var content = m.getActivityContent(activityType, newValue, actor.FullName())
	if content == "" {
		m.lo.Error("Error invalid activity for recording activity", "activity", activityType)
		return errors.New("invalid activity type for recording activity")
	}

	msg := models.Message{
		Type:             TypeActivity,
		Status:           StatusSent,
		Content:          content,
		ContentType:      ContentTypeText,
		ConversationUUID: conversationUUID,
		Private:          true,
		SenderID:         actor.ID,
		SenderType:       SenderTypeUser,
		Meta:             "{}",
	}

	m.RecordMessage(&msg)
	m.BroadcastNewConversationMessage(msg, content)
	m.conversationMgr.UpdateLastMessage(0, conversationUUID, content, msg.CreatedAt)
	return nil
}

func (m *Manager) getActivityContent(activityType, newValue, actorName string) string {
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
	}
	return content
}

func (m *Manager) processIncomingMessage(in models.IncomingMessage) error {
	var (
		trimmedMsg = m.TrimMsg(in.Message.Content)
		convMeta   = map[string]string{
			"subject":         in.Message.Subject,
			"last_message":    trimmedMsg,
			"last_message_at": time.Now().Format(time.RFC3339),
		}
	)

	convMetaJSON, err := json.Marshal(convMeta)
	if err != nil {
		m.lo.Error("error marshalling conversation meta", "error", err)
		return err
	}

	senderID, err := m.contactMgr.Upsert(in.Contact)
	if err != nil {
		m.lo.Error("error upserting contact", "error", err)
		return err
	}
	in.Message.SenderID = senderID

	// This message already exists? Return.
	conversationID, err := m.findConversationID([]string{in.Message.SourceID.String})
	if err != nil && err != ErrConversationNotFound {
		return err
	}
	if conversationID > 0 {
		return nil
	}

	isNewConversation, err := m.findOrCreateConversation(&in.Message, in.InboxID, senderID, convMetaJSON)
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
	if in.Message.ConversationUUID > "" {
		var content = ""
		if isNewConversation {
			content = m.TrimMsg(in.Message.Subject)
		} else {
			content = m.TrimMsg(in.Message.Content)
		}
		m.BroadcastNewConversationMessage(in.Message, content)
		m.conversationMgr.UpdateLastMessage(in.Message.ConversationID, in.Message.ConversationUUID, content, in.Message.CreatedAt)
	}

	// Evaluate automation rules for this new conversation.
	if isNewConversation {
		m.automationEngine.EvaluateRules(cmodels.Conversation{
			UUID:         in.Message.ConversationUUID,
			FirstMessage: in.Message.Content,
			Subject:      in.Message.Subject,
		})
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

// ProcessMessage enqueues an incoming message for processing.
func (m *Manager) ProcessMessage(message models.IncomingMessage) error {
	select {
	case m.incomingMsgQ <- message:
		return nil
	default:
		return fmt.Errorf("failed to enqueue message: %v", message.Message.ID)
	}
}

func (m *Manager) TrimMsg(msg string) string {
	plain := strings.Trim(strings.TrimSpace(html2text.HTML2Text(msg)), " \t\n\r\v\f")
	if len(plain) > maxLastMessageLen {
		plain = plain[:maxLastMessageLen]
		plain = plain + "..."
	}
	return plain
}

func (m *Manager) uploadAttachments(in *models.Message) error {
	var (
		hasInline = false
		msgID     = in.ID
		msgUUID   = in.UUID
	)
	for _, att := range in.Attachments {
		reader := bytes.NewReader(att.Content)
		url, _, _, err := m.attachmentMgr.Upload(msgUUID, att.Filename, att.ContentType, att.ContentDisposition, att.Size, reader)
		if err != nil {
			m.lo.Error("error uploading attachment", "message_uuid", msgUUID, "error", err)
			return errors.New("error uploading attachments for incoming message")
		}
		if att.ContentDisposition == attachment.DispositionInline {
			hasInline = true
			in.Content = strings.ReplaceAll(in.Content, "cid:"+att.ContentID, url)
		}
	}

	// Update the msg content the `cid:content_id` urls have been replaced.
	if hasInline {
		if _, err := m.q.UpdateMessageContent.Exec(in.Content, msgID); err != nil {
			m.lo.Error("error updating message content", "message_uuid", msgUUID)
			return fmt.Errorf("updating msg content: %w", err)
		}
	}
	return nil
}

func (m *Manager) findOrCreateConversation(in *models.Message, inboxID int, contactID int, meta []byte) (bool, error) {
	var (
		new              bool
		err              error
		conversationID   int
		conversationUUID string
	)

	// Search for existing conversation.
	sourceIDs := in.References
	if in.InReplyTo > "" {
		sourceIDs = append(sourceIDs, in.InReplyTo)
	}
	conversationID, err = m.findConversationID(sourceIDs)
	if err != nil && err != ErrConversationNotFound {
		return new, err
	}

	// Conversation not found, create one.
	if conversationID == 0 {
		new = true
		conversationID, conversationUUID, err = m.conversationMgr.Create(contactID, inboxID, meta)
		if err != nil || conversationID == 0 {
			return new, err
		}
		in.ConversationID = conversationID
		in.ConversationUUID = conversationUUID
		return new, nil
	}

	// Set UUID if not available.
	if conversationUUID == "" {
		conversationUUID, err = m.conversationMgr.GetUUID(conversationID)
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
		} else {
			m.lo.Error("error fetching msg from DB", "error", err)
			return conversationID, err
		}
	}
	return conversationID, nil
}

func (m *Manager) attachAttachments(msg *models.Message) error {
	var attachments, err = m.attachmentMgr.GetMessageAttachments(msg.ID)
	if err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return err
	}

	// TODO: set attachment headers and replace the inline image src url w
	// src="cid:ii_lxxsfhtp0"
	// a.Header.Set("Content-Disposition", "inline")
	// a.Header.Set("Content-ID", "<"+f.CID+">")

	// Fetch the blobs and attach the attachments to the message.
	for i, att := range attachments {
		attachments[i].Content, err = m.attachmentMgr.Store.GetBlob(att.UUID)
		if err != nil {
			m.lo.Error("error fetching blob for attachment", "attachment_uuid", att.UUID, "message_id", msg.ID)
			return err
		}
		attachments[i].Header = attachment.MakeHeaders(att.Filename, "", att.ContentType, att.ContentDisposition)
	}
	msg.Attachments = attachments
	return nil
}

// getOutgoingProcessingMsgIDs returns outgoing msg ids currently being processed.
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

func (m *Manager) BroadcastNewConversationMessage(message models.Message, trimmedContent string) {
	m.wsHub.BroadcastNewConversationMessage(message.ConversationUUID, trimmedContent, message.UUID, time.Now().Format(time.RFC3339), message.Private)
}
