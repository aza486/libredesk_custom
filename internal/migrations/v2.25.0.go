package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_25_0 adds message snippets, separate from send-time email templates.
func V2_25_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS message_templates (
 id BIGSERIAL PRIMARY KEY,
 uuid UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
 name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 160),
 content TEXT NOT NULL CHECK (length(btrim(content)) > 0 AND octet_length(content) <= 131072),
 scope TEXT NOT NULL CHECK (scope IN ('personal','customer_support')),
 owner_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
 created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 revision BIGINT NOT NULL DEFAULT 1 CHECK (revision > 0),
 CONSTRAINT message_templates_owner_check CHECK ((scope='personal' AND owner_id IS NOT NULL) OR (scope='customer_support' AND owner_id IS NULL))
 );
 CREATE INDEX IF NOT EXISTS message_templates_owner_name ON message_templates(owner_id,lower(name),id) WHERE scope='personal';
 CREATE INDEX IF NOT EXISTS message_templates_support_name ON message_templates(lower(name),id) WHERE scope='customer_support';
 UPDATE roles SET permissions=array_append(permissions,'message_templates:customer_support')
 WHERE name='Kundensupport' AND NOT permissions @> ARRAY['message_templates:customer_support']::text[];`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
