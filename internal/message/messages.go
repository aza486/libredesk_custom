package message

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/message/models"
	"github.com/abhinavxd/artemis/internal/utils"
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

	SenderTypeAgent   = "agent"
	SenderTypeContact = "contact"

	StatusPending   = "pending"
	StatusSent      = "sent"
	StatusDelivered = "delivered"
	StatusRead      = "read"
	StatusFailed    = "failed"
	StatusReceived  = "received"

	ActivityStatusChange        = "status_change"
	ActivityPriorityChange      = "priority_change"
	ActivityAssignedAgentChange = "assigned_agent_change"
	ActivityAssignedTeamChange  = "assigned_team_change"
	ActivitySelfAssign          = "self_assign"
	ActivityTagChange           = "tag_change"

	ContentTypeText = "text"
	ContentTypeHTML = "html"

	maxLastMessageLen = 45
)

type Manager struct {
	q                      queries
	lo                     *logf.Logger
	contactMgr             *contact.Manager
	attachmentMgr          *attachment.Manager
	conversationMgr        *conversation.Manager
	inboxMgr               *inbox.Manager
	wsHub                  *ws.Hub
	incomingMsgQ           chan models.IncomingMessage
	outgoingMsgQ           chan models.Message
	outgoingProcessingMsgs sync.Map
}

type Opts struct {
	DB                     *sqlx.DB
	Lo                     *logf.Logger
	OutgoingMsgQueueSize   int
	OutgoingMsgConcurrency int
}

type queries struct {
	GetMessage           *sqlx.Stmt `query:"get-message"`
	GetMessages          *sqlx.Stmt `query:"get-messages"`
	GetToAddress         *sqlx.Stmt `query:"get-to-address"`
	GetInReplyTo         *sqlx.Stmt `query:"get-in-reply-to"`
	GetPendingMessages   *sqlx.Stmt `query:"get-pending-messages"`
	InsertMessageByUUID  *sqlx.Stmt `query:"insert-message-by-uuid"`
	InsertMessageByID    *sqlx.Stmt `query:"insert-message-by-id"`
	UpdateMessageContent *sqlx.Stmt `query:"update-message-content"`
	UpdateMessageStatus  *sqlx.Stmt `query:"update-message-status"`
	MessageExists        *sqlx.Stmt `query:"message-exists"`
}

func New(incomingMsgQ chan models.IncomingMessage, wsHub *ws.Hub, contactMgr *contact.Manager, attachmentMgr *attachment.Manager, inboxMgr *inbox.Manager, conversationMgr *conversation.Manager, opts Opts) (*Manager, error) {
	var q queries

	if err := utils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:                      q,
		lo:                     opts.Lo,
		wsHub:                  wsHub,
		contactMgr:             contactMgr,
		attachmentMgr:          attachmentMgr,
		conversationMgr:        conversationMgr,
		inboxMgr:               inboxMgr,
		incomingMsgQ:           incomingMsgQ,
		outgoingMsgQ:           make(chan models.Message, opts.OutgoingMsgQueueSize),
		outgoingProcessingMsgs: sync.Map{},
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
func (m *Manager) RecordMessage(msg models.Message) (int64, string, error) {
	var (
		msgID          int64
		msgUUID        string
		query          *sqlx.Stmt
		convIdentifier interface{}
	)

	// Set the query to be used.
	if msg.ConversationUUID != "" {
		query = m.q.InsertMessageByUUID
		convIdentifier = msg.ConversationUUID
	} else if msg.ConversationID > 0 {
		query = m.q.InsertMessageByID
		convIdentifier = msg.ConversationID
	} else {
		return msgID, msgUUID, errors.New("conversation id or uuid is required to insert a message")
	}

	if err := query.QueryRow(msg.Type, msg.Status, convIdentifier, msg.Content, msg.SenderID, msg.SenderType, msg.Private, msg.ContentType, msg.SourceID, msg.InboxID, msg.Meta).Scan(&msgID, &msgUUID); err != nil {
		m.lo.Error("inserting message in db", "error", err)
		return msgID, msgUUID, fmt.Errorf("inserting message: %w", err)
	}

	// Attach the attachments.
	if err := m.attachmentMgr.AttachMessage(msg.Attachments, msgID); err != nil {
		m.lo.Error("error attaching attachments to the message", "error", err)
		return msgID, msgUUID, errors.New("error attaching attachments to the message")
	}

	return msgID, msgUUID, nil
}

func (m *Manager) RecordActivity(activityType, value, conversationUUID, userName string, userID int64) error {
	var content = m.getActivityContent(activityType, value, userName)
	if content == "" {
		m.lo.Error("invalid activity for inserting message", "activity", activityType)
		return errors.New("invalid activity type for inserting message")
	}

	m.RecordMessage(models.Message{
		Type:             TypeActivity,
		Status:           StatusSent,
		Content:          content,
		ContentType:      ContentTypeText,
		ConversationUUID: conversationUUID,
		Private:          true,
		SenderID:         userID,
		SenderType:       "user",
		Meta:             "{}",
	})

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
			m.lo.Info("context cancelled while sending messages.. Stopping dispatcher.")
			return
		case <-dbScanner.C:
			var (
				pendingMsgs = []models.Message{}
				msgIDs      = m.getProcessingMsgIDs()
			)

			// Skip the currently processing msg ids.
			if err := m.q.GetPendingMessages.Select(&pendingMsgs, pq.Array(msgIDs)); err != nil {
				m.lo.Error("error fetching pending messages from db", "error", err)
			}

			fmt.Printf("pendings msg %+v \n", pendingMsgs)

			// Prepare and push the message to the outgoing queue.
			for _, msg := range pendingMsgs {
				m.outgoingProcessingMsgs.Store(msg.ID, msg.ID)
				m.outgoingMsgQ <- msg
			}
		}
	}
}

