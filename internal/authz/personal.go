package authz

import "fmt"

// PersonalAccessSQL adds only the personal restriction; existing public access
// rules remain the caller's responsibility. alias must be a code constant.
func PersonalAccessSQL(alias string, userID int, isAdmin bool) string {
	return fmt.Sprintf(`(COALESCE(%s.custom_attributes->>'access_mode', 'public') <> 'personal' OR %t OR EXISTS (SELECT 1 FROM personal_inbox_owners pio WHERE pio.inbox_id=%s.inbox_id AND pio.user_id=%d))`, alias, isAdmin, alias, userID)
}
