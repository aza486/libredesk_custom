package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_15_0 commits the first eligible SENT reply and its manual-feedback outbox
// record together. No historical messages are backfilled or submitted.
func V2_15_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
 CREATE TABLE IF NOT EXISTS manual_resolution_feedback (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
 original_message_uuid UUID NOT NULL UNIQUE,
 outgoing_message_uuid UUID NOT NULL UNIQUE,
 note_uuid UUID NOT NULL,
 agent_id BIGINT NOT NULL,
 human_answer TEXT NOT NULL,
 sent_at TIMESTAMPTZ NOT NULL,
 attempt_count INTEGER NOT NULL DEFAULT 0 CHECK(attempt_count>=0),
 attempt_at TIMESTAMPTZ,
 next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 committed_at TIMESTAMPTZ
 );
 CREATE INDEX IF NOT EXISTS index_manual_resolution_feedback_pending
 ON manual_resolution_feedback(next_attempt_at) WHERE committed_at IS NULL;
 CREATE INDEX IF NOT EXISTS index_private_analysis_original
 ON conversation_messages(conversation_id,(meta->'ai_analysis'->>'original_message_uuid'))
 WHERE private AND type='outgoing';

 CREATE OR REPLACE FUNCTION libredesk_manual_resolution_feedback() RETURNS trigger LANGUAGE plpgsql AS $$
 DECLARE mc JSONB;
 BEGIN
 IF NEW.status<>'sent' OR OLD.status='sent' OR NEW.private OR NEW.type<>'outgoing' OR NEW.sender_type<>'agent'
 OR NEW.meta ? 'deleted_at' OR NEW.meta ? 'ai_feedback_original_message_uuid' OR NEW.meta ? 'ai_response'
 OR COALESCE(NEW.meta->>'is_automated','false')='true'
 OR COALESCE(NEW.meta->>'continuity_email','false')='true'
 OR COALESCE(NEW.meta->>'is_csat','false')='true' THEN RETURN NEW; END IF;
 mc:=NEW.meta->'mail_completion';
 IF mc IS NULL OR mc->>'manual_review_note_uuid' IS NULL OR mc->>'response_source' IS NOT NULL OR mc->>'snapshot_ready' IS DISTINCT FROM 'true'
 OR NOT(mc ? 'sent_at') OR COALESCE(btrim(mc->>'sent_message'),'')='' THEN RETURN NEW; END IF;

 -- Serialize eligibility with incoming mail's existing version clock. This
 -- changes neither the incoming clock nor conditional resolve behavior.
 PERFORM 1 FROM conversation_mail_versions WHERE conversation_id=NEW.conversation_id FOR UPDATE;
 INSERT INTO manual_resolution_feedback(conversation_id,original_message_uuid,outgoing_message_uuid,note_uuid,agent_id,human_answer,sent_at)
 SELECT NEW.conversation_id,original.uuid,NEW.uuid,n.uuid,NEW.sender_id,mc->>'sent_message',NEW.updated_at
 FROM conversations c JOIN inboxes i ON i.id=c.inbox_id
 JOIN users agent ON agent.id=NEW.sender_id AND agent.type='agent' AND agent.email IS DISTINCT FROM 'System'
 JOIN conversation_mail_versions clock ON clock.conversation_id=c.id
 JOIN conversation_messages original ON original.conversation_id=c.id AND original.uuid::text=mc->>'source_message_uuid'
 JOIN conversation_messages n ON n.conversation_id=c.id AND n.uuid::text=mc->>'manual_review_note_uuid'
 WHERE c.id=NEW.conversation_id AND i.channel='email'
 AND original.type='incoming' AND original.sender_type='contact' AND NOT original.private AND NOT(original.meta ? 'deleted_at')
 AND NEW.meta->>'reply_to_message_uuid'=original.uuid::text
 AND mc->>'expected_incoming_version'=original.meta->>'mail_input_version'
 AND mc->>'expected_incoming_version'=mc->>'queued_incoming_version'
 AND mc->>'expected_incoming_version'=clock.version::text
 AND n.private AND n.type='outgoing' AND n.sender_type='agent' AND NOT(n.meta ? 'deleted_at')
 AND n.id>original.id AND n.id<NEW.id
 AND n.meta->'ai_analysis'->>'kind'='n8n_manual_review'
 AND n.meta->'ai_analysis'->>'version'='1'
 AND n.meta->'ai_analysis'->>'original_message_uuid'=original.uuid::text
 AND COALESCE(n.meta->'ai_analysis'->>'suggested_message','')=''
 AND NOT EXISTS(SELECT 1 FROM conversation_messages a WHERE a.conversation_id=c.id
   AND a.private AND a.type='outgoing' AND NOT(a.meta ? 'deleted_at')
   AND a.meta->'ai_analysis'->>'kind'='n8n_ai_analysis' AND a.meta->'ai_analysis'->>'version'='1'
   AND a.meta->'ai_analysis'->>'original_message_uuid'=original.uuid::text)
 AND NOT EXISTS(SELECT 1 FROM conversation_messages r WHERE r.conversation_id=c.id
   AND r.uuid<>NEW.uuid AND r.type='outgoing' AND NOT r.private AND r.sender_type='agent' AND r.status='sent'
   AND r.id>original.id
   AND COALESCE(r.meta->>'is_automated','false')<>'true'
   AND COALESCE(r.meta->>'continuity_email','false')<>'true'
   AND COALESCE(r.meta->>'is_csat','false')<>'true'
   AND EXISTS(SELECT 1 FROM users u WHERE u.id=r.sender_id AND u.type='agent' AND u.email IS DISTINCT FROM 'System')
   AND (
    r.meta->'mail_completion'->>'source_message_uuid'=original.uuid::text
    OR r.meta->>'ai_feedback_original_message_uuid'=original.uuid::text
    OR r.meta->'ai_response'->>'source_message_uuid'=original.uuid::text
    OR (r.meta->'mail_completion'->>'source_message_uuid' IS NULL
        AND r.meta->>'ai_feedback_original_message_uuid' IS NULL
        AND r.meta->'ai_response'->>'source_message_uuid' IS NULL)))
 ON CONFLICT DO NOTHING;
 RETURN NEW;
 END $$;
 DROP TRIGGER IF EXISTS libredesk_manual_resolution_feedback ON conversation_messages;
 CREATE TRIGGER libredesk_manual_resolution_feedback AFTER UPDATE OF status ON conversation_messages
 FOR EACH ROW EXECUTE FUNCTION libredesk_manual_resolution_feedback();
 `)
	if err != nil {
		return err
	}
	return tx.Commit()
}