func (m *Manager) DispatchWorker() {
	for msg := range m.outgoingMsgQ {
		inbox, err := m.inboxMgr.GetInbox(msg.InboxID)
		if err != nil {
			m.lo.Error("error fetching inbox", "error", err, "inbox_id", msg.InboxID)
			m.outgoingProcessingMsgs.Delete(msg.ID)
			continue
		}

		msg.From = inbox.FromAddress()

		if err := m.attachAttachments(&msg); err != nil {
			m.lo.Error("error attaching attachments to msg", "error", err)
			m.outgoingProcessingMsgs.Delete(msg.ID)
			continue
		}

		msg.To, _ = m.GetToAddress(msg.ConversationID, inbox.Channel())

		if inbox.Channel() == "email" {
			msg.InReplyTo, _ = m.GetInReplyTo(msg.ConversationID)
			m.lo.Debug("set in reply to", "in_reply_to", msg.InReplyTo)
		}

		err = inbox.Send(msg)
		var newStatus = StatusSent
		if err != nil {
			newStatus = StatusFailed
			m.lo.Error("error sending message", "error", err, "inbox_id", msg.InboxID)
		}

		// Broadcast the new message status.
		m.wsHub.BroadcastMsgStatus(msg.ConversationUUID, map[string]interface{}{
			"uuid":              msg.UUID,
			"conversation_uuid": msg.ConversationUUID,
			"status":            newStatus,
		})

		if _, err := m.q.UpdateMessageStatus.Exec(newStatus, msg.UUID); err != nil {
			m.lo.Error("error updating message status in DB", "error", err, "inbox_id", msg.InboxID)
		}

		m.outgoingProcessingMsgs.Delete(msg.ID)
	}
}

func (m *Manager) GetToAddress(convID int64, channel string) ([]string, error) {
	var addr []string
	if err := m.q.GetToAddress.Select(&addr, convID, channel); err != nil {
		m.lo.Error("error fetching to address for msg", "error", err, "conv_id", convID)
		return addr, err
	}
	return addr, nil
}

func (m *Manager) GetInReplyTo(convID int64) (string, error) {
	var out string
	if err := m.q.GetInReplyTo.Get(&out, convID); err != nil {
		if err == sql.ErrNoRows {
			m.lo.Error("in reply to not found", "error", err, "conv_id", convID)
			return out, nil
		}
		m.lo.Error("error fetching in reply to", "error", err, "conv_id", convID)
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
			m.lo.Info("context cancelled while inserting messages into the DB.")
		default:
			if err := m.processIncomingMessage(msg); err != nil {
				m.lo.Error("error processing incoming msg", "error", err)
			}
		}
	}
}

