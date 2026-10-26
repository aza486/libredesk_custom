package conversation

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrManualReviewNote = errors.New("manual review note not available in this conversation")
var ErrManualReasonLocked = errors.New("feedback reason can no longer be edited for this customer message")
var ErrManualReasonInput = errors.New("feedback reason must be valid text with at most 2500 characters")
var ErrManualReasonAgent = errors.New("enabled employee required")

type ManualReason struct {
	NoteUUID         string  `json:"note_uuid"`
	OriginalUUID     string  `json:"original_message_uuid"`
	ConversationUUID string  `json:"conversation_uuid"`
	AgentID          int     `json:"agent_id"`
	Reason           *string `json:"feedback_reason"`
	Editable         bool    `json:"editable"`
}

func NormalizeManualReason(reason string) (string, error) {
	if !utf8.ValidString(reason) || strings.ContainsRune(reason, 0) {
		return "", ErrManualReasonInput
	}
	reason = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(reason, "\r\n", "\n"), "\r", "\n"))
	if utf8.RuneCountInString(reason) > 2500 {
		return "", ErrManualReasonInput
	}
	return reason, nil
}

// ManualReviewFeedback only edits the caller's annotation. Duplicate typed notes
// for one incoming message share that caller's annotation, never another agent's.
// GET passes nil; PUT passes a value; DELETE passes an empty value.
func (m *Manager) ManualReviewFeedback(ctx context.Context, cuuid, noteUUID string, agent int, reason *string) (ManualReason, error) {
	result := ManualReason{NoteUUID: noteUUID, ConversationUUID: cuuid, AgentID: agent}
	for _, s := range []string{cuuid, noteUUID} {
		if id, e := uuid.Parse(s); e != nil || id == uuid.Nil || id.String() != strings.ToLower(s) {
			return result, ErrManualReviewNote
		}
	}
	normalized := ""
	var err error
	if reason != nil {
		normalized, err = NormalizeManualReason(*reason)
		if err != nil {
			return result, err
		}
	}
	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	var allowed bool
	if err = tx.GetContext(ctx, &allowed, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND type='agent' AND enabled AND deleted_at IS NULL AND email IS DISTINCT FROM 'System')`, agent); err != nil {
		return result, err
	}
	if !allowed {
		return result, ErrManualReasonAgent
	}
	var cid int
	var version string
	if err = tx.QueryRowxContext(ctx, `SELECT c.id,v.version::text FROM conversations c JOIN inboxes i ON i.id=c.inbox_id
 JOIN conversation_mail_versions v ON v.conversation_id=c.id WHERE c.uuid=$1 AND i.channel='email' FOR UPDATE OF v`, cuuid).Scan(&cid, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrManualReviewNote
		}
		return result, err
	}
	var originalVersion string
	err = tx.QueryRowxContext(ctx, `SELECT original.uuid,original.meta->>'mail_input_version'
 FROM conversation_messages n JOIN conversation_messages original ON original.conversation_id=n.conversation_id
 AND original.uuid::text=n.meta->'ai_analysis'->>'original_message_uuid'
 WHERE n.uuid=$1 AND n.conversation_id=$2 AND n.private AND n.type='outgoing' AND n.sender_type='agent'
 AND NOT(n.meta ? 'deleted_at') AND n.meta->'ai_analysis'->>'kind'='n8n_manual_review'
 AND n.meta->'ai_analysis'->>'version'='1' AND COALESCE(n.meta->'ai_analysis'->>'suggested_message','')=''
 AND original.type='incoming' AND original.sender_type='contact' AND NOT original.private
 AND NOT(original.meta ? 'deleted_at') AND n.id>original.id`, noteUUID, cid).Scan(&result.OriginalUUID, &originalVersion)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrManualReviewNote
		}
		return result, err
	}
	canonical, err := manualReviewNote(ctx, tx, cuuid, result.OriginalUUID)
	if err != nil {
		return result, err
	}
	var committed bool
	if err = tx.GetContext(ctx, &committed, `SELECT EXISTS(SELECT 1 FROM manual_resolution_feedback WHERE original_message_uuid=$1)`, result.OriginalUUID); err != nil {
		return result, err
	}
	result.Editable = !committed && version == originalVersion && canonical != ""
	if reason != nil {
		if !result.Editable {
			return result, ErrManualReasonLocked
		}
		if normalized == "" {
			_, err = tx.ExecContext(ctx, `DELETE FROM manual_review_annotations WHERE original_message_uuid=$1 AND agent_id=$2`, result.OriginalUUID, agent)
		} else {
			_, err = tx.ExecContext(ctx, `INSERT INTO manual_review_annotations(original_message_uuid,agent_id,conversation_id,note_uuid,feedback_reason)
 VALUES($1,$2,$3,$4,$5) ON CONFLICT(original_message_uuid,agent_id) DO UPDATE SET
 note_uuid=EXCLUDED.note_uuid,feedback_reason=EXCLUDED.feedback_reason,updated_at=NOW()`, result.OriginalUUID, agent, cid, noteUUID, normalized)
		}
		if err != nil {
			return result, err
		}
	}
	var stored string
	err = tx.GetContext(ctx, &stored, `SELECT feedback_reason FROM manual_review_annotations WHERE original_message_uuid=$1 AND agent_id=$2 AND conversation_id=$3`, result.OriginalUUID, agent, cid)
	if err == nil {
		result.Reason = &stored
	} else if !errors.Is(err, sql.ErrNoRows) {
		return result, err
	}
	return result, tx.Commit()
}
