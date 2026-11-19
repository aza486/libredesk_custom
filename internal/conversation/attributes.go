package conversation

var systemAttributeKeys = [...]string{
	"private", "internal", "creator_id", "visible_users", "visibility_managers",
	"customer_visibility", "access_mode", "owner_user_id", "_spam_previous_state",
}

// IsSystemAttribute identifies fields owned by ticket creation, visibility,
// inbox ownership and Spam transitions, rather than custom attribute editors.
func IsSystemAttribute(key string) bool {
	for _, systemKey := range systemAttributeKeys {
		if key == systemKey {
			return true
		}
	}
	return false
}
