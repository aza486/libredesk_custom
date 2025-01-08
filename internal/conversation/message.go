package conversation

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	amodels "github.com/abhinavxd/artemis/internal/automation/models"
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/inbox"
	mmodels "github.com/abhinavxd/artemis/internal/media/models"
	"github.com/abhinavxd/artemis/internal/stringutil"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/lib/pq"
)

const (
	MessageIncoming = "incoming"
	MessageOutgoing = "outgoing"
	MessageActivity = "activity"

	SenderTypeUser    = "user"
	SenderTypeContact = "contact"

	MessageStatusPending  = "pending"
	MessageStatusSent     = "sent"
	MessageStatusFailed   = "failed"
	MessageStatusReceived = "received"

	ActivityStatusChange       = "status_change"
	ActivityPriorityChange     = "priority_change"
	ActivityAssignedUserChange = "assigned_user_change"
	ActivityAssignedTeamChange = "assigned_team_change"
	ActivitySelfAssign         = "self_assign"
	ActivityTagChange          = "tag_change"
	ActivitySLASet             = "sla_set"

	ContentTypeText = "text"
	ContentTypeHTML = "html"

	maxLastMessageLen  = 45
	maxMessagesPerPage = 30
)

// Run starts a pool of worker goroutines to handle message dispatching via inbox's channel and processes incoming messages. It scans for
// pending outgoing messages at the specified read interval and pushes them to the outgoing queue.
func (m *Manager) Run(ctx context.Context, dispatchConcurrency int, scanInterval time.Duration) {
	dbScanner := time.NewTicker(scanInterval)
	defer dbScanner.Stop()

	// Spawn a worker goroutine pool to dispatch messages.
	for range dispatchConcurrency {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.MessageDispatchWorker(ctx)
		}()
	}

	// Spawn a goroutine to process incoming messages.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.IncomingMessageWorker(ctx)
	}()

	// Scan pending outgoing messages and send them.
	for {
		select {
		case <-ctx.Done():
			return
		case <-dbScanner.C:
			var (
				pendingMessages = []models.Message{}
				messageIDs      = m.getOutgoingProcessingMessageIDs()
			)

			// Get pending outgoing messages and skip the currently processing message ids.
			if err := m.q.GetPendingMessages.Select(&pendingMessages, pq.Array(messageIDs)); err != nil {
				m.lo.Error("error fetching pending messages from db", "error", err)
				continue
			}

			// Prepare and push the message to the outgoing queue.
			for _, message := range pendingMessages {
				// Put the message ID in the processing map.
				m.outgoingProcessingMessages.Store(message.ID, message.ID)

				// Push the message to the outgoing message queue.
				m.outgoingMessageQueue <- message
			}
		}
	}
}

// Close signals the Manager to stop processing messages, closes channels,
// and waits for all worker goroutines to finish processing.
func (m *Manager) Close() {
	m.closedMu.Lock()
	defer m.closedMu.Unlock()
	m.closed = true
	close(m.outgoingMessageQueue)
	close(m.incomingMessageQueue)
	m.wg.Wait()
}

// IncomingMessageWorker processes incoming messages from the incoming message queue.
func (m *Manager) IncomingMessageWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-m.incomingMessageQueue:
			if !ok {
				return
			}
			if err := m.processIncomingMessage(msg); err != nil {
				m.lo.Error("error processing incoming msg", "error", err)
			}
		}
	}
}

// MessageDispatchWorker dispatches outgoing pending messages.
func (m *Manager) MessageDispatchWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-m.outgoingMessageQueue:
			if !ok {
				return
			}
			m.processOutgoingMessage(message)
		}
	}
}

