package authz

import "fmt"

// PersonalAccessSQL adds only the personal restriction; existing public access
// rules remain the caller's responsibility. alias must be a code constant.
func PersonalAccessSQL(alias string, userID int, isAdmin bool) string {
	return fmt.Sprintf(`(COALESCE(%s.custom_attributes->>'access_mode', 'public') <> 'personal' OR %t OR (%s.custom_attributes->'visible_users') @> '[%d]')`, alias, isAdmin, alias, userID)
}
