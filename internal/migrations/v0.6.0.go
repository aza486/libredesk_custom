package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_6_0 updates the database schema to v0.6.0.
func V0_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users ADD COLUMN IF NOT EXISTS reassign_replies BOOL DEFAULT FALSE NOT NULL;
	`)
	if err != nil {
		return err
	}
	return nil
}
