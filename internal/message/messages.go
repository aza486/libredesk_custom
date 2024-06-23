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
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/user"

	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/message/models"
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
	q                      queries
	lo                     *logf.Logger
	contactMgr             *contact.Manager
	attachmentMgr          *attachment.Manager
	conversationMgr        *conversation.Manager
	inboxMgr               *inbox.Manager
	userMgr                *user.Manager
	teamMgr                *team.Manager
	automationEngine       *automation.Engine
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
	InsertMessage        *sqlx.Stmt `query:"insert-message"`
	UpdateMessageContent *sqlx.Stmt `query:"update-message-content"`
	UpdateMessageStatus  *sqlx.Stmt `query:"update-message-status"`
	MessageExists        *sqlx.Stmt `query:"message-exists"`
}

func New(incomingMsgQ chan models.IncomingMessage,
	wsHub *ws.Hub,
	userMgr *user.Manager,
	teamMgr *team.Manager,
	contactMgr *contact.Manager,
	attachmentMgr *attachment.Manager,
	inboxMgr *inbox.Manager,
	conversationMgr *conversation.Manager,
	automationEngine *automation.Engine,
	opts Opts) (*Manager, error) {
	var q queries

	if err := dbutils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:                      q,
		lo:                     opts.Lo,
		wsHub:                  wsHub,
		userMgr:                userMgr,
		teamMgr:                teamMgr,
		contactMgr:             contactMgr,
		attachmentMgr:          attachmentMgr,
		conversationMgr:        conversationMgr,
		inboxMgr:               inboxMgr,
		automationEngine:       automationEngine,
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
			m.lo.Info("context cancelled while sending messages.. Stopping dispatcher.")
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
			m.lo.Debug("set in reply to for outgoing email message", "in_reply_to", msg.InReplyTo)
		}

		err = inbox.Send(msg)

		var newStatus = StatusSent
		if err != nil {
			newStatus = StatusFailed
			m.lo.Error("error sending message", "error", err, "inbox_id", msg.InboxID)
		}

		if _, err := m.q.UpdateMessageStatus.Exec(newStatus, msg.UUID); err != nil {
			m.lo.Error("error updating message status in DB", "error", err, "inbox_id", msg.InboxID)
		}

		switch newStatus {
		case StatusSent:
			m.conversationMgr.UpdateFirstReplyAt(msg.ConversationID, "", msg.CreatedAt)
		}

		// Broadcast the new message status.
		m.wsHub.BroadcastMsgStatus(msg.ConversationUUID, map[string]interface{}{
			"uuid":              msg.UUID,
			"conversation_uuid": msg.ConversationUUID,
			"status":            newStatus,
		})

		m.outgoingProcessingMsgs.Delete(msg.ID)
	}
}

func (m *Manager) GetToAddress(convID int, channel string) ([]string, error) {
	var addr []string
	if err := m.q.GetToAddress.Select(&addr, convID, channel); err != nil {
		m.lo.Error("error fetching to address for msg", "error", err, "conv_id", convID)
		return addr, err
	}
	return addr, nil
}

func (m *Manager) GetInReplyTo(convID int) (string, error) {
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
			return
		default:
			if err := m.processIncomingMessage(msg); err != nil {
				m.lo.Error("error processing incoming msg", "error", err)
			}
		}
	}
}

func (m *Manager) RecordAssigneeUserChange(updatedValue, convUUID, actorUUID string) error {
	if updatedValue == actorUUID {
		return m.RecordActivity(ActivitySelfAssign, updatedValue, convUUID, actorUUID)
	}
	assignee, err := m.userMgr.GetUser(0, updatedValue)
	if err != nil {
		m.lo.Error("Error fetching user to record assignee change", "error", err)
		return err
	}
	updatedValue = assignee.FullName()
	return m.RecordActivity(ActivityAssignedUserChange, updatedValue, convUUID, actorUUID)
}

func (m *Manager) RecordAssigneeTeamChange(updatedValue, convUUID, actorUUID string) error {
	team, err := m.teamMgr.GetTeam(updatedValue)
	if err != nil {
		return err
	}
	updatedValue = team.Name
	return m.RecordActivity(ActivityAssignedTeamChange, updatedValue, convUUID, actorUUID)
}

