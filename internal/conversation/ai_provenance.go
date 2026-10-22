package conversation

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"github.com/abhinavxd/libredesk/internal/aiassist"
	"strconv"
	"time"
)

func (m *Manager) AIResponseUsed(conversation, source string) (bool, error) {
	var used bool
	err := m.db.Get(&used, `SELECT EXISTS(SELECT 1 FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE c.uuid=$1 AND m.type='outgoing' AND NOT m.private AND m.status IN('pending','sent') AND (m.meta->'ai_response'->>'source_message_uuid'=$2 OR m.meta->>'ai_feedback_original_message_uuid'=$2))`, conversation, source)
	return used, err
}
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
type MailCompletionEvent struct {
	EventType               string  `json:"event_type"`
	ConversationUUID        string  `json:"conversation_uuid"`
	SourceMessageUUID       string  `json:"source_message_uuid"`
	OutgoingMessageUUID     string  `json:"outgoing_message_uuid"`
	ResponseSource          *string `json:"response_source"`
	AgentID                 *string `json:"agent_id"`
	SentMessage             string  `json:"sent_message"`
	SentAt                  string  `json:"sent_at"`
	ExpectedIncomingVersion string  `json:"expected_incoming_version"`
}

func (m *Manager) deliverAIProvenance(ctx context.Context, client provenanceCaller) {
	// Session-level lock: never hold a row/transaction lock over the HTTP call.
	// n8n may synchronously call the conditional resolve endpoint before responding.
	conn, err := m.db.Connx(ctx)
	if err != nil {
		return
	}
	defer conn.Close()
	var locked bool
	if conn.GetContext(ctx, &locked, `SELECT pg_try_advisory_lock(719438251)`) != nil || !locked {
		return
	}
	defer func() {
		release, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, e := conn.ExecContext(release, `SELECT pg_advisory_unlock(719438251)`); e != nil {
			conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	var rows []struct {
		UUID         string          `db:"uuid"`
		Conversation string          `db:"conversation_uuid"`
		Sender       int             `db:"sender_id"`
		Meta         json.RawMessage `db:"meta"`
	}
	err = conn.SelectContext(ctx, &rows, `SELECT m.uuid,c.uuid AS conversation_uuid,m.sender_id,m.meta FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id JOIN inboxes i ON i.id=c.inbox_id JOIN users u ON u.id=m.sender_id
 WHERE i.channel='email' AND m.status='sent' AND m.type='outgoing' AND NOT m.private AND m.sender_type='agent' AND u.type='agent' AND u.email IS DISTINCT FROM 'System'
 AND COALESCE(m.meta->>'is_automated','false')!='true' AND COALESCE(m.meta->>'continuity_email','false')!='true' AND COALESCE(m.meta->>'is_csat','false')!='true'
 AND ((m.meta->'mail_completion'->>'source_message_uuid' IS NOT NULL AND m.meta->'mail_completion'->>'snapshot_ready'='true' AND m.meta->'mail_completion' ? 'sent_at' AND NOT(m.meta->'mail_completion' ? 'committed'))
 OR (NOT(m.meta ? 'mail_completion') AND m.meta ? 'ai_response' AND m.meta ? 'ai_response_sent_body' AND m.meta ? 'ai_response_sent_at' AND NOT(m.meta ? 'ai_response_committed')))
 ORDER BY COALESCE(m.meta->'mail_completion'->>'attempt_at',m.meta->>'ai_response_attempt_at',''),m.id LIMIT 10`)
	if err != nil {
		return
	}
	for _, row := range rows {
		var meta map[string]json.RawMessage
		if json.Unmarshal(row.Meta, &meta) != nil {
			continue
		}
		var state MailCompletion
		if raw, ok := meta["mail_completion"]; ok {
			if json.Unmarshal(raw, &state) != nil {
				continue
			}
		} else {
			// Existing explicit events retain their source, snapshot and idempotency key.
			var explicit struct {
				Source string `json:"response_source"`
				UUID   string `json:"source_message_uuid"`
			}
			if json.Unmarshal(meta["ai_response"], &explicit) != nil {
				continue
			}
			v, err := validSourceTx(ctx, conn, row.Conversation, explicit.UUID)
			if err != nil {
				continue
			}
			state = MailCompletion{SourceUUID: ptr(explicit.UUID), ResponseSource: ptr(explicit.Source), AgentID: ptr(strconv.Itoa(row.Sender)), ExpectedVersion: v, QueuedVersion: v}
			state.SnapshotReady = true
			json.Unmarshal(meta["ai_response_sent_body"], &state.SentMessage)
			json.Unmarshal(meta["ai_response_sent_at"], &state.SentAt)
			raw, _ := json.Marshal(state)
			if _, err = conn.ExecContext(ctx, `UPDATE conversation_messages SET meta=jsonb_set(meta,'{mail_completion}',$1::jsonb) WHERE uuid=$2 AND NOT(meta ? 'mail_completion')`, string(raw), row.UUID); err != nil {
				continue
			}
		}
		if state.SourceUUID == nil || state.ExpectedVersion == "" || state.SentAt == "" || !state.SnapshotReady {
			continue
		}
		payload := MailCompletionEvent{EventType: "outgoing_response_sent", ConversationUUID: row.Conversation, SourceMessageUUID: *state.SourceUUID, OutgoingMessageUUID: row.UUID, ResponseSource: state.ResponseSource, AgentID: state.AgentID, SentMessage: state.SentMessage, SentAt: state.SentAt, ExpectedIncomingVersion: state.ExpectedVersion}
		var ack struct {
			Committed bool   `json:"committed"`
			Outgoing  string `json:"outgoing_message_uuid"`
		}
		requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, callErr := client.Call(requestCtx, "X-Libredesk-Provenance-Token", row.UUID, payload, &ack)
		cancel()
		patch := map[string]any{"attempt_at": time.Now().UTC().Format(time.RFC3339Nano)}
		committed := callErr == nil && ack.Committed && ack.Outgoing == row.UUID
		if committed {
			patch["committed"] = true
		}
		raw, _ := json.Marshal(patch)
		// Merge only delivery fields: a concurrent resolve result must survive.
		_, err = conn.ExecContext(ctx, `UPDATE conversation_messages SET meta=jsonb_set(meta,'{mail_completion}',(meta->'mail_completion')||$1::jsonb) || CASE WHEN $3 AND meta ? 'ai_response' THEN '{"ai_response_committed":true}'::jsonb ELSE '{}'::jsonb END WHERE uuid=$2`, string(raw), row.UUID, committed)
		if err != nil {
			return
		}
	}
}
