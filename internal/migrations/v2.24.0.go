package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_24_0 supports contactless private internal conversations and stored-only
// messages. No existing conversation, contact or message is rewritten.
func V2_24_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`ALTER TYPE message_type ADD VALUE IF NOT EXISTS 'internal';
 ALTER TYPE message_status ADD VALUE IF NOT EXISTS 'stored';
 ALTER TABLE conversations ALTER COLUMN contact_id DROP NOT NULL;
 DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='conversations'::regclass AND conname='conversations_contact_required') THEN
 ALTER TABLE conversations ADD CONSTRAINT conversations_contact_required CHECK
 (contact_id IS NOT NULL OR custom_attributes @> '{"internal":true,"private":true}'::jsonb);
 END IF;
 END $$;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
