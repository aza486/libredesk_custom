package conversation

import (
	"encoding/json"
	"slices"

	azmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
)

const SpamDetectionPurpose = "spam_detection"

// SendSpamDetectionNote is called only after API-key authentication, messages:write
// and conversation ACL checks. The explicit role grant is independent of the AI
// allowlist, system-tag grants and Admin. No template, mentions or delivery path.
func (m *Manager) SendSpamDetectionNote(actor umodels.User, uuid, content string) (models.Message, error) {
	if actor.ID <= 0 || !actor.Enabled || actor.Type != umodels.UserTypeAgent || !slices.Contains(actor.Permissions, azmodels.PermMessagesWriteSpamDetectionNote) {
		return models.Message{}, envelope.NewError(envelope.PermissionError, "Explicit spam detection note permission required", nil)
	}
	message := models.Message{
		ConversationUUID: uuid, SenderID: actor.ID, SenderType: models.SenderTypeAgent,
		Type: models.MessageOutgoing, Status: models.MessageStatusSent, Private: true,
		Content: content, ContentType: models.ContentTypeText,
		Meta: json.RawMessage(`{"security_note":{"purpose":"spam_detection"}}`),
	}
	if err := m.insertMessage(&message, true); err != nil {
		return models.Message{}, err
	}
	return message, nil
}

func lockSpamDetectionNote(tx *sqlx.Tx, message *models.Message) error {
	if !message.Private || message.SenderType != models.SenderTypeAgent || message.Type != models.MessageOutgoing || message.Status != models.MessageStatusSent || automaticAIMeta(message.Meta) {
		return envelope.NewError(envelope.PermissionError, "Invalid spam detection note", nil)
	}
	// Serialize with the existing Spam tag/status transitions until insertion commits.
	var id int
	if err := tx.Get(&id, `SELECT id FROM conversations WHERE uuid=$1 FOR UPDATE`, message.ConversationUUID); err != nil {
		return err
	}
	spam, err := conversationHasSpam(tx, message.ConversationUUID)
	if err != nil {
		return err
	}
	if !spam {
		return envelope.NewError(envelope.PermissionError, "Spam detection notes require an existing Spam tag", nil)
	}
	return nil
}
