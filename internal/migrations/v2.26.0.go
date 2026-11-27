package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_26_0 stores staged template files and immutable message references to their
// existing media objects. No existing file or business row is rewritten.
func V2_26_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`ALTER TABLE message_templates DROP CONSTRAINT IF EXISTS message_templates_content_check;
 ALTER TABLE message_templates ADD CONSTRAINT message_templates_content_check CHECK (octet_length(content)<=131072);
 CREATE TABLE IF NOT EXISTS message_template_attachments (
 media_id INT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,
 template_id BIGINT REFERENCES message_templates(id) ON DELETE CASCADE,
 uploaded_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
 );
 CREATE INDEX IF NOT EXISTS message_template_attachments_template ON message_template_attachments(template_id);
 CREATE TABLE IF NOT EXISTS media_blob_references (
 media_id INT PRIMARY KEY REFERENCES media(id) ON DELETE CASCADE,
 source_media_id INT NOT NULL REFERENCES media(id) ON DELETE RESTRICT,
 user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
 conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
 draft_type TEXT NOT NULL CHECK (draft_type IN ('reply','private_note')),
 CHECK (media_id<>source_media_id)
 );
 CREATE INDEX IF NOT EXISTS media_blob_references_source ON media_blob_references(source_media_id);
 CREATE INDEX IF NOT EXISTS media_blob_references_draft ON media_blob_references(user_id,conversation_id,draft_type,source_media_id);
 CREATE TABLE IF NOT EXISTS media_storage_deletions (
 uuid UUID PRIMARY KEY,
 store media_store NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
 );
 CREATE OR REPLACE FUNCTION validate_message_template_content() RETURNS trigger LANGUAGE plpgsql AS $$
 DECLARE old_id bigint; new_id bigint; candidate bigint;
 BEGIN
 IF TG_TABLE_NAME='message_templates' THEN
 IF TG_OP<>'INSERT' THEN old_id:=OLD.id; END IF;
 IF TG_OP<>'DELETE' THEN new_id:=NEW.id; END IF;
 ELSE
 IF TG_OP<>'INSERT' THEN old_id:=OLD.template_id; END IF;
 IF TG_OP<>'DELETE' THEN new_id:=NEW.template_id; END IF;
 END IF;
 FOREACH candidate IN ARRAY ARRAY[old_id,new_id] LOOP
 IF EXISTS(SELECT 1 FROM message_templates t WHERE t.id=candidate AND length(btrim(t.content))=0 AND NOT EXISTS(SELECT 1 FROM message_template_attachments ta WHERE ta.template_id=t.id)) THEN
 RAISE EXCEPTION 'Message template requires text or an attachment' USING ERRCODE='23514';
 END IF;
 END LOOP;
 RETURN NULL;
 END $$;
 DROP TRIGGER IF EXISTS validate_message_template_content ON message_templates;
 CREATE CONSTRAINT TRIGGER validate_message_template_content AFTER INSERT OR UPDATE ON message_templates DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION validate_message_template_content();
 DROP TRIGGER IF EXISTS validate_message_template_content ON message_template_attachments;
 CREATE CONSTRAINT TRIGGER validate_message_template_content AFTER DELETE OR UPDATE ON message_template_attachments DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION validate_message_template_content();`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
