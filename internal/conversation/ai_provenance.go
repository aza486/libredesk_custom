package conversation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/abhinavxd/libredesk/internal/aiassist"
)

// AIResponseUsed is scoped to a source, never inferred from body or timestamps.
func (m *Manager) AIResponseUsed(conversation, source string) (bool, error) {
	var used bool
	err := m.db.Get(&used, `SELECT EXISTS(SELECT 1 FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE c.uuid=$1 AND m.type='outgoing' AND NOT m.private AND m.status IN ('pending','sent') AND (m.meta->'ai_response'->>'source_message_uuid'=$2 OR m.meta->>'ai_feedback_original_message_uuid'=$2))`, conversation, source)
	return used, err
}

// RunAIProvenance is an independent durable delivery loop. It never invokes email
// sending or feedback. Receiver idempotency is mandatory across lost responses.
func (m *Manager) RunAIProvenance(ctx context.Context, client *aiassist.Client) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		m.deliverAIProvenance(ctx, client)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type provenanceCaller interface {
	Call(context.Context, string, string, any, any) (int, error)
}

func (m *Manager) deliverAIProvenance(ctx context.Context, client provenanceCaller) {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	var locked bool
	if tx.GetContext(ctx, &locked, `SELECT pg_try_advisory_xact_lock(719438251)`) != nil || !locked {
		return
	}
	var rows []struct {
		UUID         string          `db:"uuid"`
		Conversation string          `db:"conversation_uuid"`
		Meta         json.RawMessage `db:"meta"`
	}
	err = tx.SelectContext(ctx, &rows, `SELECT m.uuid,c.uuid AS conversation_uuid,m.meta FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE m.status='sent' AND m.type='outgoing' AND NOT m.private AND m.sender_type='agent' AND m.meta ? 'ai_response' AND m.meta ? 'ai_response_sent_at' AND m.meta ? 'ai_response_sent_body' AND NOT (m.meta ? 'ai_response_committed') ORDER BY COALESCE(m.meta->>'ai_response_attempt_at',''),m.id LIMIT 10`)
	if err != nil {
		return
	}
	for _, row := range rows {
		var meta map[string]json.RawMessage
		if json.Unmarshal(row.Meta, &meta) != nil {
			continue
		}
		var response map[string]string
		if json.Unmarshal(meta["ai_response"], &response) != nil {
			continue
		}
		var body, sent string
		json.Unmarshal(meta["ai_response_sent_body"], &body)
		json.Unmarshal(meta["ai_response_sent_at"], &sent)
		payload := map[string]string{"event_type": "outgoing_response_sent", "conversation_uuid": row.Conversation, "source_message_uuid": response["source_message_uuid"], "outgoing_message_uuid": row.UUID, "response_source": response["response_source"], "sent_message": body, "sent_at": sent}
		var ack struct {
			Committed bool   `json:"committed"`
			Outgoing  string `json:"outgoing_message_uuid"`
		}
		requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, callErr := client.Call(requestCtx, "X-Libredesk-Provenance-Token", row.UUID, payload, &ack)
		cancel()
		patch := map[string]any{"ai_response_attempt_at": time.Now().UTC().Format(time.RFC3339Nano)}
		if callErr == nil && ack.Committed && ack.Outgoing == row.UUID {
			patch["ai_response_committed"] = true
		}
		raw, _ := json.Marshal(patch)
		if _, err = tx.ExecContext(ctx, `UPDATE conversation_messages SET meta=meta||$1::jsonb WHERE uuid=$2`, string(raw), row.UUID); err != nil {
			return
		}
	}
	tx.Commit()
}
