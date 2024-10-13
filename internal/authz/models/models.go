package models

const (
	// Conversation
	PermConversationsReadAll            = "conversations:read_all"
	PermConversationsReadUnassigned     = "conversations:read_unassigned"
	PermConversationsReadAssigned       = "conversations:read_assigned"
	PermConversationsRead               = "conversations:read"
	PermConversationsUpdateUserAssignee = "conversations:update_user_assignee"
	PermConversationsUpdateTeamAssignee = "conversations:update_team_assignee"
	PermConversationsUpdatePriority     = "conversations:update_priority"
	PermConversationsUpdateStatus       = "conversations:update_status"
	PermConversationsUpdateTags         = "conversations:update_tags"
	PermMessagesRead                    = "messages:read"
	PermMessagesWrite                   = "messages:write"

	// Conversation Status
	PermStatusRead   = "status:read"
	PermStatusWrite  = "status:write"
	PermStatusDelete = "status:delete"

	// Admin
	PermAdminRead = "admin:read"

	// Settings
	PermSettingsGeneralWrite       = "settings_general:write"
	PermSettingsNotificationsWrite = "settings_notifications:write"
	PermSettingsNotificationsRead  = "settings_notifications:read"

	// OpenID Connect SSO
	PermOIDCRead   = "oidc:read"
	PermOIDCWrite  = "oidc:write"
	PermOIDCDelete = "oidc:delete"

	// Tags
	PermTagsWrite  = "tags:write"
	PermTagsDelete = "tags:delete"

	// Canned Responses
	PermCannedResponsesWrite  = "canned_responses:write"
	PermCannedResponsesDelete = "canned_responses:delete"

	// Dashboard
	PermDashboardGlobalRead = "dashboard_global:read"

	// Users
	PermUsersRead  = "users:read"
	PermUsersWrite = "users:write"

	// Teams
	PermTeamsRead  = "teams:read"
	PermTeamsWrite = "teams:write"

	// Automations
	PermAutomationsRead   = "automations:read"
	PermAutomationsWrite  = "automations:write"
	PermAutomationsDelete = "automations:delete"

	// Inboxes
	PermInboxesRead   = "inboxes:read"
	PermInboxesWrite  = "inboxes:write"
	PermInboxesDelete = "inboxes:delete"

	// Roles
	PermRolesRead   = "roles:read"
	PermRolesWrite  = "roles:write"
	PermRolesDelete = "roles:delete"

	// Templates
	PermTemplatesRead   = "templates:read"
	PermTemplatesWrite  = "templates:write"
	PermTemplatesDelete = "templates:delete"
)

var validPermissions = map[string]struct{}{
	PermConversationsReadAll:            {},
	PermConversationsReadUnassigned:     {},
	PermConversationsReadAssigned:       {},
	PermConversationsRead:               {},
	PermConversationsUpdateUserAssignee: {},
	PermConversationsUpdateTeamAssignee: {},
	PermConversationsUpdatePriority:     {},
	PermConversationsUpdateStatus:       {},
	PermConversationsUpdateTags:         {},
	PermMessagesRead:                    {},
	PermMessagesWrite:                   {},
	PermStatusRead:                      {},
	PermStatusWrite:                     {},
	PermStatusDelete:                    {},
	PermAdminRead:                       {},
	PermSettingsGeneralWrite:            {},
	PermSettingsNotificationsWrite:      {},
	PermSettingsNotificationsRead:       {},
	PermOIDCRead:                        {},
	PermOIDCWrite:                       {},
	PermOIDCDelete:                      {},
	PermTagsWrite:                       {},
	PermTagsDelete:                      {},
	PermCannedResponsesWrite:            {},
	PermCannedResponsesDelete:           {},
	PermDashboardGlobalRead:             {},
	PermUsersRead:                       {},
	PermUsersWrite:                      {},
	PermTeamsRead:                       {},
	PermTeamsWrite:                      {},
	PermAutomationsRead:                 {},
	PermAutomationsWrite:                {},
	PermAutomationsDelete:               {},
	PermInboxesRead:                     {},
	PermInboxesWrite:                    {},
	PermInboxesDelete:                   {},
	PermRolesRead:                       {},
	PermRolesWrite:                      {},
	PermRolesDelete:                     {},
	PermTemplatesRead:                   {},
	PermTemplatesWrite:                  {},
	PermTemplatesDelete:                 {},
}

// IsValidPermission retuns true if it's a valid perm.
func IsValidPermission(permission string) bool {
	_, exists := validPermissions[permission]
	return exists
}
