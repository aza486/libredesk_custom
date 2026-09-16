package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0 introduces the canonical many-to-many user assignment relation.
// assigned_user_id remains a compatibility projection while callers migrate.
func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_assignees (
			conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			assigned_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (conversation_id, user_id)
		);
		CREATE INDEX IF NOT EXISTS index_conversation_assignees_on_user_id
			ON conversation_assignees(user_id, conversation_id);
		INSERT INTO conversation_assignees (conversation_id, user_id)
		SELECT id, assigned_user_id FROM conversations WHERE assigned_user_id IS NOT NULL
		ON CONFLICT (conversation_id, user_id) DO NOTHING;
	`)
	return err
}
