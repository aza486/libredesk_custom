package main

import (
	"mime"
	"net/http"
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
	g.GET("/uploads/{uuid}", reqAuth(handleServeMedia))

	// Settings.
	g.GET("/api/settings/general", handleGetGeneralSettings)
	g.PUT("/api/settings/general", reqAuthAndPerm(handleUpdateGeneralSettings, "settings_general", "write"))
	g.GET("/api/settings/notifications/email", reqAuthAndPerm(handleGetEmailNotificationSettings, "settings_notifications", "read"))
	g.PUT("/api/settings/notifications/email", reqAuthAndPerm(handleUpdateEmailNotificationSettings, "settings_notifications", "write"))

	// OpenID SSO.
	g.GET("/api/oidc", handleGetAllOIDC)
	g.GET("/api/oidc/{id}", reqAuthAndPerm(handleGetOIDC, "oidc", "read"))
	g.POST("/api/oidc", reqAuthAndPerm(handleCreateOIDC, "oidc", "write"))
	g.PUT("/api/oidc/{id}", reqAuthAndPerm(handleUpdateOIDC, "oidc", "write"))
	g.DELETE("/api/oidc/{id}", reqAuthAndPerm(handleDeleteOIDC, "oidc", "delete"))

	// Conversation and message.
	g.GET("/api/conversations/all", reqAuthAndPerm(handleGetAllConversations, "conversations", "read_all"))
	g.GET("/api/conversations/unassigned", reqAuthAndPerm(handleGetUnassignedConversations, "conversations", "read_unassigned"))
	g.GET("/api/conversations/assigned", reqAuthAndPerm(handleGetAssignedConversations, "conversations", "read_assigned"))
	g.GET("/api/conversations/{uuid}", reqAuthAndPerm(handleGetConversation, "conversations", "read"))
	g.GET("/api/conversations/{uuid}/participants", reqAuthAndPerm(handleGetConversationParticipants, "conversations", "read"))
	g.PUT("/api/conversations/{uuid}/assignee/user", reqAuthAndPerm(handleUpdateConversationUserAssignee, "conversations", "update_user_assignee"))
	g.PUT("/api/conversations/{uuid}/assignee/team", reqAuthAndPerm(handleUpdateTeamAssignee, "conversations", "update_team_assignee"))
	g.PUT("/api/conversations/{uuid}/priority", reqAuthAndPerm(handleUpdateConversationPriority, "conversations", "update_priority"))
	g.PUT("/api/conversations/{uuid}/status", reqAuthAndPerm(handleUpdateConversationStatus, "conversations", "update_status"))
	g.PUT("/api/conversations/{uuid}/last-seen", reqAuthAndPerm(handleUpdateConversationAssigneeLastSeen, "conversations", "read"))
	g.POST("/api/conversations/{uuid}/tags", reqAuthAndPerm(handleAddConversationTags, "conversations", "update_tags"))
	g.POST("/api/conversations/{cuuid}/messages", reqAuthAndPerm(handleSendMessage, "messages", "write"))
	g.GET("/api/conversations/{uuid}/messages", reqAuthAndPerm(handleGetMessages, "messages", "read"))
	g.PUT("/api/conversations/{cuuid}/messages/{uuid}/retry", reqAuthAndPerm(handleRetryMessage, "messages", "write"))
	g.GET("/api/conversations/{cuuid}/messages/{uuid}", reqAuthAndPerm(handleGetMessage, "messages", "read"))

	// Status and priority.
	g.GET("/api/statuses", reqAuth(handleGetStatuses))
	g.POST("/api/statuses", reqAuthAndPerm(handleCreateStatus, "status", "write"))
	g.PUT("/api/statuses/{id}", reqAuthAndPerm(handleUpdateStatus, "status", "write"))
	g.DELETE("/api/statuses/{id}", reqAuthAndPerm(handleDeleteStatus, "status", "delete"))
	g.GET("/api/priorities", reqAuth(handleGetPriorities))

	// Tag.
	g.GET("/api/tags", reqAuth(handleGetTags))
	g.POST("/api/tags", reqAuthAndPerm(handleCreateTag, "tags", "write"))
	g.PUT("/api/tags/{id}", reqAuthAndPerm(handleUpdateTag, "tags", "write"))
	g.DELETE("/api/tags/{id}", reqAuthAndPerm(handleDeleteTag, "tags", "delete"))

	// Media.
	g.POST("/api/media", reqAuth(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", reqAuth(handleGetCannedResponses))
	g.POST("/api/canned-responses", reqAuthAndPerm(handleCreateCannedResponse, "canned_responses", "write"))
	g.PUT("/api/canned-responses/{id}", reqAuthAndPerm(handleUpdateCannedResponse, "canned_responses", "write"))
	g.DELETE("/api/canned-responses/{id}", reqAuthAndPerm(handleDeleteCannedResponse, "canned_responses", "delete"))

	// User.
	g.GET("/api/users/me", reqAuth(handleGetCurrentUser))
	g.PUT("/api/users/me", reqAuth(handleUpdateCurrentUser))
	g.DELETE("/api/users/me/avatar", reqAuth(handleDeleteAvatar))
	g.GET("/api/users/compact", reqAuth(handleGetUsersCompact))
	g.GET("/api/users", reqAuthAndPerm(handleGetUsers, "users", "read"))
	g.GET("/api/users/{id}", reqAuthAndPerm(handleGetUser, "users", "read"))
	g.POST("/api/users", reqAuthAndPerm(handleCreateUser, "users", "write"))
	g.PUT("/api/users/{id}", reqAuthAndPerm(handleUpdateUser, "users", "write"))

	// Team.
	g.GET("/api/teams/compact", reqAuth(handleGetTeamsCompact))
	g.GET("/api/teams", reqAuthAndPerm(handleGetTeams, "teams", "read"))
	g.GET("/api/teams/{id}", reqAuthAndPerm(handleGetTeam, "teams", "read"))
	g.PUT("/api/teams/{id}", reqAuthAndPerm(handleUpdateTeam, "teams", "write"))
	g.POST("/api/teams", reqAuthAndPerm(handleCreateTeam, "teams", "write"))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Automation.
	g.GET("/api/automation/rules", reqAuthAndPerm(handleGetAutomationRules, "automations", "read"))
	g.GET("/api/automation/rules/{id}", reqAuthAndPerm(handleGetAutomationRule, "automations", "read"))
	g.POST("/api/automation/rules", reqAuthAndPerm(handleCreateAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}/toggle", reqAuthAndPerm(handleToggleAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}", reqAuthAndPerm(handleUpdateAutomationRule, "automations", "write"))
	g.DELETE("/api/automation/rules/{id}", reqAuthAndPerm(handleDeleteAutomationRule, "automations", "delete"))

	// Inbox.
	g.GET("/api/inboxes", reqAuthAndPerm(handleGetInboxes, "inboxes", "read"))
	g.GET("/api/inboxes/{id}", reqAuthAndPerm(handleGetInbox, "inboxes", "read"))
	g.POST("/api/inboxes", reqAuthAndPerm(handleCreateInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}/toggle", reqAuthAndPerm(handleToggleInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}", reqAuthAndPerm(handleUpdateInbox, "inboxes", "write"))
	g.DELETE("/api/inboxes/{id}", reqAuthAndPerm(handleDeleteInbox, "inboxes", "delete"))

	// Role.
	g.GET("/api/roles", reqAuthAndPerm(handleGetRoles, "roles", "read"))
	g.GET("/api/roles/{id}", reqAuthAndPerm(handleGetRole, "roles", "read"))
	g.POST("/api/roles", reqAuthAndPerm(handleCreateRole, "roles", "write"))
	g.PUT("/api/roles/{id}", reqAuthAndPerm(handleUpdateRole, "roles", "write"))
	g.DELETE("/api/roles/{id}", reqAuthAndPerm(handleDeleteRole, "roles", "delete"))

	// Dashboard.
	g.GET("/api/dashboard/global/counts", reqAuthAndPerm(handleDashboardCounts, "dashboard_global", "read"))
	g.GET("/api/dashboard/global/charts", reqAuthAndPerm(handleDashboardCharts, "dashboard_global", "read"))

	// Template.
	g.GET("/api/templates", reqAuthAndPerm(handleGetTemplates, "templates", "read"))
	g.GET("/api/templates/{id}", reqAuthAndPerm(handleGetTemplate, "templates", "read"))
	g.POST("/api/templates", reqAuthAndPerm(handleCreateTemplate, "templates", "write"))
	g.PUT("/api/templates/{id}", reqAuthAndPerm(handleUpdateTemplate, "templates", "write"))
	g.DELETE("/api/templates/{id}", reqAuthAndPerm(handleDeleteTemplate, "templates", "delete"))

	// WebSocket.
	g.GET("/api/ws", reqAuth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Frontend pages.
	g.GET("/", notAuthenticatedPage(serveIndexPage))
	g.GET("/dashboard", authenticatedPage(serveIndexPage))
	g.GET("/conversations", authenticatedPage(serveIndexPage))
	g.GET("/conversations/{all:*}", authenticatedPage(serveIndexPage))
	g.GET("/account/profile", authenticatedPage(serveIndexPage))
	g.GET("/admin/{all:*}", authenticatedPage(serveIndexPage))
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
	file, err := app.fs.Get("/frontend/dist/index.html")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "Page not found", nil, "InputException")
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.SetBody(file.ReadBytes())

	// Set csrf cookie if not already set.
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
	finalPath := filepath.Join("frontend/dist", filePath)
	file, err := app.fs.Get(finalPath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "File not found", nil, "InputException")
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
