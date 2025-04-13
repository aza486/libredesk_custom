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

	// Add `contacts:manage` permission to Admin role
	_, err = db.Exec(`
		UPDATE roles 
		SET permissions = array_append(permissions, 'contacts:manage')
		WHERE name = 'Admin' AND NOT ('contacts:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	// Add `custom_attributes:manage` permission to Admin role
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'custom_attributes:manage')
		WHERE name = 'Admin' AND NOT ('custom_attributes:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	// Create table for custom attribute definitions
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS custom_attribute_definitions (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			"name" TEXT NOT NULL,
			description TEXT NOT NULL,
			applies_to TEXT NOT NULL,
			key TEXT NOT NULL,
			values TEXT[] DEFAULT '{}'::TEXT[] NOT NULL,
			data_type TEXT NOT NULL,
			regex TEXT NULL,
			regex_hint TEXT NULL,
			CONSTRAINT constraint_custom_attribute_definitions_on_name CHECK (length("name") <= 140),
			CONSTRAINT constraint_custom_attribute_definitions_on_description CHECK (length(description) <= 300),
			CONSTRAINT constraint_custom_attribute_definitions_on_key CHECK (length(key) <= 140),
			CONSTRAINT constraint_custom_attribute_definitions_on_applies_to CHECK (length(applies_to) <= 50),
			CONSTRAINT constraint_custom_attribute_definitions_on_data_type CHECK (length(data_type) <= 100),
			CONSTRAINT constraint_custom_attribute_definitions_on_regex CHECK (length(regex) <= 1000),
			CONSTRAINT constraint_custom_attribute_definitions_on_regex_hint CHECK (length(regex_hint) <= 1000),
			CONSTRAINT constraint_custom_attribute_definitions_key_applies_to_unique UNIQUE (key, applies_to)
		);

	`)
	if err != nil {
		return err
	}

	// Create contact notes table.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS contact_notes (
			id SERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			contact_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			note TEXT NOT NULL,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
		);
		CREATE INDEX index_contact_notes_on_contact_id_created_at ON contact_notes (contact_id, created_at);
	`)
	if err != nil {
		return err
	}
	return nil
}
