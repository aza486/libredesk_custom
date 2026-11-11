package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_21_0 links email inboxes to the existing shared outgoing templates.
func V2_21_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`
ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS default_signature_id INTEGER
 REFERENCES templates(id) ON DELETE SET NULL
 CHECK (default_signature_id IS NULL OR channel = 'email');
CREATE INDEX IF NOT EXISTS inboxes_default_signature_idx ON inboxes(default_signature_id)
 WHERE default_signature_id IS NOT NULL;

-- Enforce template type even for concurrent writes or changes of template type.
CREATE OR REPLACE FUNCTION check_inbox_signature_type() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.default_signature_id IS NOT NULL THEN
  PERFORM id FROM templates WHERE id=NEW.default_signature_id AND type='email_outgoing' FOR SHARE;
  IF NOT FOUND THEN
   RAISE EXCEPTION 'Inbox signature must be an outgoing email template' USING ERRCODE='23514';
  END IF;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS inbox_signature_type ON inboxes;
CREATE TRIGGER inbox_signature_type BEFORE INSERT OR UPDATE OF default_signature_id,channel ON inboxes
 FOR EACH ROW EXECUTE FUNCTION check_inbox_signature_type();

CREATE OR REPLACE FUNCTION protect_inbox_signature_type() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.type <> 'email_outgoing' AND EXISTS(SELECT 1 FROM inboxes WHERE default_signature_id=NEW.id) THEN
  RAISE EXCEPTION 'Outgoing template is used by an inbox' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS protect_inbox_signature_type ON templates;
CREATE TRIGGER protect_inbox_signature_type BEFORE UPDATE OF type ON templates
 FOR EACH ROW EXECUTE FUNCTION protect_inbox_signature_type();
`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
