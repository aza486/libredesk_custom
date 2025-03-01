package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_4_0 updates the database schema to V0_4_0.
func V0_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Admin role gets the new ai:manage permission, as this user is supposed to have all permissions.
	_, err := db.Exec(`
		UPDATE roles 
		SET permissions = array_append(permissions, 'ai:manage')
		WHERE name = 'Admin' AND NOT ('ai:manage' = ANY(permissions));
	`)
	return err
}
