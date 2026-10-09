package conversation

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/google/uuid"
)

// AIAnalysis is the only integration-owned message metadata accepted here.
type AIAnalysis struct {
	Kind                string `json:"kind"`
	Version             int    `json:"version"`
	OriginalMessageUUID string `json:"original_message_uuid"`
	SuggestedMessage    string `json:"suggested_message"`
}

var htmlTag = regexp.MustCompile(`(?is)<\s*(?:/?[a-z][a-z0-9:-]*(?:\s[^<>]*)?\s*/?|!--.*?--|!doctype[^<>]*)>`)

func (a AIAnalysis) Validate() error {
	invalid := errors.New("invalid ai_analysis: expected kind n8n_ai_analysis, version 1, canonical UUID and nonempty plaintext suggestion up to 65536 UTF-16 units")
	id, err := uuid.Parse(a.OriginalMessageUUID)
	if err != nil || id == uuid.Nil || !strings.EqualFold(id.String(), a.OriginalMessageUUID) || a.Kind != "n8n_ai_analysis" || a.Version != 1 || !utf8.ValidString(a.SuggestedMessage) || strings.TrimSpace(a.SuggestedMessage) == "" || strings.ContainsRune(a.SuggestedMessage, 0) || htmlTag.MatchString(a.SuggestedMessage) {
		return invalid
	}
	// JavaScript string.length uses UTF-16 units: enforce the actual frontend limit.
	if len(utf16.Encode([]rune(a.SuggestedMessage))) > 65536 {
		return invalid
	}
	return nil
}

func ValidateAIAnalysisOriginal(a AIAnalysis, original models.Message, conversationUUID string) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if !strings.EqualFold(original.UUID, a.OriginalMessageUUID) || original.ConversationUUID != conversationUUID || original.Type != models.MessageIncoming || original.SenderType != models.SenderTypeContact || original.Private {
		return errors.New("ai_analysis original must be a public incoming contact message in this conversation")
	}
	return nil
}

// SendAIAnalysisNote reuses the private-note insertion path; no outgoing transport.
func (m *Manager) SendAIAnalysisNote(senderID int, conversationUUID, content string, analysis AIAnalysis) (models.Message, error) {
	if err := analysis.Validate(); err != nil {
		return models.Message{}, envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	var original models.Message
	if err := m.q.GetMessage.Get(&original, analysis.OriginalMessageUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Message{}, envelope.NewError(envelope.NotFoundError, "Original customer message not found", nil)
		}
		m.lo.Error("error fetching AI analysis original", "error", err)
		return models.Message{}, envelope.NewError(envelope.GeneralError, "Could not fetch original customer message", nil)
	}
	if err := ValidateAIAnalysisOriginal(analysis, original, conversationUUID); err != nil {
		return models.Message{}, envelope.NewError(envelope.InputError, err.Error(), nil)
	}
	// Normalize the reference to the DB representation used by the frontend.
	analysis.OriginalMessageUUID = original.UUID
	meta, err := json.Marshal(map[string]any{"ai_analysis": analysis})
	if err != nil {
		return models.Message{}, err
	}
	return m.sendPrivateNote(nil, senderID, conversationUUID, content, nil, meta, false)
}
