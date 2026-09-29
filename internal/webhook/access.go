package webhook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/webhook/models"
)

// Preserve every original field as raw JSON, including integer precision.
func (m *Manager) withInboxAccess(event models.WebhookEvent, data any) (any, error) {
	if !strings.HasPrefix(string(event), "conversation.") && !strings.HasPrefix(string(event), "message.") {
		return data, nil
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	var ref struct {
		UUID             string `json:"uuid"`
		ConversationUUID string `json:"conversation_uuid"`
		Conversation     struct {
			UUID string `json:"uuid"`
		} `json:"conversation"`
	}
	if err := json.Unmarshal(raw, &ref); err != nil {
		return nil, err
	}
	uuid := ref.ConversationUUID
	if uuid == "" {
		uuid = ref.Conversation.UUID
	}
	if uuid == "" && strings.HasPrefix(string(event), "conversation.") {
		uuid = ref.UUID
	}
	var mode string
	var owner json.RawMessage
	// Conversation attributes retain the original owner after an inbox owner change.
	err = m.db.QueryRow(`SELECT i.access_mode, COALESCE(c.custom_attributes->'owner_user_id', to_jsonb(i.owner_user_id), 'null'::jsonb)
 FROM conversations c JOIN inboxes i ON i.id=c.inbox_id WHERE c.uuid=$1`, uuid).Scan(&mode, &owner)
	if err != nil {
		return nil, fmt.Errorf("loading webhook inbox access: %w", err)
	}
	payload["inbox_access_mode"], _ = json.Marshal(mode)
	payload["owner_user_id"] = owner
	return payload, nil
}
