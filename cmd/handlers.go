package main

import (
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// initHandlers initializes the HTTP routes and handlers for the application.
func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	// Authentication.
	g.POST("/api/login", handleLogin)
	g.GET("/logout", handleLogout)
	g.GET("/api/oidc/{id}/login", handleOIDCLogin)
	g.GET("/api/oidc/finish", handleOIDCCallback)

	// Health check.
	g.GET("/health", handleHealthCheck)

	// Serve media files.
	g.GET("/uploads/{uuid}", auth(handleServeMedia))

	// Settings.
	g.GET("/api/settings/general", handleGetGeneralSettings)
	g.PUT("/api/settings/general", authPerm(handleUpdateGeneralSettings, "settings_general", "write"))
	g.GET("/api/settings/notifications/email", authPerm(handleGetEmailNotificationSettings, "settings_notifications", "read"))
	g.PUT("/api/settings/notifications/email", authPerm(handleUpdateEmailNotificationSettings, "settings_notifications", "write"))

	// OpenID SSO.
	g.GET("/api/oidc", handleGetAllOIDC)
	g.GET("/api/oidc/{id}", authPerm(handleGetOIDC, "oidc", "read"))
	g.POST("/api/oidc", authPerm(handleCreateOIDC, "oidc", "write"))
	g.PUT("/api/oidc/{id}", authPerm(handleUpdateOIDC, "oidc", "write"))
	g.DELETE("/api/oidc/{id}", authPerm(handleDeleteOIDC, "oidc", "delete"))

	// Conversation and message.
	g.GET("/api/conversations/all", authPerm(handleGetAllConversations, "conversations", "read_all"))
	g.GET("/api/conversations/unassigned", authPerm(handleGetUnassignedConversations, "conversations", "read_unassigned"))
	g.GET("/api/conversations/assigned", authPerm(handleGetAssignedConversations, "conversations", "read_assigned"))
	g.GET("/api/conversations/{uuid}", authPerm(handleGetConversation, "conversations", "read"))
	g.GET("/api/conversations/{uuid}/participants", authPerm(handleGetConversationParticipants, "conversations", "read"))
	g.PUT("/api/conversations/{uuid}/assignee/user", authPerm(handleUpdateConversationUserAssignee, "conversations", "update_user_assignee"))
	g.PUT("/api/conversations/{uuid}/assignee/team", authPerm(handleUpdateTeamAssignee, "conversations", "update_team_assignee"))
	g.PUT("/api/conversations/{uuid}/priority", authPerm(handleUpdateConversationPriority, "conversations", "update_priority"))
	g.PUT("/api/conversations/{uuid}/status", authPerm(handleUpdateConversationStatus, "conversations", "update_status"))
	g.PUT("/api/conversations/{uuid}/last-seen", authPerm(handleUpdateConversationAssigneeLastSeen, "conversations", "read"))
	g.POST("/api/conversations/{uuid}/tags", authPerm(handleAddConversationTags, "conversations", "update_tags"))
	g.POST("/api/conversations/{cuuid}/messages", authPerm(handleSendMessage, "messages", "write"))
	g.GET("/api/conversations/{uuid}/messages", authPerm(handleGetMessages, "messages", "read"))
	g.PUT("/api/conversations/{cuuid}/messages/{uuid}/retry", authPerm(handleRetryMessage, "messages", "write"))
	g.GET("/api/conversations/{cuuid}/messages/{uuid}", authPerm(handleGetMessage, "messages", "read"))

	// Status and priority.
	g.GET("/api/statuses", auth(handleGetStatuses))
	g.POST("/api/statuses", authPerm(handleCreateStatus, "status", "write"))
	g.PUT("/api/statuses/{id}", authPerm(handleUpdateStatus, "status", "write"))
	g.DELETE("/api/statuses/{id}", authPerm(handleDeleteStatus, "status", "delete"))
	g.GET("/api/priorities", auth(handleGetPriorities))

	// Tag.
	g.GET("/api/tags", auth(handleGetTags))
	g.POST("/api/tags", authPerm(handleCreateTag, "tags", "write"))
	g.PUT("/api/tags/{id}", authPerm(handleUpdateTag, "tags", "write"))
	g.DELETE("/api/tags/{id}", authPerm(handleDeleteTag, "tags", "delete"))

	// Media.
	g.POST("/api/media", auth(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", auth(handleGetCannedResponses))
	g.POST("/api/canned-responses", authPerm(handleCreateCannedResponse, "canned_responses", "write"))
	g.PUT("/api/canned-responses/{id}", authPerm(handleUpdateCannedResponse, "canned_responses", "write"))
	g.DELETE("/api/canned-responses/{id}", authPerm(handleDeleteCannedResponse, "canned_responses", "delete"))

	// User.
	g.GET("/api/users/me", auth(handleGetCurrentUser))
	g.PUT("/api/users/me", auth(handleUpdateCurrentUser))
	g.DELETE("/api/users/me/avatar", auth(handleDeleteAvatar))
	g.GET("/api/users/compact", auth(handleGetUsersCompact))
	g.GET("/api/users", authPerm(handleGetUsers, "users", "read"))
	g.GET("/api/users/{id}", authPerm(handleGetUser, "users", "read"))
	g.POST("/api/users", authPerm(handleCreateUser, "users", "write"))
	g.PUT("/api/users/{id}", authPerm(handleUpdateUser, "users", "write"))

	// Team.
	g.GET("/api/teams/compact", auth(handleGetTeamsCompact))
	g.GET("/api/teams", authPerm(handleGetTeams, "teams", "read"))
	g.GET("/api/teams/{id}", authPerm(handleGetTeam, "teams", "read"))
	g.PUT("/api/teams/{id}", authPerm(handleUpdateTeam, "teams", "write"))
	g.POST("/api/teams", authPerm(handleCreateTeam, "teams", "write"))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Automation.
	g.GET("/api/automation/rules", authPerm(handleGetAutomationRules, "automations", "read"))
	g.GET("/api/automation/rules/{id}", authPerm(handleGetAutomationRule, "automations", "read"))
	g.POST("/api/automation/rules", authPerm(handleCreateAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}/toggle", authPerm(handleToggleAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}", authPerm(handleUpdateAutomationRule, "automations", "write"))
	g.DELETE("/api/automation/rules/{id}", authPerm(handleDeleteAutomationRule, "automations", "delete"))

	// Inbox.
	g.GET("/api/inboxes", authPerm(handleGetInboxes, "inboxes", "read"))
	g.GET("/api/inboxes/{id}", authPerm(handleGetInbox, "inboxes", "read"))
	g.POST("/api/inboxes", authPerm(handleCreateInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}/toggle", authPerm(handleToggleInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}", authPerm(handleUpdateInbox, "inboxes", "write"))
	g.DELETE("/api/inboxes/{id}", authPerm(handleDeleteInbox, "inboxes", "delete"))

	// Role.
	g.GET("/api/roles", authPerm(handleGetRoles, "roles", "read"))
	g.GET("/api/roles/{id}", authPerm(handleGetRole, "roles", "read"))
	g.POST("/api/roles", authPerm(handleCreateRole, "roles", "write"))
	g.PUT("/api/roles/{id}", authPerm(handleUpdateRole, "roles", "write"))
	g.DELETE("/api/roles/{id}", authPerm(handleDeleteRole, "roles", "delete"))

	// Dashboard.
	g.GET("/api/dashboard/global/counts", authPerm(handleDashboardCounts, "dashboard_global", "read"))
	g.GET("/api/dashboard/global/charts", authPerm(handleDashboardCharts, "dashboard_global", "read"))

	// Template.
	g.GET("/api/templates", authPerm(handleGetTemplates, "templates", "read"))
	g.GET("/api/templates/{id}", authPerm(handleGetTemplate, "templates", "read"))
	g.POST("/api/templates", authPerm(handleCreateTemplate, "templates", "write"))
	g.PUT("/api/templates/{id}", authPerm(handleUpdateTemplate, "templates", "write"))
	g.DELETE("/api/templates/{id}", authPerm(handleDeleteTemplate, "templates", "delete"))

	// WebSocket.
	g.GET("/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Frontend pages.
	g.GET("/", notAuthPage(serveIndexPage))
	g.GET("/dashboard", authPage(serveIndexPage))
	g.GET("/conversations", authPage(serveIndexPage))
	g.GET("/conversations/{all:*}", authPage(serveIndexPage))
	g.GET("/account/profile", authPage(serveIndexPage))
	g.GET("/admin/{all:*}", authPage(serveIndexPage))
	g.GET("/assets/{all:*}", serveStaticFiles)
	g.GET("/images/{all:*}", serveStaticFiles)
}

// serveIndexPage serves the main index page of the application.
func serveIndexPage(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Prevent caching of the index page.
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")

	// Serve the index.html file from the embedded filesystem.
	file, err := app.fs.Get(path.Join(frontendDir, "index.html"))
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "Page not found", nil, envelope.NotFoundError)
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.SetBody(file.ReadBytes())

	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	return nil
}

// serveStaticFiles serves static assets from the embedded filesystem.
func serveStaticFiles(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Get the requested file path.
	filePath := string(r.RequestCtx.Path())

	// Fetch and serve the file from the embedded filesystem.
	finalPath := filepath.Join(frontendDir, filePath)
	file, err := app.fs.Get(finalPath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "File not found", nil, envelope.NotFoundError)
	}

	// Set the appropriate Content-Type based on the file extension.
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(file.ReadBytes())
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// sendErrorEnvelope sends a standardized error response to the client.
func sendErrorEnvelope(r *fastglue.Request, err error) error {
	e, ok := err.(envelope.Error)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error interface conversion failed", nil, fastglue.ErrorType(envelope.GeneralError))
	}
	return r.SendErrorEnvelope(e.Code, e.Error(), e.Data, fastglue.ErrorType(e.ErrorType))
}

// handleHealthCheck handles the health check endpoint.
func handleHealthCheck(r *fastglue.Request) error {
	return r.SendEnvelope(true)
}
