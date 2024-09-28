package main

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// initHandlers initializes the HTTP routes and handlers for the application.
func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	// Authentication.
	g.POST("/api/login", handleLogin)
	g.GET("/api/logout", sess(handleLogout))
	g.GET("/api/oidc/{id}/login", handleOIDCLogin)
	g.GET("/api/oidc/finish", handleOIDCCallback)

	// Health check.
	g.GET("/health", handleHealthCheck)

	// Serve uploaded files.
	g.GET("/uploads/{all:*}", sess(handleServeUploadedFiles))

	// Settings.
	g.GET("/api/settings/general", handleGetGeneralSettings)
	g.PUT("/api/settings/general", perm(handleUpdateGeneralSettings, "settings_general", "write"))
	g.GET("/api/settings/notifications/email", perm(handleGetEmailNotificationSettings, "settings_notifications", "read"))
	g.PUT("/api/settings/notifications/email", perm(handleUpdateEmailNotificationSettings, "settings_notifications", "write"))

	// OpenID.
	g.GET("/api/oidc", handleGetAllOIDC)
	g.GET("/api/oidc/{id}", perm(handleGetOIDC, "oidc", "read"))
	g.POST("/api/oidc", perm(handleCreateOIDC, "oidc", "write"))
	g.PUT("/api/oidc/{id}", perm(handleUpdateOIDC, "oidc", "write"))
	g.DELETE("/api/oidc/{id}", perm(handleDeleteOIDC, "oidc", "delete"))

	// Conversation and message.
	g.GET("/api/conversations/all", perm(handleGetAllConversations, "conversations", "read_all"))
	g.GET("/api/conversations/team", perm(handleGetTeamConversations, "conversations", "read_team"))
	g.GET("/api/conversations/assigned", perm(handleGetAssignedConversations, "conversations", "read_assigned"))

	g.PUT("/api/conversations/{uuid}/assignee/user", perm(handleUpdateConversationUserAssignee, "conversations", "update_user_assignee"))
	g.PUT("/api/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee, "conversations", "update_team_assignee"))
	g.PUT("/api/conversations/{uuid}/priority", perm(handleUpdateConversationPriority, "conversations", "update_priority"))
	g.PUT("/api/conversations/{uuid}/status", perm(handleUpdateConversationStatus, "conversations", "update_status"))

	g.GET("/api/conversations/{uuid}", perm(handleGetConversation, "conversations", "read"))
	g.GET("/api/conversations/{uuid}/participants", perm(handleGetConversationParticipants, "conversations", "read"))
	g.PUT("/api/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen, "conversations", "read"))
	g.POST("/api/conversations/{uuid}/tags", perm(handleAddConversationTags, "conversations", "update_tags"))
	g.GET("/api/conversations/{uuid}/messages", perm(handleGetMessages, "messages", "read"))
	g.POST("/api/conversations/{cuuid}/messages", perm(handleSendMessage, "messages", "write"))
	g.PUT("/api/conversations/{cuuid}/messages/{uuid}/retry", perm(handleRetryMessage, "messages", "write"))
	g.GET("/api/conversations/{cuuid}/messages/{uuid}", perm(handleGetMessage, "messages", "read"))

	// Status and priority.
	g.GET("/api/statuses", sess(handleGetStatuses))
	g.POST("/api/statuses", perm(handleCreateStatus, "status", "write"))
	g.PUT("/api/statuses/{id}", perm(handleUpdateStatus, "status", "write"))
	g.DELETE("/api/statuses/{id}", perm(handleDeleteStatus, "status", "delete"))
	g.GET("/api/priorities", sess(handleGetPriorities))

	// Tag.
	g.GET("/api/tags", sess(handleGetTags))
	g.POST("/api/tags", perm(handleCreateTag, "tags", "write"))
	g.PUT("/api/tags/{id}", perm(handleUpdateTag, "tags", "write"))
	g.DELETE("/api/tags/{id}", perm(handleDeleteTag, "tags", "delete"))

	// Media.
	g.POST("/api/media", sess(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", sess(handleGetCannedResponses))
	g.POST("/api/canned-responses", perm(handleCreateCannedResponse, "canned_responses", "write"))
	g.PUT("/api/canned-responses/{id}", perm(handleUpdateCannedResponse, "canned_responses", "write"))
	g.DELETE("/api/canned-responses/{id}", perm(handleDeleteCannedResponse, "canned_responses", "delete"))

	// User.
	g.GET("/api/users/me", sess(handleGetCurrentUser))
	g.PUT("/api/users/me", sess(handleUpdateCurrentUser))
	g.DELETE("/api/users/me/avatar", sess(handleDeleteAvatar))
	g.GET("/api/users", perm(handleGetUsers, "users", "read"))
	g.GET("/api/users/{id}", perm(handleGetUser, "users", "read"))
	g.POST("/api/users", perm(handleCreateUser, "users", "write"))
	g.PUT("/api/users/{id}", perm(handleUpdateUser, "users", "write"))

	// Team.
	g.GET("/api/teams", perm(handleGetTeams, "teams", "read"))
	g.GET("/api/teams/{id}", perm(handleGetTeam, "teams", "read"))
	g.PUT("/api/teams/{id}", perm(handleUpdateTeam, "teams", "write"))
	g.POST("/api/teams", perm(handleCreateTeam, "teams", "write"))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Automation.
	g.GET("/api/automation/rules", perm(handleGetAutomationRules, "automations", "read"))
	g.GET("/api/automation/rules/{id}", perm(handleGetAutomationRule, "automations", "read"))
	g.POST("/api/automation/rules", perm(handleCreateAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}/toggle", perm(handleToggleAutomationRule, "automations", "write"))
	g.PUT("/api/automation/rules/{id}", perm(handleUpdateAutomationRule, "automations", "write"))
	g.DELETE("/api/automation/rules/{id}", perm(handleDeleteAutomationRule, "automations", "delete"))

	// Inbox.
	g.GET("/api/inboxes", perm(handleGetInboxes, "inboxes", "read"))
	g.GET("/api/inboxes/{id}", perm(handleGetInbox, "inboxes", "read"))
	g.POST("/api/inboxes", perm(handleCreateInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes", "write"))
	g.PUT("/api/inboxes/{id}", perm(handleUpdateInbox, "inboxes", "write"))
	g.DELETE("/api/inboxes/{id}", perm(handleDeleteInbox, "inboxes", "delete"))

	// Role.
	g.GET("/api/roles", perm(handleGetRoles, "roles", "read"))
	g.GET("/api/roles/{id}", perm(handleGetRole, "roles", "read"))
	g.POST("/api/roles", perm(handleCreateRole, "roles", "write"))
	g.PUT("/api/roles/{id}", perm(handleUpdateRole, "roles", "write"))
	g.DELETE("/api/roles/{id}", perm(handleDeleteRole, "roles", "delete"))

	// Dashboard.
	g.GET("/api/dashboard/global/counts", perm(handleDashboardCounts, "dashboard_global", "read"))
	g.GET("/api/dashboard/global/charts", perm(handleDashboardCharts, "dashboard_global", "read"))

	// Template.
	g.GET("/api/templates", perm(handleGetTemplates, "templates", "read"))
	g.GET("/api/templates/{id}", perm(handleGetTemplate, "templates", "read"))
	g.POST("/api/templates", perm(handleCreateTemplate, "templates", "read"))
	g.PUT("/api/templates/{id}", perm(handleUpdateTemplate, "templates", "read"))

	// WebSocket.
	g.GET("/api/ws", sess(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Frontend pages.
	g.GET("/", sess(noAuthPage(serveIndexPage)))
	g.GET("/dashboard", sess(authPage(serveIndexPage)))
	g.GET("/conversations", sess(authPage(serveIndexPage)))
	g.GET("/conversations/{all:*}", sess(authPage(serveIndexPage)))
	g.GET("/account/profile", sess(authPage(serveIndexPage)))
	g.GET("/assets/{all:*}", serveStaticFiles)
}

// handleServeUploadedFiles serves uploaded files from the local filesystem or S3.
func handleServeUploadedFiles(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Extract the file name from the URL path.
	_, fileName := filepath.Split(string(r.RequestCtx.URI().Path()))

	// Remove the "thumb_" prefix.
	fileName = strings.TrimPrefix(fileName, "thumb_")

	// Fetch media metadata from the database.
	media, err := app.media.GetByFilename(fileName)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if the user has permission to access the linked model.
	if media.Model.String == "messages" {
		allowed, err := app.authz.Enforce(user, media.Model.String, "read")
		if err != nil {
			return r.SendErrorEnvelope(http.StatusInternalServerError, "Error checking permissions", nil, envelope.GeneralError)
		}
		if !allowed {
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Permission denied", nil, envelope.PermissionError)
		}

		// Validate access to the related conversation.
		conversation, err := app.conversation.GetConversationByMessageID(media.ModelID.Int)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		_, err = enforceConversationAccess(app, conversation.UUID, user)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	var uploadProvider = ko.String("upload.provider")
	if uploadProvider == "localfs" {
		fasthttp.ServeFile(r.RequestCtx, filepath.Join(ko.String("upload.localfs.upload_path"), fileName))
	}
	if uploadProvider == "s3" {
		s3URL := app.media.GetURL(fileName)
		r.RequestCtx.Redirect(s3URL, http.StatusFound)
		return nil
	}
	return nil
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
