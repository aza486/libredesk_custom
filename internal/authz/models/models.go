package models

const (
	// Conversation
	PermConversationsReadAll            = "conversations:read_all"
	PermConversationsReadUnassigned     = "conversations:read_unassigned"
	PermConversationsReadAssigned       = "conversations:read_assigned"
	PermConversationsReadTeamInbox      = "conversations:read_team_inbox"
	PermConversationsReadTeamAll        = "conversations:read_team_all"
	PermConversationsRead               = "conversations:read"
	PermConversationsUpdateUserAssignee = "conversations:update_user_assignee"
	PermConversationsUpdateTeamAssignee = "conversations:update_team_assignee"
	PermConversationsUpdatePriority     = "conversations:update_priority"
	PermConversationsUpdateStatus       = "conversations:update_status"
	PermConversationsUpdateTags         = "conversations:update_tags"
	// Explicit API-only classification grant; never implied by Admin or API keys.
	PermConversationsAddSystemTags = "conversations:add_system_tags"
	PermConversationWrite          = "conversations:write"
	PermMessagesRead               = "messages:read"
	PermMessagesWrite              = "messages:write"
	// Explicit API-only grant for stored spam detection notes; never implied by Admin.
	PermMessagesWriteSpamDetectionNote = "messages:write_spam_detection_note"
	PermMessagesWriteAsContact         = "messages:write_as_contact"

	// View
	PermViewManage        = "view:manage"
	PermSharedViewsManage = "shared_views:manage"

	// Status
	PermStatusManage = "status:manage"

	// Tags
	PermTagsManage = "tags:manage"

	// Macros
	PermMacrosManage = "macros:manage"

	// Users
	PermUsersManage = "users:manage"

	// Teams
	PermTeamsManage = "teams:manage"

	// Automations
	PermAutomationsManage = "automations:manage"

	// Inboxes
	PermInboxesManage = "inboxes:manage"

	// Roles
	PermRolesManage = "roles:manage"

	// Webhooks
	PermWebhooksManage = "webhooks:manage"

	// Context Links
	PermContextLinksManage = "context_links:manage"

	// Templates
	PermTemplatesManage         = "templates:manage"
	PermMessageTemplatesSupport = "message_templates:customer_support"

	// Reports
	PermReportsManage = "reports:manage"

	// Business Hours
	PermBusinessHoursManage = "business_hours:manage"

	// SLA
	PermSLAManage = "sla:manage"

	// General Settings
	PermServiceEmailsManage   = "service_emails:manage"
	PermGeneralSettingsManage = "general_settings:manage"

	// Notification Settings
	PermNotificationSettingsManage = "notification_settings:manage"

	// OpenID Connect SSO
	PermOIDCManage = "oidc:manage"

	// AI
	PermAIManage         = "ai:manage"
	PermAILearningManage = "ai_learning:manage"

	// Help center
	PermHelpCenterManage = "help_center:manage"

	// Contacts
	PermContactsReadAll = "contacts:read_all"
	PermContactsRead    = "contacts:read"
	PermContactsWrite   = "contacts:write"
	PermContactsBlock   = "contacts:block"
	PermContactsDelete  = "contacts:delete"
	PermContactsExport  = "contacts:export"

	// Contact Notes
	PermContactNotesRead   = "contact_notes:read"
	PermContactNotesWrite  = "contact_notes:write"
	PermContactNotesDelete = "contact_notes:delete"

	// Custom attributes
	PermCustomAttributesManage = "custom_attributes:manage"

	// Activity log
	PermActivityLogsManage = "activity_logs:manage"
)

var validPermissions = map[string]struct{}{
	PermServiceEmailsManage:             {},
	PermAILearningManage:                {},
	PermConversationsReadAll:            {},
	PermConversationsReadUnassigned:     {},
	PermConversationsReadAssigned:       {},
	PermConversationsReadTeamInbox:      {},
	PermConversationsReadTeamAll:        {},
	PermConversationsRead:               {},
	PermConversationsUpdateUserAssignee: {},
	PermConversationsUpdateTeamAssignee: {},
	PermConversationsUpdatePriority:     {},
	PermConversationsUpdateStatus:       {},
	PermConversationsUpdateTags:         {},
	PermConversationsAddSystemTags:      {},
	PermConversationWrite:               {},
	PermMessagesRead:                    {},
	PermMessagesWrite:                   {},
	PermMessagesWriteSpamDetectionNote:  {},
	PermMessagesWriteAsContact:          {},
	PermViewManage:                      {},
	PermSharedViewsManage:               {},
	PermStatusManage:                    {},
	PermTagsManage:                      {},
	PermMacrosManage:                    {},
	PermUsersManage:                     {},
	PermTeamsManage:                     {},
	PermAutomationsManage:               {},
	PermInboxesManage:                   {},
	PermRolesManage:                     {},
	PermTemplatesManage:                 {},
	PermMessageTemplatesSupport:         {},
	PermReportsManage:                   {},
	PermBusinessHoursManage:             {},
	PermSLAManage:                       {},
	PermGeneralSettingsManage:           {},
	PermNotificationSettingsManage:      {},
	PermOIDCManage:                      {},
	PermAIManage:                        {},
	PermHelpCenterManage:                {},
	PermCustomAttributesManage:          {},
	PermContactsReadAll:                 {},
	PermContactsRead:                    {},
	PermContactsWrite:                   {},
	PermContactsBlock:                   {},
	PermContactsDelete:                  {},
	PermContactsExport:                  {},
	PermContactNotesRead:                {},
	PermContactNotesWrite:               {},
	PermContactNotesDelete:              {},
	PermActivityLogsManage:              {},
	PermWebhooksManage:                  {},
	PermContextLinksManage:              {},
}

// PermissionExists returns true if the permission exists else false
func PermissionExists(permission string) bool {
	_, exists := validPermissions[permission]
	return exists
}
