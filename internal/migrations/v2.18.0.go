package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_18_0 adds shared logical sidebar folders to the LibreDesk database.
func V2_18_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sidebar_folders (
 id BIGSERIAL PRIMARY KEY,
 name TEXT NOT NULL CHECK (char_length(name) BETWEEN 2 AND 140),
 scope_type TEXT NOT NULL CHECK (scope_type IN ('customer','team','personal')),
 scope_id BIGINT NOT NULL CHECK ((scope_type='customer' AND scope_id=0) OR (scope_type<>'customer' AND scope_id>0)),
 filters JSONB NOT NULL,
 sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order>=0),
 show_count BOOLEAN NOT NULL DEFAULT true,
 created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
 );
 CREATE INDEX IF NOT EXISTS sidebar_folders_scope_order ON sidebar_folders(scope_type,scope_id,sort_order,id);`)
	return err
}
