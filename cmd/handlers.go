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
	g.GET("/api/logout", authMiddleware(handleLogout, "", ""))
	g.GET("/api/oidc/{id}/login", handleOIDCLogin)
	g.GET("/api/oidc/finish", handleOIDCCallback)

	// Health check.
	g.GET("/health", handleHealthCheck)

	// Serve media files.
	g.GET("/uploads/{uuid}", authMiddleware(handleServeMedia, "", ""))

	// Settings.
	g.GET("/api/settings/general", handleGetGeneralSettings)
	g.PUT("/api/settings/general", authMiddleware(handleUpdateGeneralSettings, "settings_general", "write"))
	g.GET("/api/settings/notifications/email", authMiddleware(handleGetEmailNotificationSettings, "settings_notifications", "read"))
	g.PUT("/api/settings/notifications/email", authMiddleware(handleUpdateEmailNotificationSettings, "settings_notifications", "write"))

	// OpenID SSO.
	g.GET("/api/oidc", handleGetAllOIDC)
	g.GET("/api/oidc/{id}", authMiddleware(handleGetOIDC, "oidc", "read"))
	g.POST("/api/oidc", authMiddleware(handleCreateOIDC, "oidc", "write"))
	g.PUT("/api/oidc/{id}", authMiddleware(handleUpdateOIDC, "oidc", "write"))
	g.DELETE("/api/oidc/{id}", authMiddleware(handleDeleteOIDC, "oidc", "delete"))

	// Conversation and message.
	g.GET("/api/conversations/all", authMiddleware(handleGetAllConversations, "conversations", "read_all"))
	g.GET("/api/conversations/unassigned", authMiddleware(handleGetUnassignedConversations, "conversations", "read_unassigned"))
	g.GET("/api/conversations/assigned", authMiddleware(handleGetAssignedConversations, "conversations", "read_assigned"))
	g.GET("/api/conversations/{uuid}", authMiddleware(handleGetConversation, "conversations", "read"))
	g.GET("/api/conversations/{uuid}/participants", authMiddleware(handleGetConversationParticipants, "conversations", "read"))
	g.PUT("/api/conversations/{uuid}/assignee/user", authMiddleware(handleUpdateConversationUserAssignee, "conversations", "update_user_assignee"))
	g.PUT("/api/conversations/{uuid}/assignee/team", authMiddleware(handleUpdateTeamAssignee, "conversations", "update_team_assignee"))
	g.PUT("/api/conversations/{uuid}/priority", authMiddleware(handleUpdateConversationPriority, "conversations", "update_priority"))
	g.PUT("/api/conversations/{uuid}/status", authMiddleware(handleUpdateConversationStatus, "conversations", "update_status"))
	g.PUT("/api/conversations/{uuid}/last-seen", authMiddleware(handleUpdateConversationAssigneeLastSeen, "conversations", "read"))
	g.POST("/api/conversations/{uuid}/tags", authMiddleware(handleAddConversationTags, "conversations", "update_tags"))
	g.POST("/api/conversations/{cuuid}/messages", authMiddleware(handleSendMessage, "messages", "write"))
	g.GET("/api/conversations/{uuid}/messages", authMiddleware(handleGetMessages, "messages", "read"))
	g.PUT("/api/conversations/{cuuid}/messages/{uuid}/retry", authMiddleware(handleRetryMessage, "messages", "write"))
	g.GET("/api/conversations/{cuuid}/messages/{uuid}", authMiddleware(handleGetMessage, "messages", "read"))

	// Status and priority.
	g.GET("/api/statuses", authMiddleware(handleGetStatuses, "", ""))
	g.POST("/api/statuses", authMiddleware(handleCreateStatus, "status", "write"))
	g.PUT("/api/statuses/{id}", authMiddleware(handleUpdateStatus, "status", "write"))
	g.DELETE("/api/statuses/{id}", authMiddleware(handleDeleteStatus, "status", "delete"))
	g.GET("/api/priorities", authMiddleware(handleGetPriorities, "", ""))

	// Tag.
	g.GET("/api/tags", authMiddleware(handleGetTags, "", ""))
	g.POST("/api/tags", authMiddleware(handleCreateTag, "tags", "write"))
	g.PUT("/api/tags/{id}", authMiddleware(handleUpdateTag, "tags", "write"))
	g.DELETE("/api/tags/{id}", authMiddleware(handleDeleteTag, "tags", "delete"))

	// Media.
	g.POST("/api/media", authMiddleware(handleMediaUpload, "", ""))

	// Canned response.
	g.GET("/api/canned-responses", authMiddleware(handleGetCannedResponses, "", ""))
	g.POST("/api/canned-responses", authMiddleware(handleCreateCannedResponse, "canned_responses", "write"))
	g.PUT("/api/canned-responses/{id}", authMiddleware(handleUpdateCannedResponse, "canned_responses", "write"))
	g.DELETE("/api/canned-responses/{id}", authMiddleware(handleDeleteCannedResponse, "canned_responses", "delete"))

	// User.
	g.GET("/api/users/me", authMiddleware(handleGetCurrentUser, "", ""))
	g.PUT("/api/users/me", authMiddleware(handleUpdateCurrentUser, "", ""))
	g.DELETE("/api/users/me/avatar", authMiddleware(handleDeleteAvatar, "", ""))
	g.GET("/api/users/compact", authMiddleware(handleGetUsersCompact, "", ""))
	g.GET("/api/users", authMiddleware(handleGetUsers, "users", "read"))
	g.GET("/api/users/{id}", authMiddleware(handleGetUser, "users", "read"))
	g.POST("/api/users", authMiddleware(handleCreateUser, "users", "write"))
	g.PUT("/api/users/{id}", authMiddleware(handleUpdateUser, "users", "write"))

	// Team.
	g.GET("/api/teams/compact", authMiddleware(handleGetTeamsCompact, "", ""))
	g.GET("/api/teams", authMiddleware(handleGetTeams, "teams", "read"))
	g.GET("/api/teams/{id}", authMiddleware(handleGetTeam, "teams", "read"))
	g.PUT("/api/teams/{id}", authMiddleware(handleUpdateTeam, "teams", "write"))
	g.POST("/api/teams", authMiddleware(handleCreateTeam, "teams", "write"))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Automation.
	g.GET("/api/automation/rules", authMiddleware(handleGetAutomationRules, "automations", "read"))
	g.GET("/api/automation/rules/{id}", authMiddleware(handleGetAutomationRule, "automations", "read"))
	g.POST("/api/automation/rules", authMiddleware(handleCreateAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}/toggle", authMiddleware(handleToggleAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}", authMiddleware(handleUpdateAutomationRule, "automations", "write"))
	g.DELETE("/api/automation/rules/{id}", authMiddleware(handleDeleteAutomationRule, "automations", "delete"))

	// Inbox.
	g.GET("/api/inboxes", authMiddleware(handleGetInboxes, "inboxes", "read"))
	g.GET("/api/inboxes/{id}", authMiddleware(handleGetInbox, "inboxes", "read"))
	g.POST("/api/inboxes", authMiddleware(handleCreateInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}/toggle", authMiddleware(handleToggleInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}", authMiddleware(handleUpdateInbox, "inboxes", "write"))
	g.DELETE("/api/inboxes/{id}", authMiddleware(handleDeleteInbox, "inboxes", "delete"))

	// Role.
	g.GET("/api/roles", authMiddleware(handleGetRoles, "roles", "read"))
	g.GET("/api/roles/{id}", authMiddleware(handleGetRole, "roles", "read"))
	g.POST("/api/roles", authMiddleware(handleCreateRole, "roles", "write"))
	g.PUT("/api/roles/{id}", authMiddleware(handleUpdateRole, "roles", "write"))
	g.DELETE("/api/roles/{id}", authMiddleware(handleDeleteRole, "roles", "delete"))

	// Dashboard.
	g.GET("/api/dashboard/global/counts", authMiddleware(handleDashboardCounts, "dashboard_global", "read"))
	g.GET("/api/dashboard/global/charts", authMiddleware(handleDashboardCharts, "dashboard_global", "read"))

	// Template.
	g.GET("/api/templates", authMiddleware(handleGetTemplates, "templates", "read"))
	g.GET("/api/templates/{id}", authMiddleware(handleGetTemplate, "templates", "read"))
	g.POST("/api/templates", authMiddleware(handleCreateTemplate, "templates", "write"))
	g.PUT("/api/templates/{id}", authMiddleware(handleUpdateTemplate, "templates", "write"))
	g.DELETE("/api/templates/{id}", authMiddleware(handleDeleteTemplate, "templates", "delete"))

	// WebSocket.
	g.GET("/api/ws", authMiddleware(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}, "", ""))

	// Frontend pages.
	g.GET("/", notAuthenticatedPage(serveIndexPage))
	g.GET("/dashboard", authenticatedPage(serveIndexPage))
	g.GET("/conversations", authenticatedPage(serveIndexPage))
	g.GET("/conversations/{all:*}", authenticatedPage(serveIndexPage))
	g.GET("/account/profile", authenticatedPage(serveIndexPage))
	g.GET("/admin/{all:*}", authenticatedPage(serveIndexPage))
	g.GET("/assets/{all:*}", serveStaticFiles)
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
