package conversation

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/aifeedback"
	"github.com/jmoiron/sqlx"
)

func manualReviewNote(ctx context.Context, tx *sqlx.Tx, conversation, source string) (string, error) {
	var note string
	err := tx.GetContext(ctx, &note, `SELECT n.uuid FROM conversation_messages n
 JOIN conversations c ON c.id=n.conversation_id
 JOIN conversation_messages original ON original.uuid::text=$2 AND original.conversation_id=c.id
 WHERE c.uuid=$1 AND n.private AND n.type='outgoing' AND n.sender_type='agent'
 AND NOT(n.meta ? 'deleted_at') AND n.id>original.id
 AND n.meta->'ai_analysis'->>'kind'='n8n_manual_review'
 AND n.meta->'ai_analysis'->>'version'='1'
 AND n.meta->'ai_analysis'->>'original_message_uuid'=$2
 AND COALESCE(n.meta->'ai_analysis'->>'suggested_message','')=''
 AND NOT EXISTS(SELECT 1 FROM conversation_messages a WHERE a.conversation_id=c.id
   AND a.private AND a.type='outgoing' AND NOT(a.meta ? 'deleted_at')
   AND a.meta->'ai_analysis'->>'kind'='n8n_ai_analysis'
   AND a.meta->'ai_analysis'->>'version'='1'
   AND a.meta->'ai_analysis'->>'original_message_uuid'=$2)
 AND NOT EXISTS(SELECT 1 FROM conversation_messages r WHERE r.conversation_id=c.id
   AND r.type='outgoing' AND NOT r.private AND r.sender_type='agent' AND r.status='sent'
   AND r.id>original.id
   AND COALESCE(r.meta->>'is_automated','false')<>'true'
   AND COALESCE(r.meta->>'continuity_email','false')<>'true'
   AND COALESCE(r.meta->>'is_csat','false')<>'true'
   AND EXISTS(SELECT 1 FROM users u WHERE u.id=r.sender_id AND u.type='agent' AND u.email IS DISTINCT FROM 'System')
   AND (
    r.meta->'mail_completion'->>'source_message_uuid'=$2
    OR r.meta->>'ai_feedback_original_message_uuid'=$2
    OR r.meta->'ai_response'->>'source_message_uuid'=$2
    OR (r.meta->'mail_completion'->>'source_message_uuid' IS NULL
        AND r.meta->>'ai_feedback_original_message_uuid' IS NULL
        AND r.meta->'ai_response'->>'source_message_uuid' IS NULL)))
 ORDER BY n.id LIMIT 1`, conversation, source)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return note, err
}

type manualFeedbackSubmitter interface {
	SubmitManualResolution(context.Context, aifeedback.ManualResolutionSubmission) error
}

// RunManualResolutionFeedback retries only the durable feedback outbox, never SMTP.
func (m *Manager) RunManualResolutionFeedback(ctx context.Context, client *aifeedback.Client) {
	if client == nil {
		return
	}
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		m.deliverManualResolutionFeedback(ctx, client)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) deliverManualResolutionFeedback(ctx context.Context, client manualFeedbackSubmitter) {
	conn, err := m.db.Connx(ctx)
	if err != nil {
		return
	}
	defer conn.Close()
	var locked bool
	// Distinct from the provenance worker; neither endpoint shares its state.
	if conn.GetContext(ctx, &locked, `SELECT pg_try_advisory_lock(719438252)`) != nil || !locked {
		return
	}
	defer func() {
		release, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, e := conn.ExecContext(release, `SELECT pg_advisory_unlock(719438252)`); e != nil {
			conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	var rows []struct {
		Reason       *string   `db:"feedback_reason"`
		ID           string    `db:"id"`
		Original     string    `db:"original_message_uuid"`
		Outgoing     string    `db:"outgoing_message_uuid"`
		Conversation string    `db:"conversation_uuid"`
		Agent        int       `db:"agent_id"`
		Body         string    `db:"human_answer"`
		SentAt       time.Time `db:"sent_at"`
	}
	err = conn.SelectContext(ctx, &rows, `SELECT f.id,f.original_message_uuid,f.outgoing_message_uuid,
 c.uuid AS conversation_uuid,f.agent_id,f.human_answer,f.sent_at,f.feedback_reason
 FROM manual_resolution_feedback f JOIN conversations c ON c.id=f.conversation_id
 JOIN inboxes i ON i.id=c.inbox_id
 JOIN conversation_messages m ON m.uuid=f.outgoing_message_uuid AND m.conversation_id=c.id
 JOIN conversation_messages original ON original.uuid=f.original_message_uuid AND original.conversation_id=c.id
 WHERE f.committed_at IS NULL AND f.next_attempt_at<=NOW() AND i.channel='email'
 AND m.status='sent' AND m.type='outgoing' AND NOT m.private AND m.sender_type='agent' AND m.sender_id=f.agent_id
 AND NOT(m.meta ? 'deleted_at') AND NOT(m.meta ? 'ai_feedback_original_message_uuid') AND NOT(m.meta ? 'ai_response')
 AND original.type='incoming' AND original.sender_type='contact' AND NOT original.private AND NOT(original.meta ? 'deleted_at')
 ORDER BY f.next_attempt_at,f.id LIMIT 10`)
	if err != nil {
		m.lo.Error("error reading manual feedback outbox", "error", err)
		return
	}
	for _, row := range rows {
		payload := aifeedback.ManualResolutionSubmission{
			ID: row.ID, MailEventMessageUUID: row.Original, ConversationUUID: row.Conversation,
			OutgoingMessageUUID: row.Outgoing, AgentID: strconv.Itoa(row.Agent), HumanAnswer: row.Body,
			FeedbackReason: row.Reason, FeedbackType: "manual_resolution", ResponseSource: "human", SentAt: row.SentAt,
		}
		request, cancel := context.WithTimeout(ctx, 10*time.Second)
		callErr := client.SubmitManualResolution(request, payload)
		cancel()
		if callErr != nil {
			m.lo.Error("manual feedback commit not confirmed", "message_uuid", row.Outgoing, "error", callErr)
		}
		_, err = conn.ExecContext(ctx, `UPDATE manual_resolution_feedback SET
 attempt_count=attempt_count+1,attempt_at=NOW(),
 next_attempt_at=NOW()+INTERVAL '15 seconds'*LEAST(20,power(2,LEAST(attempt_count,5))),
 committed_at=CASE WHEN $2 THEN NOW() ELSE NULL END WHERE id=$1 AND committed_at IS NULL`, row.ID, callErr == nil)
		if err != nil {
			m.lo.Error("error saving manual feedback receipt", "message_uuid", row.Outgoing, "error", err)
			return
		}
	}
}
