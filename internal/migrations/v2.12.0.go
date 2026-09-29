package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_12_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
 ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS access_mode TEXT NOT NULL DEFAULT 'public'
   CHECK (access_mode IN ('public', 'personal'));
 ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS owner_user_id BIGINT REFERENCES users(id) ON DELETE RESTRICT;
 DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='inboxes_access_owner_check') THEN
 ALTER TABLE inboxes ADD CONSTRAINT inboxes_access_owner_check CHECK (
   (access_mode='public' AND owner_user_id IS NULL) OR
   (access_mode='personal' AND owner_user_id IS NOT NULL AND channel='email'));
 END IF;
 END $$;
 CREATE OR REPLACE FUNCTION validate_inbox_access() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN
   IF TG_OP='UPDATE' AND NEW.access_mode <> OLD.access_mode AND EXISTS (SELECT 1 FROM conversations WHERE inbox_id=NEW.id) THEN
     RAISE EXCEPTION 'Cannot change access mode: inbox contains conversations' USING ERRCODE='23514', CONSTRAINT='inbox_access_mode_locked';
   END IF;
   IF NEW.access_mode='personal' THEN
     PERFORM id FROM users WHERE id=NEW.owner_user_id AND type='agent' AND email IS DISTINCT FROM 'System' AND enabled AND deleted_at IS NULL FOR SHARE;
     IF NOT FOUND THEN RAISE EXCEPTION 'Personal inbox owner must be an active employee' USING ERRCODE='23514'; END IF;
   END IF;
   RETURN NEW;
 END $$;
 DROP TRIGGER IF EXISTS validate_inbox_access ON inboxes;
 CREATE TRIGGER validate_inbox_access BEFORE INSERT OR UPDATE ON inboxes FOR EACH ROW EXECUTE FUNCTION validate_inbox_access();
 CREATE OR REPLACE FUNCTION protect_personal_inbox_owner() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN
   IF (NEW.deleted_at IS NOT NULL OR NOT NEW.enabled OR NEW.type <> 'agent') AND EXISTS (SELECT 1 FROM inboxes WHERE owner_user_id=OLD.id AND access_mode='personal') THEN
     RAISE EXCEPTION 'Reassign personal inboxes before deleting or disabling their owner' USING ERRCODE='23514', CONSTRAINT='personal_inbox_owner_active';
   END IF;
   RETURN NEW;
 END $$;
 DROP TRIGGER IF EXISTS protect_personal_inbox_owner ON users;
 CREATE TRIGGER protect_personal_inbox_owner BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION protect_personal_inbox_owner();
 `)
	return err
}
