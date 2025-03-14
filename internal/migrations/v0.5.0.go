package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_5_0 updates the database schema to v0.5.0.
func V0_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'applied_sla_status') THEN
				CREATE TYPE "applied_sla_status" AS ENUM ('pending', 'breached', 'met', 'partially_met');
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		ALTER TABLE applied_slas ADD COLUMN IF NOT EXISTS status applied_sla_status DEFAULT 'pending' NOT NULL;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE INDEX IF NOT EXISTS index_applied_slas_on_status ON applied_slas(status);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO settings (key, value)
		VALUES 
			('notification.email.tls_type', '"starttls"'::jsonb),
			('notification.email.tls_skip_verify', 'false'::jsonb),
			('notification.email.hello_hostname', '""'::jsonb)
		ON CONFLICT (key) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		UPDATE inboxes
		SET config = 
			CASE 
				WHEN config#>'{imap,0,tls_type}' IS NULL 
				THEN jsonb_set(config, '{imap,0,tls_type}', '"tls"') 
				ELSE config 
			END,
			
			config = 
			CASE 
				WHEN config#>'{imap,0,tls_skip_verify}' IS NULL 
				THEN jsonb_set(config, '{imap,0,tls_skip_verify}', 'false') 
				ELSE config 
			END,

			config = 
			CASE 
				WHEN config#>'{imap,0,scan_inbox_since}' IS NULL 
				THEN jsonb_set(config, '{imap,0,scan_inbox_since}', '"48h"') 
				ELSE config 
			END,

			config = 
			CASE 
				WHEN config#>'{smtp,0,tls_type}' IS NULL 
				THEN jsonb_set(config, '{smtp,0,tls_type}', '"starttls"') 
				ELSE config 
			END,

			config = 
			CASE 
				WHEN config#>'{smtp,0,tls_skip_verify}' IS NULL 
				THEN jsonb_set(config, '{smtp,0,tls_skip_verify}', 'false') 
				ELSE config 
			END,

			config = 
			CASE 
				WHEN config#>'{smtp,0,hello_hostname}' IS NULL 
				THEN jsonb_set(config, '{smtp,0,hello_hostname}', '""') 
				ELSE config 
			END
		WHERE config->'imap' IS NOT NULL OR config->'smtp' IS NOT NULL;
	`)
	if err != nil {
		return err
	}

	return nil
}
