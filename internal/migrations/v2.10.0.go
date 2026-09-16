package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_10_0 makes existing non-private tickets customer-visible only. Active
// Kundensupport members are visibility managers and current assignees retain
// visibility when the restriction is enabled.
func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		WITH support AS (
			SELECT COALESCE(jsonb_agg(u.id), '[]'::jsonb) AS ids
			FROM users u
			JOIN user_roles ur ON ur.user_id = u.id
			JOIN roles r ON r.id = ur.role_id
			WHERE r.name = 'Kundensupport' AND u.type = 'agent' AND u.enabled = true
		)
		UPDATE conversations c
		SET custom_attributes = c.custom_attributes || jsonb_build_object(
			'customer_visibility', true,
			'visibility_managers', support.ids,
			'visible_users', (
				SELECT COALESCE(jsonb_agg(DISTINCT user_id), '[]'::jsonb)
				FROM (
					SELECT value::bigint AS user_id
					FROM jsonb_array_elements_text(COALESCE(c.custom_attributes->'visible_users', '[]'::jsonb))
					UNION
					SELECT value::bigint FROM jsonb_array_elements_text(support.ids)
					UNION
					SELECT user_id FROM conversation_assignees WHERE conversation_id = c.id
				) visible
			)
		)
		FROM support
		WHERE COALESCE((c.custom_attributes->>'private')::boolean, false) = false;
	`)
	return err
}
