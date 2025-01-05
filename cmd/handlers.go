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

var (
	slaReqFields = map[string][2]int{"name": {1, 255}, "description": {1, 255}, "first_response_time": {1, 255}, "resolution_time": {1, 255}}
)

// initHandlers initializes the HTTP routes and handlers for the application.
func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	// Authentication.
	g.POST("/api/v1/login", handleLogin)
	g.GET("/logout", handleLogout)
	g.GET("/api/v1/oidc/{id}/login", handleOIDCLogin)
	g.GET("/api/v1/oidc/finish", handleOIDCCallback)

	// Serve media files.
	g.GET("/uploads/{uuid}", auth(handleServeMedia))

	// Settings.
	g.GET("/api/v1/settings/general", handleGetGeneralSettings)
	g.PUT("/api/v1/settings/general", perm(handleUpdateGeneralSettings, "settings_general", "write"))
	g.GET("/api/v1/settings/notifications/email", perm(handleGetEmailNotificationSettings, "settings_notifications", "read"))
	g.PUT("/api/v1/settings/notifications/email", perm(handleUpdateEmailNotificationSettings, "settings_notifications", "write"))

	// OpenID connect single sign-on.
	g.GET("/api/v1/oidc", handleGetAllOIDC)
	g.GET("/api/v1/oidc/{id}", perm(handleGetOIDC, "oidc", "read"))
	g.POST("/api/v1/oidc", perm(handleCreateOIDC, "oidc", "write"))
	g.PUT("/api/v1/oidc/{id}", perm(handleUpdateOIDC, "oidc", "write"))
	g.DELETE("/api/v1/oidc/{id}", perm(handleDeleteOIDC, "oidc", "delete"))

	// All.
	g.GET("/api/v1/conversations/all", perm(handleGetAllConversations, "conversations", "read_all"))
	// Not assigned to any user or team.
	g.GET("/api/v1/conversations/unassigned", perm(handleGetUnassignedConversations, "conversations", "read_unassigned"))
	// Assigned to logged in user.
	g.GET("/api/v1/conversations/assigned", perm(handleGetAssignedConversations, "conversations", "read_assigned"))
	// Unassigned conversations assigned to a team.
	g.GET("/api/v1/teams/{team_id}/conversations/unassigned", perm(handleGetTeamUnassignedConversations, "conversations", "read_assigned"))
	// Filtered by view.
	g.GET("/api/v1/views/{view_id}/conversations", perm(handleGetViewConversations, "conversations", "read"))

	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations", "read"))
	g.GET("/api/v1/conversations/{uuid}/participants", perm(handleGetConversationParticipants, "conversations", "read"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/user", perm(handleUpdateConversationUserAssignee, "conversations", "update_user_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee, "conversations", "update_team_assignee"))
	g.PUT("/api/v1/conversations/{uuid}/priority", perm(handleUpdateConversationPriority, "conversations", "update_priority"))
	g.PUT("/api/v1/conversations/{uuid}/status", perm(handleUpdateConversationStatus, "conversations", "update_status"))
	g.PUT("/api/v1/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen, "conversations", "read"))
	g.POST("/api/v1/conversations/{uuid}/tags", perm(handleAddConversationTags, "conversations", "update_tags"))
	g.POST("/api/v1/conversations/{cuuid}/messages", perm(handleSendMessage, "messages", "write"))
	g.GET("/api/v1/conversations/{uuid}/messages", perm(handleGetMessages, "messages", "read"))
	g.PUT("/api/v1/conversations/{cuuid}/messages/{uuid}/retry", perm(handleRetryMessage, "messages", "write"))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}", perm(handleGetMessage, "messages", "read"))

	// Views.
	g.GET("/api/v1/views/me", auth(handleGetUserViews))
	g.POST("/api/v1/views/me", auth(handleCreateUserView))
	g.PUT("/api/v1/views/me/{id}", auth(handleUpdateUserView))
	g.DELETE("/api/v1/views/me/{id}", auth(handleDeleteUserView))

	// Status and priority.
	g.GET("/api/v1/statuses", auth(handleGetStatuses))
	g.POST("/api/v1/statuses", perm(handleCreateStatus, "status", "write"))
	g.PUT("/api/v1/statuses/{id}", perm(handleUpdateStatus, "status", "write"))
	g.DELETE("/api/v1/statuses/{id}", perm(handleDeleteStatus, "status", "delete"))
	g.GET("/api/v1/priorities", auth(handleGetPriorities))

	// Tag.
	g.GET("/api/v1/tags", auth(handleGetTags))
	g.POST("/api/v1/tags", perm(handleCreateTag, "tags", "write"))
	g.PUT("/api/v1/tags/{id}", perm(handleUpdateTag, "tags", "write"))
	g.DELETE("/api/v1/tags/{id}", perm(handleDeleteTag, "tags", "delete"))

	// Media.
	g.POST("/api/v1/media", auth(handleMediaUpload))

	// Canned response.
	g.GET("/api/v1/canned-responses", auth(handleGetCannedResponses))
	g.POST("/api/v1/canned-responses", perm(handleCreateCannedResponse, "canned_responses", "write"))
	g.PUT("/api/v1/canned-responses/{id}", perm(handleUpdateCannedResponse, "canned_responses", "write"))
	g.DELETE("/api/v1/canned-responses/{id}", perm(handleDeleteCannedResponse, "canned_responses", "delete"))

	// User.
	g.GET("/api/v1/users/me", auth(handleGetCurrentUser))
	g.PUT("/api/v1/users/me", auth(handleUpdateCurrentUser))
	g.GET("/api/v1/users/me/teams", auth(handleGetCurrentUserTeams))
	g.DELETE("/api/v1/users/me/avatar", auth(handleDeleteAvatar))
	g.GET("/api/v1/users/compact", auth(handleGetUsersCompact))
	g.GET("/api/v1/users", perm(handleGetUsers, "users", "read"))
	g.GET("/api/v1/users/{id}", perm(handleGetUser, "users", "read"))
	g.POST("/api/v1/users", perm(handleCreateUser, "users", "write"))
	g.PUT("/api/v1/users/{id}", perm(handleUpdateUser, "users", "write"))
	g.DELETE("/api/v1/users/{id}", perm(handleDeleteUser, "users", "delete"))
	g.POST("/api/v1/users/reset-password", tryAuth(handleResetPassword))
	g.POST("/api/v1/users/set-password", tryAuth(handleSetPassword))

	// Team.
	g.GET("/api/v1/teams/compact", auth(handleGetTeamsCompact))
	g.GET("/api/v1/teams", perm(handleGetTeams, "teams", "read"))
	g.GET("/api/v1/teams/{id}", perm(handleGetTeam, "teams", "read"))
	g.POST("/api/v1/teams", perm(handleCreateTeam, "teams", "write"))
	g.PUT("/api/v1/teams/{id}", perm(handleUpdateTeam, "teams", "write"))
	g.DELETE("/api/v1/teams/{id}", perm(handleDeleteTeam, "teams", "delete"))

	// i18n.
	g.GET("/api/v1/lang/{lang}", handleGetI18nLang)

	// Automation.
	g.GET("/api/v1/automation/rules", perm(handleGetAutomationRules, "automations", "read"))
	g.GET("/api/v1/automation/rules/{id}", perm(handleGetAutomationRule, "automations", "read"))
	g.POST("/api/v1/automation/rules", perm(handleCreateAutomationRule, "automations", "write"))
	g.PUT("/api/v1/automation/rules/{id}/toggle", perm(handleToggleAutomationRule, "automations", "write"))
	g.PUT("/api/v1/automation/rules/{id}", perm(handleUpdateAutomationRule, "automations", "write"))
	g.DELETE("/api/v1/automation/rules/{id}", perm(handleDeleteAutomationRule, "automations", "delete"))

	// Inbox.
	g.GET("/api/v1/inboxes", perm(handleGetInboxes, "inboxes", "read"))
	g.GET("/api/v1/inboxes/{id}", perm(handleGetInbox, "inboxes", "read"))
	g.POST("/api/v1/inboxes", perm(handleCreateInbox, "inboxes", "write"))
	g.PUT("/api/v1/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes", "write"))
	g.PUT("/api/v1/inboxes/{id}", perm(handleUpdateInbox, "inboxes", "write"))
	g.DELETE("/api/v1/inboxes/{id}", perm(handleDeleteInbox, "inboxes", "delete"))

	// Role.
	g.GET("/api/v1/roles", perm(handleGetRoles, "roles", "read"))
	g.GET("/api/v1/roles/{id}", perm(handleGetRole, "roles", "read"))
	g.POST("/api/v1/roles", perm(handleCreateRole, "roles", "write"))
	g.PUT("/api/v1/roles/{id}", perm(handleUpdateRole, "roles", "write"))
	g.DELETE("/api/v1/roles/{id}", perm(handleDeleteRole, "roles", "delete"))

	// Dashboard.
	g.GET("/api/v1/dashboard/global/counts", perm(handleDashboardCounts, "dashboard_global", "read"))
	g.GET("/api/v1/dashboard/global/charts", perm(handleDashboardCharts, "dashboard_global", "read"))

	// Template.
	g.GET("/api/v1/templates", perm(handleGetTemplates, "templates", "read"))
	g.GET("/api/v1/templates/{id}", perm(handleGetTemplate, "templates", "read"))
	g.POST("/api/v1/templates", perm(handleCreateTemplate, "templates", "write"))
	g.PUT("/api/v1/templates/{id}", perm(handleUpdateTemplate, "templates", "write"))
	g.DELETE("/api/v1/templates/{id}", perm(handleDeleteTemplate, "templates", "delete"))

	// Business hours.
	g.GET("/api/v1/business-hours", auth(handleGetBusinessHours))
	g.GET("/api/v1/business-hours/{id}", auth(handleGetBusinessHour))
	g.POST("/api/v1/business-hours", auth(handleCreateBusinessHours))
	g.PUT("/api/v1/business-hours/{id}", auth(handleUpdateBusinessHours))
	g.DELETE("/api/v1/business-hours/{id}", auth(handleDeleteBusinessHour))

	// SLA.
	g.GET("/api/v1/sla", auth(handleGetSLAs))
	g.GET("/api/v1/sla/{id}", auth(handleGetSLA))
	g.POST("/api/v1/sla", auth(fastglue.ReqLenRangeParams(handleCreateSLA, slaReqFields)))
	g.PUT("/api/v1/sla/{id}", auth(fastglue.ReqLenRangeParams(handleUpdateSLA, slaReqFields)))
	g.DELETE("/api/v1/sla/{id}", auth(handleDeleteSLA))

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
	g.GET("/reset-password", notAuthPage(serveIndexPage))
	g.GET("/set-password", notAuthPage(serveIndexPage))
	g.GET("/assets/{all:*}", serveFrontendStaticFiles)
	g.GET("/images/{all:*}", serveFrontendStaticFiles)
	g.GET("/static/public/{all:*}", serveStaticFiles)

	// Public pages.
	g.GET("/csat/{uuid}", handleShowCSAT)
	g.POST("/csat/{uuid}", fastglue.ReqLenRangeParams(handleUpdateCSATResponse, map[string][2]int{"feedback": {1, 1000}}))

	// Health check.
	g.GET("/health", handleHealthCheck)
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

	file, err := app.fs.Get(filePath)
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

// serveFrontendStaticFiles serves static assets from the embedded filesystem.
func serveFrontendStaticFiles(r *fastglue.Request) error {
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
