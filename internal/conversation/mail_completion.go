package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var ErrReplySource = errors.New("invalid or conflicting reply source")
var ErrCompletionConflict = errors.New("new customer input; completion must not resolve this conversation")
var ErrCompletionNotSent = errors.New("completion requires a sent employee email with a source and transport snapshot")

// Strings preserve BIGINT versions across JavaScript and JSON clients.
type MailCompletion struct {
	SnapshotReady   bool    `json:"snapshot_ready,omitempty"`
	SourceUUID      *string `json:"source_message_uuid"`
	ResponseSource  *string `json:"response_source"`
	AgentID         *string `json:"agent_id"`
	ExpectedVersion string  `json:"expected_incoming_version"`
	QueuedVersion   string  `json:"queued_incoming_version"`
	SentMessage     string  `json:"sent_message"`
	SentAt          string  `json:"sent_at,omitempty"`
	Committed       bool    `json:"committed,omitempty"`
	AttemptAt       string  `json:"attempt_at,omitempty"`
	ResolveResult   string  `json:"resolve_result,omitempty"`
}
type ReplySource struct {
	UUID      string    `db:"uuid" json:"uuid"`
	Preview   string    `db:"preview" json:"preview"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Version   string    `db:"version" json:"incoming_version"`
}

func (m *Manager) ReplySources(cuuid string, limit, offset int) ([]ReplySource, int, error) {
	rows := []ReplySource{}
	var total int
	where := ` FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id JOIN inboxes i ON i.id=c.inbox_id WHERE c.uuid=$1 AND i.channel='email' AND m.type='incoming' AND m.sender_type='contact' AND NOT m.private AND NOT(m.meta ? 'deleted_at')`
	if err := m.db.Get(&total, `SELECT count(*)`+where, cuuid); err != nil {
		return nil, 0, err
	}
	err := m.db.Select(&rows, `SELECT m.uuid,LEFT(m.text_content,160) AS preview,m.created_at,COALESCE(m.meta->>'mail_input_version','') AS version`+where+` ORDER BY m.id DESC LIMIT $2 OFFSET $3`, cuuid, limit, offset)
	return rows, total, err
}
func validSourceTx(ctx context.Context, db sqlx.QueryerContext, cuuid, source string) (string, error) {
	if id, err := uuid.Parse(source); err != nil || id == uuid.Nil {
		return "", ErrReplySource
	}
	var version string
	err := sqlx.GetContext(ctx, db, &version, `SELECT m.meta->>'mail_input_version' FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id JOIN inboxes i ON i.id=c.inbox_id WHERE c.uuid=$1 AND i.channel='email' AND m.uuid=$2 AND m.type='incoming' AND m.sender_type='contact' AND NOT m.private AND NOT(m.meta ? 'deleted_at')`, cuuid, source)
	if err == sql.ErrNoRows {
		return "", ErrReplySource
	}
	if err != nil {
		return "", err
	}
	if n, e := strconv.ParseInt(version, 10, 64); e != nil || n <= 0 {
		return "", ErrReplySource
	}
	return version, nil
}

// prepareMailCompletion runs before InsertMessage. The immutable source version
// is the resolve baseline, never a guessed most-recent message.
func (m *Manager) prepareMailCompletion(cuuid string, sender int, meta map[string]any) (*MailCompletion, error) {
	var employee bool
	if err := m.db.Get(&employee, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND type='agent' AND email IS DISTINCT FROM 'System')`, sender); err != nil {
		return nil, err
	}
	if !employee || meta["is_automated"] == true || meta["continuity_email"] == true || meta["is_csat"] == true {
		return nil, nil
	}
	source, _ := meta["reply_to_message_uuid"].(string)
	var explicit struct {
		Source string `json:"response_source"`
		UUID   string `json:"source_message_uuid"`
	}
	raw, _ := json.Marshal(meta["ai_response"])
	_ = json.Unmarshal(raw, &explicit)
	feedback, _ := meta["ai_feedback_original_message_uuid"].(string)
	for _, candidate := range []string{explicit.UUID, feedback} {
		if candidate != "" {
			if source != "" && source != candidate {
				return nil, ErrReplySource
			}
			source = candidate
		}
	}
	state := &MailCompletion{AgentID: ptr(strconv.Itoa(sender))}
	if explicit.Source != "" {
		switch explicit.Source {
		case "ai", "ai_adapted", "human":
			state.ResponseSource = ptr(explicit.Source)
		default:
			return nil, ErrReplySource
		}
	} else if feedback != "" {
		state.ResponseSource = ptr("human")
	}
	var version int64
	if err := m.db.Get(&version, `SELECT COALESCE((SELECT v.version FROM conversation_mail_versions v JOIN conversations c ON c.id=v.conversation_id WHERE c.uuid=$1),0)`, cuuid); err != nil {
		return nil, err
	}
	state.QueuedVersion = strconv.FormatInt(version, 10)
	if source == "" {
		candidates, total, err := m.ReplySources(cuuid, 2, 0)
		if err != nil {
			return nil, err
		}
		if total == 1 && len(candidates) == 1 {
			source = candidates[0].UUID
		}
	}
	if source != "" {
		v, err := validSourceTx(context.Background(), m.db, cuuid, source)
		if err != nil {
			return nil, err
		}
		state.SourceUUID = ptr(source)
		state.ExpectedVersion = v
		chosen, _ := strconv.ParseInt(v, 10, 64)
		if chosen > version {
			state.QueuedVersion = v
		}
		meta["reply_to_message_uuid"] = source
	}
	return state, nil
}
func ptr(s string) *string { return &s }

