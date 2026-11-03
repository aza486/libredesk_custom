package migrations

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_19_0 extends sidebar folder scopes without changing existing definitions.
// my_inbox uses the employee ID, not a personal inbox ID.
func V2_19_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`ALTER TABLE sidebar_folders
 DROP CONSTRAINT IF EXISTS sidebar_folders_scope_type_check,
 DROP CONSTRAINT IF EXISTS sidebar_folders_scope_id_check,
 ADD CONSTRAINT sidebar_folders_scope_type_check CHECK (scope_type IN ('customer','team','personal','my_inbox')),
 ADD CONSTRAINT sidebar_folders_scope_id_check CHECK ((scope_type='customer' AND scope_id=0) OR (scope_type IN ('team','personal','my_inbox') AND scope_id>0))`); err != nil {
		return err
	}
	return tx.Commit()
}
