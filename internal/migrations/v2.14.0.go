package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_14_0 installs a monotonic email input clock. The trigger and conditional
// resolve serialize on its row; deleted messages never turn the clock backwards.
func V2_14_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
 LOCK TABLE conversation_messages IN SHARE ROW EXCLUSIVE MODE;
 CREATE TABLE IF NOT EXISTS conversation_mail_versions (
 conversation_id BIGINT PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
 version BIGINT NOT NULL DEFAULT 0 CHECK(version>=0),
 resolved_version BIGINT
 );
 WITH ranked AS (
 SELECT m.id,ROW_NUMBER() OVER(PARTITION BY m.conversation_id ORDER BY m.id) AS version
 FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id JOIN inboxes i ON i.id=c.inbox_id
 WHERE i.channel='email' AND m.type='incoming' AND m.sender_type='contact' AND NOT m.private
 ) UPDATE conversation_messages m SET meta=COALESCE(m.meta,'{}'::jsonb)||jsonb_build_object('mail_input_version',ranked.version::text)
 FROM ranked WHERE m.id=ranked.id AND NOT (COALESCE(m.meta,'{}'::jsonb) ? 'mail_input_version');
 INSERT INTO conversation_mail_versions(conversation_id,version)
 SELECT conversation_id,MAX((meta->>'mail_input_version')::bigint) FROM conversation_messages
 WHERE meta ? 'mail_input_version' GROUP BY conversation_id
 ON CONFLICT(conversation_id) DO UPDATE SET version=GREATEST(conversation_mail_versions.version,excluded.version);
 CREATE OR REPLACE FUNCTION libredesk_mail_input_clock() RETURNS trigger LANGUAGE plpgsql AS $$
 DECLARE v BIGINT; reopen BOOLEAN;
 BEGIN
 IF NEW.type='incoming' AND NEW.sender_type='contact' AND NOT NEW.private AND EXISTS(
 SELECT 1 FROM conversations c JOIN inboxes i ON i.id=c.inbox_id WHERE c.id=NEW.conversation_id AND i.channel='email') THEN
 INSERT INTO conversation_mail_versions(conversation_id) VALUES(NEW.conversation_id) ON CONFLICT DO NOTHING;
 SELECT resolved_version IS NOT NULL INTO reopen FROM conversation_mail_versions WHERE conversation_id=NEW.conversation_id FOR UPDATE;
 UPDATE conversation_mail_versions SET version=version+1,resolved_version=NULL WHERE conversation_id=NEW.conversation_id RETURNING version INTO v;
 NEW.meta:=COALESCE(NEW.meta,'{}'::jsonb)||jsonb_build_object('mail_input_version',v::text);
 -- If this input waited for a V2 resolve, atomically undo that resolution.
 -- Ordinary incoming hooks still handle their existing notifications afterwards.
 IF reopen THEN
 UPDATE conversations SET status_id=(SELECT id FROM conversation_statuses WHERE name='Open'),resolved_at=NULL,updated_at=NOW()
 WHERE id=NEW.conversation_id AND status_id IN(SELECT id FROM conversation_statuses WHERE category='resolved');
 END IF;
 END IF;
 RETURN NEW;
 END $$;
 DROP TRIGGER IF EXISTS libredesk_mail_input_clock ON conversation_messages;
 CREATE TRIGGER libredesk_mail_input_clock BEFORE INSERT ON conversation_messages FOR EACH ROW EXECUTE FUNCTION libredesk_mail_input_clock();
 CREATE INDEX IF NOT EXISTS index_mail_completion_pending ON conversation_messages(id)
 WHERE type='outgoing' AND NOT private AND meta ? 'mail_completion' AND NOT (meta->'mail_completion' ? 'committed');
 `)
	if err != nil {
		return err
	}
	return tx.Commit()
}
