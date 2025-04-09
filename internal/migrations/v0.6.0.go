package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_6_0 updates the database schema to v0.6.0.
func V0_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Add new column for last login timestamp
	_, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ NULL;
	`)
	if err != nil {
		return err
	}

	// Add new enum value for user availability status
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

	// Add new column for phone number calling code
	_, err = db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_number_calling_code TEXT NULL;
	`)
	if err != nil {
		return err
	}

	// Add constraint for phone number calling code
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1
				FROM information_schema.constraint_column_usage
				WHERE table_name = 'users'
				AND column_name = 'phone_number_calling_code'
				AND constraint_name = 'constraint_users_on_phone_number_calling_code'
			) THEN
				ALTER TABLE users
				ADD CONSTRAINT constraint_users_on_phone_number_calling_code
				CHECK (LENGTH(phone_number_calling_code) <= 10);
			END IF;
		END
		$$;
	`)
	if err != nil {
		return err
	}
	return nil
}