// AssignReplySource repairs an unassigned legacy/ambiguous send. An existing
// association is immutable; a future input cannot be attached to an earlier send.
func (m *Manager) AssignReplySource(ctx context.Context, cuuid, outgoing, source string) error {
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw []byte
	err = tx.GetContext(ctx, &raw, `SELECT m.meta FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id JOIN inboxes i ON i.id=c.inbox_id WHERE c.uuid=$1 AND m.uuid=$2 AND i.channel='email' AND m.type='outgoing' AND m.sender_type='agent' AND NOT m.private AND m.meta ? 'mail_completion' FOR UPDATE OF m`, cuuid, outgoing)
	if err == sql.ErrNoRows {
		return ErrReplySource
	}
	if err != nil {
		return err
	}
	var meta map[string]json.RawMessage
	if err = json.Unmarshal(raw, &meta); err != nil {
		return err
	}
	var state MailCompletion
	if err = json.Unmarshal(meta["mail_completion"], &state); err != nil {
		return err
	}
	if state.SourceUUID != nil {
		if *state.SourceUUID == source {
			return nil
		}
		return ErrReplySource
	}
	v, err := validSourceTx(ctx, tx, cuuid, source)
	if err != nil {
		return err
	}
	n, _ := strconv.ParseInt(v, 10, 64)
	max, _ := strconv.ParseInt(state.QueuedVersion, 10, 64)
	if n > max {
		return ErrReplySource
	}
	state.SourceUUID = ptr(source)
	state.ExpectedVersion = v
	b, _ := json.Marshal(state)
	patch, _ := json.Marshal(map[string]any{"reply_to_message_uuid": source, "mail_completion": json.RawMessage(b)})
	_, err = tx.ExecContext(ctx, `UPDATE conversation_messages SET meta=meta||$1::jsonb WHERE uuid=$2`, string(patch), outgoing)
	if err != nil {
		return err
	}
	return tx.Commit()
}

type ResolveCompletionResult struct {
	PreviousStatus string `json:"-"`
	Changed        bool   `json:"-"`
	Outcome        string `json:"outcome"`
	Replayed       bool   `json:"replayed"`
}

func (m *Manager) ResolveMailCompletion(ctx context.Context, cuuid, outgoing, expected string) (ResolveCompletionResult, error) {
	result := ResolveCompletionResult{}
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var cid int
	var current int64
	// Same lock order as the input trigger: clock first, then conversation/message.
	err = tx.QueryRowxContext(ctx, `SELECT c.id,v.version FROM conversations c JOIN inboxes i ON i.id=c.inbox_id JOIN conversation_mail_versions v ON v.conversation_id=c.id WHERE c.uuid=$1 AND i.channel='email' FOR UPDATE OF v`, cuuid).Scan(&cid, &current)
	if err == sql.ErrNoRows {
		return result, ErrCompletionNotSent
	}
	if err != nil {
		return result, err
	}
	var raw []byte
	err = tx.GetContext(ctx, &raw, `SELECT meta FROM conversation_messages WHERE conversation_id=$1 AND uuid=$2 AND type='outgoing' AND sender_type='agent' AND NOT private AND status='sent' AND meta ? 'mail_completion' FOR UPDATE`, cid, outgoing)
	if err == sql.ErrNoRows {
		return result, ErrCompletionNotSent
	}
	if err != nil {
		return result, err
	}
	var meta map[string]json.RawMessage
	json.Unmarshal(raw, &meta)
	var state MailCompletion
	if json.Unmarshal(meta["mail_completion"], &state) != nil || state.SourceUUID == nil || state.ExpectedVersion == "" || state.SentAt == "" || !state.SnapshotReady {
		return result, ErrCompletionNotSent
	}
	if expected != state.ExpectedVersion {
		return result, ErrReplySource
	}
	if state.ResolveResult != "" {
		result.Outcome = state.ResolveResult
		result.Replayed = true
		if result.Outcome == "superseded" {
			return result, ErrCompletionConflict
		}
		return result, nil
	}
	result.Outcome = "resolved"
	if strconv.FormatInt(current, 10) != expected {
		result.Outcome = "superseded"
	} else {
		var previous string
		if err = tx.GetContext(ctx, &previous, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE c.id=$1 FOR UPDATE OF c`, cid); err != nil {
			return result, err
		}
		result.PreviousStatus = previous
		result.Changed = previous != "Resolved"
		_, err = tx.ExecContext(ctx, `UPDATE conversations SET status_id=(SELECT id FROM conversation_statuses WHERE name='Resolved'),resolved_at=COALESCE(resolved_at,NOW()),snoozed_until=NULL,updated_at=NOW() WHERE id=$1`, cid)
		if err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE conversation_mail_versions SET resolved_version=version WHERE conversation_id=$1`, cid); err != nil {
			return result, err
		}
	}
	state.ResolveResult = result.Outcome
	patch, _ := json.Marshal(state)
	if _, err = tx.ExecContext(ctx, `UPDATE conversation_messages SET meta=jsonb_set(meta,'{mail_completion}',$1::jsonb) WHERE uuid=$2`, string(patch), outgoing); err != nil {
		return result, err
	}
	if err = tx.Commit(); err != nil {
		return ResolveCompletionResult{}, err
	}
	if result.Outcome == "superseded" {
		return result, ErrCompletionConflict
	}
	return result, nil
}
