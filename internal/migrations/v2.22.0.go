package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_22_0 adds an optional business phone without defaulting existing users.
func V2_22_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	_, err := db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS work_phone_number VARCHAR(64)`)
	return err
}
