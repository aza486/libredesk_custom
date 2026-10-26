package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_16_0 preserves the deployed v2.15 SENT eligibility trigger. A separate
// BEFORE INSERT trigger snapshots an annotation into that same single outbox row.
func V2_16_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
 ALTER TABLE manual_resolution_feedback ADD COLUMN IF NOT EXISTS feedback_reason TEXT
 CHECK(feedback_reason IS NULL OR char_length(feedback_reason)<=2500);
 CREATE TABLE IF NOT EXISTS manual_review_annotations (
 original_message_uuid UUID NOT NULL REFERENCES conversation_messages(uuid) ON DELETE CASCADE,
 agent_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
 note_uuid UUID NOT NULL REFERENCES conversation_messages(uuid) ON DELETE CASCADE,
 feedback_reason TEXT NOT NULL CHECK(char_length(feedback_reason) BETWEEN 1 AND 2500),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY(original_message_uuid,agent_id)
 );
 CREATE OR REPLACE FUNCTION libredesk_manual_reason_snapshot() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN
 -- v2.15's SENT trigger already owns the conversation input-clock lock.
 -- Annotation writes acquire the same lock before checking SENT eligibility.
 SELECT a.feedback_reason INTO NEW.feedback_reason
 FROM manual_review_annotations a
 JOIN conversation_messages n ON n.uuid=a.note_uuid AND n.conversation_id=a.conversation_id
 WHERE a.original_message_uuid=NEW.original_message_uuid AND a.agent_id=NEW.agent_id
 AND a.conversation_id=NEW.conversation_id
 AND n.private AND n.type='outgoing' AND n.sender_type='agent' AND NOT(n.meta ? 'deleted_at')
 AND n.meta->'ai_analysis'->>'kind'='n8n_manual_review'
 AND n.meta->'ai_analysis'->>'version'='1'
 AND n.meta->'ai_analysis'->>'original_message_uuid'=NEW.original_message_uuid::text
 AND COALESCE(n.meta->'ai_analysis'->>'suggested_message','')='';
 RETURN NEW;
 END $$;
 DROP TRIGGER IF EXISTS libredesk_manual_reason_snapshot ON manual_resolution_feedback;
 CREATE TRIGGER libredesk_manual_reason_snapshot BEFORE INSERT ON manual_resolution_feedback
 FOR EACH ROW EXECUTE FUNCTION libredesk_manual_reason_snapshot();
 CREATE OR REPLACE FUNCTION libredesk_manual_reason_immutable() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN
 IF NEW.feedback_reason IS DISTINCT FROM OLD.feedback_reason THEN
 RAISE EXCEPTION 'SENT feedback reason is immutable' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
 END $$;
 DROP TRIGGER IF EXISTS libredesk_manual_reason_immutable ON manual_resolution_feedback;
 CREATE TRIGGER libredesk_manual_reason_immutable BEFORE UPDATE OF feedback_reason ON manual_resolution_feedback
 FOR EACH ROW EXECUTE FUNCTION libredesk_manual_reason_immutable();
 `)
	if err != nil {
		return err
	}
	return tx.Commit()
}
