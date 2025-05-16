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

	// Add contacts permissions to Admin role
	permissionsToAdd := []string{
		"contacts:read_all",
		"contacts:read",
		"contacts:write",
		"contacts:block",
		"contact_notes:read",
		"contact_notes:write",
		"contact_notes:delete",
	}
	for _, permission := range permissionsToAdd {
		_, err = db.Exec(`
			UPDATE roles 
			SET permissions = array_append(permissions, $1)
			WHERE name = 'Admin' AND NOT ($1 = ANY(permissions));
		`, permission)
		if err != nil {
			return err
		}
	}

	// Add `contacts:read` permission to Agent role
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'contacts:read')
		WHERE name = 'Agent' AND NOT ('contacts:read' = ANY(permissions));
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
		CREATE INDEX IF NOT EXISTS index_contact_notes_on_contact_id_created_at ON contact_notes (contact_id, created_at);
	`)
	if err != nil {
		return err
	}

	// Add a new "Last reply at" column to the conversations table and populate it.
	_, err = db.Exec(`
		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_reply_at TIMESTAMPTZ NULL;
		UPDATE conversations SET last_reply_at=first_reply_at WHERE last_reply_at IS NULL AND first_reply_at IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	// Create activity_log_type enum if not exists
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_type WHERE typname = 'activity_log_type'
			) THEN
				CREATE TYPE activity_log_type AS ENUM ('agent_login', 'agent_logout', 'agent_away', 'agent_away_reassigned', 'agent_online');
			END IF;
		END
		$$;
	`)
	if err != nil {
		return err
	}

	// Create activity_logs table if not exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS activity_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			activity_type activity_log_type NOT NULL,
			activity_description TEXT NOT NULL,
			actor_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			target_model_type TEXT NOT NULL,
			target_model_id BIGINT NOT NULL,
			ip INET
		);
		CREATE INDEX IF NOT EXISTS index_activity_logs_on_actor_id ON activity_logs (actor_id);
		CREATE INDEX IF NOT EXISTS index_activity_logs_on_activity_type ON activity_logs (activity_type);
		CREATE INDEX IF NOT EXISTS index_activity_logs_on_created_at ON activity_logs (created_at);
	`)
	if err != nil {
		return err
	}

	// Add `activity_logs:manage` permission to Admin role
	_, err = db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'activity_logs:manage')
		WHERE name = 'Admin' AND NOT ('activity_logs:manage' = ANY(permissions));
	`)
	if err != nil {
		return err
	}

	return nil
}
