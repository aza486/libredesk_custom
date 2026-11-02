package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_17_0 grants the separate learning permission only to the built-in Admin role.
// The pg-n8n schema is installed separately from the explicit SQL artifact.
func V2_17_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	_, err := db.Exec(`UPDATE roles SET permissions=array_append(permissions,'ai_learning:manage') WHERE name='Admin' AND NOT ('ai_learning:manage'=ANY(permissions))`)
	return err
}
