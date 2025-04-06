package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_6_0 updates the database schema to v0.6.0.
func V0_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ NULL;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_enum e
				JOIN pg_type t ON t.oid = e.enumtypid
				WHERE t.typname = 'user_availability_status'
				AND e.enumlabel = 'away_and_reassigning'
			) THEN
				ALTER TYPE user_availability_status ADD VALUE 'away_and_reassigning';
			END IF;
		END
		$$;
	`)
	if err != nil {
		return err
	}
	return nil
}
