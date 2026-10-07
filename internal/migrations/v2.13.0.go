package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V2_13_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS personal_inbox_owners (
			inbox_id INT NOT NULL REFERENCES inboxes(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (inbox_id, user_id)
		);
		CREATE INDEX IF NOT EXISTS index_personal_inbox_owners_on_user_id
			ON personal_inbox_owners(user_id, inbox_id);
		INSERT INTO personal_inbox_owners (inbox_id, user_id)
		SELECT id, owner_user_id FROM inboxes
		WHERE access_mode = 'personal' AND owner_user_id IS NOT NULL
		ON CONFLICT (inbox_id, user_id) DO NOTHING;

		CREATE OR REPLACE FUNCTION validate_personal_inbox_owner() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM inboxes WHERE id = NEW.inbox_id AND access_mode = 'personal'
			) THEN
				RAISE EXCEPTION 'Owners can only be added to personal inboxes' USING ERRCODE='23514';
			END IF;
			PERFORM id FROM users WHERE id = NEW.user_id AND type = 'agent' AND email IS DISTINCT FROM 'System' AND enabled AND deleted_at IS NULL FOR SHARE;
			IF NOT FOUND THEN
				RAISE EXCEPTION 'Personal inbox owners must be active employees' USING ERRCODE='23514';
			END IF;
			RETURN NEW;
		END $$;
		DROP TRIGGER IF EXISTS validate_personal_inbox_owner ON personal_inbox_owners;
		CREATE TRIGGER validate_personal_inbox_owner BEFORE INSERT OR UPDATE ON personal_inbox_owners FOR EACH ROW EXECUTE FUNCTION validate_personal_inbox_owner();

		CREATE OR REPLACE FUNCTION sync_personal_inbox_primary_owner() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.access_mode = 'personal' AND NEW.owner_user_id IS NOT NULL THEN
				INSERT INTO personal_inbox_owners (inbox_id, user_id)
				VALUES (NEW.id, NEW.owner_user_id)
				ON CONFLICT (inbox_id, user_id) DO NOTHING;
			END IF;
			RETURN NEW;
		END $$;
		DROP TRIGGER IF EXISTS sync_personal_inbox_primary_owner ON inboxes;
		CREATE TRIGGER sync_personal_inbox_primary_owner AFTER INSERT OR UPDATE ON inboxes FOR EACH ROW EXECUTE FUNCTION sync_personal_inbox_primary_owner();

		CREATE OR REPLACE FUNCTION protect_personal_inbox_owner() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF TG_OP = 'DELETE' AND EXISTS (
				SELECT 1 FROM inboxes WHERE id = OLD.inbox_id AND access_mode = 'personal'
			) AND (SELECT COUNT(*) FROM personal_inbox_owners WHERE inbox_id = OLD.inbox_id) <= 1 THEN
				RAISE EXCEPTION 'A personal inbox must have at least one owner' USING ERRCODE='23514', CONSTRAINT='personal_inbox_last_owner';
			END IF;
			RETURN OLD;
		END $$;
		DROP TRIGGER IF EXISTS protect_personal_inbox_owner ON personal_inbox_owners;
		CREATE TRIGGER protect_personal_inbox_owner BEFORE DELETE ON personal_inbox_owners FOR EACH ROW EXECUTE FUNCTION protect_personal_inbox_owner();

		CREATE OR REPLACE FUNCTION protect_personal_inbox_owner_user() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			IF (NEW.deleted_at IS NOT NULL OR NOT NEW.enabled OR NEW.type <> 'agent') AND EXISTS (
				SELECT 1 FROM personal_inbox_owners WHERE user_id = OLD.id
			) THEN
				RAISE EXCEPTION 'Reassign personal inboxes before deleting or disabling their owner' USING ERRCODE='23514', CONSTRAINT='personal_inbox_owner_active';
			END IF;
			RETURN NEW;
		END $$;
		DROP TRIGGER IF EXISTS protect_personal_inbox_owner ON users;
		CREATE TRIGGER protect_personal_inbox_owner BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION protect_personal_inbox_owner_user();
	`)
	return err
}