func (m *Manager) processIncomingMessage(in models.IncomingMessage) error {
	var (
		conversationMeta = "{}"
		err              error
	)

	if in.Message.Subject != "" {
		conversationMeta = fmt.Sprintf(`{"subject": "%s"}`, in.Message.Subject)
	}

	in.Message.SenderID, err = m.contactMgr.Upsert(in.Contact)
	if err != nil {
		m.lo.Error("error upserting contact", "error", err)
		return err
	}

	// This message already exists?
	m.lo.Debug("searching for message with id", "source_id", in.Message.SourceID.String)
	conversationID, err := m.findConversationID([]string{in.Message.SourceID.String})
	if err != nil && err != ErrConversationNotFound {
		return err
	}
	if conversationID > 0 {
		m.lo.Debug("conversation already exists for message", "source_id", in.Message.SourceID.String)
		return nil
	}

	conversationID, newConv, err := m.findOrCreateConversation(&in.Message, in.InboxID, in.Message.SenderID, conversationMeta)
	if err != nil {
		m.lo.Error("error creating conversation", "error", err)
		return err
	}
	in.Message.ConversationID = conversationID

	in.Message.ID, in.Message.UUID, err = m.RecordMessage(in.Message)
	if err != nil {
		m.lo.Error("error inserting conversation message", "error", err)
		return fmt.Errorf("inserting conversation message: %w", err)
	}

	if err := m.uploadAttachments(&in.Message); err != nil {
		m.lo.Error("error uploading message attachments", "msg_uuid", in.Message.UUID, "error", err)
		return fmt.Errorf("uploading message attachments: %w", err)
	}

	// WS update.
	cuuid, err := m.conversationMgr.GetUUID(in.Message.ConversationID)
	if err != nil {
		m.lo.Error("error fetching uuid for conversation", "conversation_id", in.Message.ConversationID, "error", err)
	} else if cuuid > "" {
		// Broastcast new msg to all conversation subscribers.
		content := m.TrimMsg(in.Message.Content)
		if newConv {
			content = m.TrimMsg(in.Message.Subject)
		}
		now := time.Now().Format(time.DateTime)
		m.wsHub.BroadcastNewMsg(cuuid, map[string]interface{}{
			"conversation_uuid": cuuid,
			"uuid":              in.Message.UUID,
			"last_message":      content,
			"first_name":        in.Contact.FirstName,
			"last_name":         in.Contact.LastName,
			"avatar_url":        "",
			"last_message_at":   now,
			"private":           false,
		})
	}

	return nil
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
		inlineAttachments = false
		msgID             = in.ID
		msgUUID           = in.UUID
	)
	for _, att := range in.Attachments {
		reader := bytes.NewReader(att.Content)
		url, _, _, err := m.attachmentMgr.Upload(msgUUID, att.Filename, att.ContentType, att.ContentDisposition, att.Size, reader)
		if err != nil {
			m.lo.Error("error uploading attachment", "message_uuid", msgUUID, "error", err)
			return errors.New("error uploading attachments for incoming message")
		}
		if att.ContentDisposition == attachment.DispositionInline {
			inlineAttachments = true
			in.Content = strings.ReplaceAll(in.Content, "cid:"+att.ContentID, url)
		}
	}

	// Update the msg content the `cid:content_id` urls have been replaced.
	if inlineAttachments {
		if _, err := m.q.UpdateMessageContent.Exec(in.Content, msgID); err != nil {
			m.lo.Error("error updating message content", "message_uuid", msgUUID)
			return fmt.Errorf("updating msg content: %w", err)
		}
	}
	return nil
}

func (m *Manager) findOrCreateConversation(in *models.Message, inboxID int, contactID int64, conversationMeta string) (int64, bool, error) {
	var (
		conversationID int64
		newConv        bool
		err            error
	)

	// Search for existing conversation.
	if conversationID == 0 && in.InReplyTo > "" {
		m.lo.Debug("searching for message with id", "source_id", in.InReplyTo)
		conversationID, err = m.findConversationID([]string{in.InReplyTo})
		if err != nil && err != ErrConversationNotFound {
			return conversationID, newConv, err
		}
	}
	if conversationID == 0 && len(in.References) > 0 {
		m.lo.Debug("searching for message with id", "source_id", in.References)
		conversationID, err = m.findConversationID(in.References)
		if err != nil && err != ErrConversationNotFound {
			return conversationID, newConv, err
		}
	}

	// Conversation not found, create one.
	if conversationID == 0 {
		newConv = true
		conversationID, err = m.conversationMgr.Create(contactID, inboxID, conversationMeta)
		if err != nil || conversationID == 0 {
			return conversationID, newConv, fmt.Errorf("inserting conversation: %w", err)
		}
	}
	return conversationID, newConv, nil
}

func (m *Manager) getActivityContent(activityType, value, userName string) string {
	var content = ""
	switch activityType {
	case ActivityAssignedAgentChange:
		content = fmt.Sprintf("Assigned to %s by %s", value, userName)
	case ActivityAssignedTeamChange:
		content = fmt.Sprintf("Assigned to %s team by %s", value, userName)
	case ActivitySelfAssign:
		content = fmt.Sprintf("%s self-assigned this conversation", userName)
	case ActivityPriorityChange:
		content = fmt.Sprintf("%s changed priority to %s", userName, value)
	case ActivityStatusChange:
		content = fmt.Sprintf("%s marked the conversation as %s", userName, value)
	case ActivityTagChange:
		content = fmt.Sprintf("%s added tags %s", userName, value)
	}
	return content
}

// findConversationID finds the conversation ID from the message source ID.
func (m *Manager) findConversationID(sourceIDs []string) (int64, error) {
	var conversationID int64
	if err := m.q.MessageExists.QueryRow(pq.Array(sourceIDs)).Scan(&conversationID); err != nil {
		if err == sql.ErrNoRows {
			return conversationID, ErrConversationNotFound
		} else {
			m.lo.Error("error checking for existing message", "error", err)
			return conversationID, fmt.Errorf("checking for existing message: %w", err)
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

// getProcessingMsgIDs returns outgoing msg ids currently being processed.
func (m *Manager) getProcessingMsgIDs() []int64 {
	var out = make([]int64, 0)
	m.outgoingProcessingMsgs.Range(func(key, _ any) bool {
		if k, ok := key.(int64); ok {
			out = append(out, k)
		}
		return true
	})
	return out
}
