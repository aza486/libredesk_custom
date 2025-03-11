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

	return nil
}