// processOutgoingMessage handles the processing of a single outgoing message
func (m *Manager) processOutgoingMessage(message models.Message) {
	defer m.outgoingProcessingMessages.Delete(message.ID)

	// Helper function to handle errors
	handleError := func(err error, errorMsg string) bool {
		if err != nil {
			m.lo.Error(errorMsg, "error", err, "id", message.ID)
			m.UpdateMessageStatus(message.UUID, MessageStatusFailed)
			return true
		}
		return false
	}

	// Get inbox
	inbox, err := m.inboxStore.Get(message.InboxID)
	if handleError(err, "error fetching inbox") {
		return
	}

	// Render content in template
	if err := m.RenderContentInTemplate(inbox.Channel(), &message); err != nil {
		handleError(err, "error rendering content in template")
		return
	}

	// Attach attachments to the message
	if err := m.attachAttachmentsToMessage(&message); err != nil {
		handleError(err, "error attaching attachments to message")
		return
	}

	// Set message properties
	message.From = inbox.FromAddress()
	message.To, _ = m.GetToAddress(message.ConversationID)
	message.InReplyTo, _ = m.GetLatestReceivedMessageSourceID(message.ConversationID)

	// Send message
	err = inbox.Send(message)
	if handleError(err, "error sending message") {
		return
	}

	// Update status and first reply time
	m.UpdateMessageStatus(message.UUID, MessageStatusSent)
	m.UpdateConversationFirstReplyAt(message.ConversationUUID, message.ConversationID, message.CreatedAt)
}

// RenderContentInTemplate renders message content in template.
func (m *Manager) RenderContentInTemplate(channel string, message *models.Message) error {
	switch channel {
	case inbox.ChannelEmail:
		conversation, err := m.GetConversation(0, message.ConversationUUID)
		if err != nil {
			m.lo.Error("error fetching conversation", "uuid", message.ConversationUUID, "error", err)
			return fmt.Errorf("fetching conversation: %w", err)
		}
		message.Content, err = m.template.RenderWithBaseTemplate(conversation, message.Content)
		if err != nil {
			m.lo.Error("could not render email content using template", "id", message.ID, "error", err)
			return fmt.Errorf("could not render email content using template: %w", err)
		}
	default:
		m.lo.Warn("unknown message channel", "channel", channel)
		return fmt.Errorf("unknown message channel: %s", channel)
	}
	return nil
}