func (m *Manager) RecordPriorityChange(updatedValue, convUUID, actorUUID string) error {
	return m.RecordActivity(ActivityPriorityChange, updatedValue, convUUID, actorUUID)
}

func (m *Manager) RecordStatusChange(updatedValue, convUUID, actorUUID string) error {
	return m.RecordActivity(ActivityStatusChange, updatedValue, convUUID, actorUUID)
}

func (m *Manager) RecordActivity(activityType, updatedValue, conversationUUID, actorUUID string) error {
	var (
		actor, err = m.userMgr.GetUser(0, actorUUID)
	)
	if err != nil {
		m.lo.Error("Error fetching user for recording message activity", "error", err)
		return err
	}

	var content = m.getActivityContent(activityType, updatedValue, actor.FullName())
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
	m.BroadcastNewMsg(msg, "")

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

	if err = m.findOrCreateConversation(&in.Message, in.InboxID, senderID, convMetaJSON); err != nil {
		m.lo.Error("error creating conversation", "error", err)
		return err
	}

	if err = m.RecordMessage(&in.Message); err != nil {
		m.lo.Error("error inserting conversation message", "error", err)
		return fmt.Errorf("inserting conversation message: %w", err)
	}

	if err := m.uploadAttachments(&in.Message); err != nil {
		m.lo.Error("error uploading message attachments", "msg_uuid", in.Message.UUID, "error", err)
		return fmt.Errorf("uploading message attachments: %w", err)
	}

	// Send WS update.
	if in.Message.ConversationUUID > "" {
		m.BroadcastNewMsg(in.Message, "")
	}

	// Evaluate automation rules for this conversation.
	if in.Message.NewConversation {
		m.automationEngine.EvaluateRules(cmodels.Conversation{
			UUID:         in.Message.ConversationUUID,
			FirstMessage: in.Message.Content,
			Subject:      in.Message.Subject,
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

func (m *Manager) findOrCreateConversation(in *models.Message, inboxID int, contactID int, meta []byte) error {
	var (
		conversationID   int
		conversationUUID string
		newConv          bool
		err              error
	)

	// Search for existing conversation.
	if conversationID == 0 && in.InReplyTo > "" {
		conversationID, err = m.findConversationID([]string{in.InReplyTo})
		if err != nil && err != ErrConversationNotFound {
			return err
		}
	}
	if conversationID == 0 && len(in.References) > 0 {
		conversationID, err = m.findConversationID(in.References)
		if err != nil && err != ErrConversationNotFound {
			return err
		}
	}

	// Conversation not found, create one.
	if conversationID == 0 {
		newConv = true
		conversationID, err = m.conversationMgr.Create(contactID, inboxID, meta)
		if err != nil || conversationID == 0 {
			return fmt.Errorf("inserting conversation: %w", err)
		}
	}

	// Fetch & return the UUID of the conversation for UI updates.
	conversationUUID, err = m.conversationMgr.GetUUID(conversationID)
	if err != nil {
		m.lo.Error("Error fetching conversation UUID from id", err)
	}

	in.ConversationID = conversationID
	in.ConversationUUID = conversationUUID
	in.NewConversation = newConv

	return nil
}

// findConversationID finds the conversation ID from the message source ID.
func (m *Manager) findConversationID(sourceIDs []string) (int, error) {
	var conversationID int
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

// getOutgoingProcessingMsgIDs returns outgoing msg ids currently being processed.
func (m *Manager) getOutgoingProcessingMsgIDs() []int {
	var out = make([]int, 0)
	m.outgoingProcessingMsgs.Range(func(key, _ any) bool {
		if k, ok := key.(int); ok {
			out = append(out, k)
		}
		return true
	})
	return out
}

func (m *Manager) BroadcastNewMsg(msg models.Message, trimmedContent string) {
	if trimmedContent == "" {
		var content = ""
		if msg.NewConversation {
			content = m.TrimMsg(msg.Subject)
		} else {
			content = m.TrimMsg(msg.Content)
		}
		trimmedContent = content
	}
	m.wsHub.BroadcastNewMsg(msg.ConversationUUID, map[string]interface{}{
		"conversation_uuid": msg.ConversationUUID,
		"uuid":              msg.UUID,
		"last_message":      trimmedContent,
		"last_message_at":   time.Now().Format(time.DateTime),
		"private":           msg.Private,
	})
}