// GetConversationMessages retrieves messages for a specific conversation.
func (m *Manager) GetConversationMessages(conversationUUID string, page, pageSize int) ([]models.Message, int, error) {
	var (
		messages = make([]models.Message, 0)
		qArgs    []interface{}
	)

	qArgs = append(qArgs, conversationUUID)
	query, pageSize, qArgs, err := m.generateMessagesQuery(m.q.GetMessages, qArgs, page, pageSize)
	if err != nil {
		m.lo.Error("error generating messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}

	tx, err := m.db.BeginTxx(context.Background(), nil)
	defer tx.Rollback()
	if err != nil {
		m.lo.Error("error preparing get messages query", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}

	if err := tx.Select(&messages, query, qArgs...); err != nil {
		m.lo.Error("error fetching conversations", "error", err)
		return messages, pageSize, envelope.NewError(envelope.GeneralError, "Error fetching messages", nil)
	}

	return messages, pageSize, nil
}

// GetMessage retrieves a message by UUID.
func (m *Manager) GetMessage(uuid string) (models.Message, error) {
	var message models.Message
	if err := m.q.GetMessage.Get(&message, uuid); err != nil {
		m.lo.Error("error fetching message", "uuid", uuid, "error", err)
		return message, envelope.NewError(envelope.GeneralError, "Error fetching message", nil)
	}
	return message, nil
}

// UpdateMessageStatus updates the status of a message.
func (m *Manager) UpdateMessageStatus(uuid string, status string) error {
	if _, err := m.q.UpdateMessageStatus.Exec(status, uuid); err != nil {
		m.lo.Error("error updating message status", "error", err, "uuid", uuid)
		return err
	}

	// Broadcast messge status update to all conversation subscribers.
	conversationUUID, _ := m.getConversationUUIDFromMessageUUID(uuid)
	m.BroadcastMessagePropUpdate(conversationUUID, uuid, "status" /*property*/, status)
	return nil
}

// MarkMessageAsPending updates message status to `Pending`, so if it's a outgoing message it can be picked up again by a worker.
func (m *Manager) MarkMessageAsPending(uuid string) error {
	if err := m.UpdateMessageStatus(uuid, MessageStatusPending); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error retrying message", nil)
	}
	return nil
}

// SendPrivateNote inserts a private message in a conversation.
func (m *Manager) SendPrivateNote(media []mmodels.Media, senderID int, conversationUUID, content string) error {
	// Insert Message.
	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             MessageOutgoing,
		SenderType:       SenderTypeUser,
		Status:           MessageStatusSent,
		Content:          content,
		ContentType:      ContentTypeHTML,
		Private:          true,
		Media:            media,
	}
	return m.InsertMessage(&message)
}

// SendReply inserts a reply message in a conversation.
func (m *Manager) SendReply(media []mmodels.Media, senderID int, conversationUUID, content, meta string) error {
	message := models.Message{
		ConversationUUID: conversationUUID,
		SenderID:         senderID,
		Type:             MessageOutgoing,
		SenderType:       SenderTypeUser,
		Status:           MessageStatusPending,
		Content:          content,
		ContentType:      ContentTypeHTML,
		Private:          false,
		Media:            media,
		Meta:             meta,
	}
	return m.InsertMessage(&message)
}

// InsertMessage inserts a message and attaches the media to the message.
func (m *Manager) InsertMessage(message *models.Message) error {
	// Private message is always sent.
	if message.Private {
		message.Status = MessageStatusSent
	}

	if message.Meta == "" {
		message.Meta = "{}"
	}

	// Insert Message.
	if err := m.q.InsertMessage.QueryRow(message.Type, message.Status, message.ConversationID, message.ConversationUUID, message.Content, message.SenderID, message.SenderType,
		message.Private, message.ContentType, message.SourceID, message.Meta).Scan(&message.ID, &message.UUID, &message.CreatedAt); err != nil {
		m.lo.Error("error inserting message in db", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error sending message", nil)
	}

	// Attach message to the media.
	for _, media := range message.Media {
		m.mediaStore.Attach(media.ID, mmodels.ModelMessages, message.ID)
	}

	// Add this user as a participant.
	if err := m.AddConversationParticipant(message.SenderID, message.ConversationUUID); err != nil {
		return envelope.NewError(envelope.GeneralError, "Error sending message", nil)
	}

	// Update conversation last message details.
	lastMessage := stringutil.HTML2Text(message.Content)
	m.UpdateConversationLastMessage(message.ConversationID, message.ConversationUUID, lastMessage, message.CreatedAt)

	// Broadcast new message to all conversation subscribers.
	m.BroadcastNewConversationMessage(message.ConversationUUID, lastMessage, message.UUID, message.CreatedAt.Format(time.RFC3339), message.Type, message.Private)
	return nil
}

// RecordAssigneeUserChange records an activity for a user assignee change.
func (m *Manager) RecordAssigneeUserChange(conversationUUID string, assigneeID int, actor umodels.User) error {
	// Self assign.
	if assigneeID == actor.ID {
		return m.InsertConversationActivity(ActivitySelfAssign, conversationUUID, actor.FullName(), actor)
	}

	// Assignment to another user.
	assignee, err := m.userStore.Get(assigneeID)
	if err != nil {
		return err
	}
	return m.InsertConversationActivity(ActivityAssignedUserChange, conversationUUID, assignee.FullName(), actor)
}

// RecordAssigneeTeamChange records an activity for a team assignee change.
func (m *Manager) RecordAssigneeTeamChange(conversationUUID string, teamID int, actor umodels.User) error {
	team, err := m.teamStore.Get(teamID)
	if err != nil {
		return err
	}
	return m.InsertConversationActivity(ActivityAssignedTeamChange, conversationUUID, team.Name, actor)
}

// RecordPriorityChange records an activity for a priority change.
func (m *Manager) RecordPriorityChange(priority, conversationUUID string, actor umodels.User) error {
	return m.InsertConversationActivity(ActivityPriorityChange, conversationUUID, priority, actor)
}

// RecordStatusChange records an activity for a status change.
func (m *Manager) RecordStatusChange(status, conversationUUID string, actor umodels.User) error {
	return m.InsertConversationActivity(ActivityStatusChange, conversationUUID, status, actor)
}

// RecordSLASet records an activity for an SLA set.
func (m *Manager) RecordSLASet(conversationUUID string, actor umodels.User) error {
	return m.InsertConversationActivity(ActivitySLASet, conversationUUID, "", actor)
}

// InsertConversationActivity inserts an activity message.
func (m *Manager) InsertConversationActivity(activityType, conversationUUID, newValue string, actor umodels.User) error {
	content, err := m.getMessageActivityContent(activityType, newValue, actor.FullName())
	if err != nil {
		m.lo.Error("error could not generate activity content", "error", err)
		return err
	}

	message := models.Message{
		Type:             MessageActivity,
		Status:           MessageStatusSent,
		Content:          content,
		ContentType:      ContentTypeText,
		ConversationUUID: conversationUUID,
		Private:          true,
		SenderID:         actor.ID,
		SenderType:       SenderTypeUser,
	}

	// InsertMessage message in DB.
	m.InsertMessage(&message)
	return nil
}

// getConversationUUIDFromMessageUUID returns conversation UUID from message UUID.
func (m *Manager) getConversationUUIDFromMessageUUID(uuid string) (string, error) {
	var conversationUUID string
	if err := m.q.GetConversationUUIDFromMessageUUID.Get(&conversationUUID, uuid); err != nil {
		m.lo.Error("error fetching conversation uuid from message uuid", "uuid", uuid, "error", err)
		return conversationUUID, err
	}
	return conversationUUID, nil
}

// getMessageActivityContent generates activity content based on the activity type.
func (m *Manager) getMessageActivityContent(activityType, newValue, actorName string) (string, error) {
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
	case ActivitySLASet:
		content = fmt.Sprintf("%s set an SLA to this conversation", actorName)
	default:
		return "", fmt.Errorf("invalid activity type %s", activityType)
	}
	return content, nil
}

// processIncomingMessage handles the insertion of an incoming message and
// associated contact. It finds or creates the contact, checks for existing
// conversations, and creates a new conversation if necessary. It also
// inserts the message, uploads any attachments, and queues the conversation evaluation of automation rules.
func (m *Manager) processIncomingMessage(in models.IncomingMessage) error {
	var err error

	// Find or create contact and set sender ID.
	if err = m.userStore.CreateContact(&in.Contact); err != nil {
		m.lo.Error("error upserting contact", "error", err)
		return err
	}
	in.Message.SenderID = in.Contact.ID

	// This message already exists?
	conversationID, err := m.findConversationID([]string{in.Message.SourceID.String})
	if err != nil && err != ErrConversationNotFound {
		return err
	}
	if conversationID > 0 {
		return nil
	}

	// Find or create new conversation.
	isNewConversation, err := m.findOrCreateConversation(&in.Message, in.InboxID, in.Contact.ContactChannelID, in.Contact.ID)
	if err != nil {
		return err
	}

	// Insert message.
	if err = m.InsertMessage(&in.Message); err != nil {
		return err
	}

	// Upload attachments.
	if err := m.uploadMessageAttachments(&in.Message); err != nil {
		return err
	}

	// Evaluate automation rules for this conversation.
	if isNewConversation {
		m.automation.EvaluateNewConversationRules(in.Message.ConversationUUID)
	} else {
		m.automation.EvaluateConversationUpdateRules(in.Message.ConversationUUID, amodels.EventConversationMessageIncoming)
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

// EnqueueIncoming enqueues an incoming message for inserting in db.
func (m *Manager) EnqueueIncoming(message models.IncomingMessage) error {
	m.closedMu.Lock()
	defer m.closedMu.Unlock()
	if m.closed {
		return errors.New("incoming message queue is closed")
	}

	select {
	case m.incomingMessageQueue <- message:
		return nil
	default:
		m.lo.Warn("WARNING: incoming message queue is full")
		return errors.New("incoming message queue is full")
	}
}

// GetConversationByMessageID returns conversation by message id.
func (m *Manager) GetConversationByMessageID(id int) (models.Conversation, error) {
	var conversation = models.Conversation{}
	if err := m.q.GetConversationByMessageID.Get(&conversation, id); err != nil {
		if err == sql.ErrNoRows {
			return conversation, ErrConversationNotFound
		}
		m.lo.Error("error fetching message from DB", "error", err)
		return conversation, envelope.NewError(envelope.GeneralError, "Error fetching message", nil)
	}
	return conversation, nil
}

// generateMessagesQuery generates the SQL query for fetching messages in a conversation.
func (c *Manager) generateMessagesQuery(baseQuery string, qArgs []interface{}, page, pageSize int) (string, int, []interface{}, error) {
	if page <= 0 {
		return "", 0, nil, errors.New("page must be greater than 0")
	}
	if pageSize > maxMessagesPerPage {
		pageSize = maxMessagesPerPage
	}
	if pageSize <= 0 {
		return "", 0, nil, errors.New("page size must be greater than 0")
	}

	// Calculate the offset
	offset := (page - 1) * pageSize

	// Append LIMIT and OFFSET to query arguments
	qArgs = append(qArgs, pageSize, offset)

	// Include LIMIT and OFFSET in the SQL query
	sqlQuery := fmt.Sprintf(baseQuery, fmt.Sprintf("LIMIT $%d OFFSET $%d", len(qArgs)-1, len(qArgs)))
	return sqlQuery, pageSize, qArgs, nil
}

// uploadMessageAttachments uploads attachments for a message.
func (m *Manager) uploadMessageAttachments(message *models.Message) error {
	if len(message.Attachments) == 0 {
		return nil
	}

	var uploadErr error
	for _, attachment := range message.Attachments {
		// Check if this attachment already exists by content ID.
		exists, err := m.mediaStore.ContentIDExists(attachment.ContentID)
		if err != nil {
			m.lo.Error("error checking media existence", "error", err)
			continue
		}

		if exists {
			m.lo.Debug("attachment already exists", "content_id", attachment.ContentID)
			continue
		}

		m.lo.Debug("uploading message attachment", "name", attachment.Name)
		attachment.Name = stringutil.SanitizeFilename(attachment.Name)

		reader := bytes.NewReader(attachment.Content)
		_, err = m.mediaStore.UploadAndInsert(
			attachment.Name,
			attachment.ContentType,
			attachment.ContentID,
			mmodels.ModelMessages,
			message.ID,
			reader,
			attachment.Size,
			attachment.Disposition,
			[]byte("{}"),
		)

		if err != nil {
			uploadErr = err
			m.lo.Error("failed to upload attachment", "name", attachment.Name, "error", err)
		}
	}
	return uploadErr
}

// findOrCreateConversation finds or creates a conversation for the given message.
func (m *Manager) findOrCreateConversation(in *models.Message, inboxID, contactChannelID, contactID int) (bool, error) {
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
		lastMessage := stringutil.HTML2Text(in.Content)
		lastMessageAt := time.Now()
		conversationID, conversationUUID, err = m.CreateConversation(contactID, contactChannelID, inboxID, lastMessage, lastMessageAt, in.Subject)
		if err != nil || conversationID == 0 {
			return new, err
		}
		in.ConversationID = conversationID
		in.ConversationUUID = conversationUUID
		return new, nil
	}
	// Get UUID.
	if conversationUUID == "" {
		conversationUUID, err = m.GetConversationUUID(conversationID)
		if err != nil {
			return new, err
		}
	}
	in.ConversationID = conversationID
	in.ConversationUUID = conversationUUID
	return new, nil
}

// findConversationID finds the conversation ID from the message source ID.
func (m *Manager) findConversationID(messageSourceIDs []string) (int, error) {
	if len(messageSourceIDs) == 0 {
		return 0, ErrConversationNotFound
	}
	var conversationID int
	if err := m.q.MessageExistsBySourceID.QueryRow(pq.Array(messageSourceIDs)).Scan(&conversationID); err != nil {
		if err == sql.ErrNoRows {
			return conversationID, ErrConversationNotFound
		}
		m.lo.Error("error fetching msg from DB", "error", err)
		return conversationID, err
	}
	return conversationID, nil
}

// attachAttachmentsToMessage attaches attachment blobs to message.
func (m *Manager) attachAttachmentsToMessage(message *models.Message) error {
	var attachments attachment.Attachments

	// Get all media for this message.
	medias, err := m.mediaStore.GetByModel(message.ID, mmodels.ModelMessages)
	if err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return err
	}

	// Fetch blobs.
	for _, media := range medias {
		blob, err := m.mediaStore.GetBlob(media.UUID)
		if err != nil {
			m.lo.Error("error fetching media blob", "error", err)
			return err
		}
		attachment := attachment.Attachment{
			Name:    media.Filename,
			Content: blob,
			Header:  attachment.MakeHeader(media.ContentType, media.UUID, media.Filename, "base64", media.Disposition),
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
