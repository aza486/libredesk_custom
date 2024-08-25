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

func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	g.POST("/api/login", handleLogin)
	g.GET("/api/logout", handleLogout)
	g.GET("/api/oidc/{id}/login", handleOIDCLogin)
	g.GET("/api/oidc/finish", handleOIDCCallback)

	// Health check.
	g.GET("/api/health", handleHealthCheck)

	// Uploaded files on localfs.
	if ko.String("upload.provider") == "localfs" {
		g.GET("/uploads/{all:*}", perm(handleServeUploadedFiles))
	}

	// Settings.
	g.GET("/api/settings/general", handleGetGeneralSettings)
	g.PUT("/api/settings/general", perm(handleUpdateGeneralSettings, "settings:manage_general"))
	g.GET("/api/settings/upload", perm(handleGetUploadSettings, "settings:manage_file"))
	g.PUT("/api/settings/upload", perm(handleUpdateUploadSettings, "settings:manage_file"))

	// OIDC.
	g.GET("/api/oidc", handleGetAllOIDC)
	g.GET("/api/oidc/{id}", perm(handleGetOIDC, "login:manage"))
	g.POST("/api/oidc", perm(handleCreateOIDC, "login:manage"))
	g.PUT("/api/oidc/{id}", perm(handleUpdateOIDC, "login:manage"))
	g.DELETE("/api/oidc/{id}", perm(handleDeleteOIDC, "login:manage"))

	// Conversation & message.
	g.GET("/api/conversations/all", perm(handleGetAllConversations, "conversations:all"))
	g.GET("/api/conversations/team", perm(handleGetTeamConversations, "conversations:team"))
	g.GET("/api/conversations/assigned", perm(handleGetAssignedConversations, "conversations:assigned"))
	g.PUT("/api/conversations/{uuid}/assignee/user", perm(handleUpdateConversationUserAssignee, "conversations:edit_user"))
	g.PUT("/api/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee, "conversations:edit_team"))
	g.PUT("/api/conversations/{uuid}/priority", perm(handleUpdateConversationPriority, "conversations:edit_priority"))
	g.PUT("/api/conversations/{uuid}/status", perm(handleUpdateConversationStatus, "conversations:edit_status"))
	g.GET("/api/conversations/{uuid}", perm(handleGetConversation))
	g.GET("/api/conversations/{uuid}/participants", perm(handleGetConversationParticipants))
	g.PUT("/api/conversations/{uuid}/last-seen", perm(handleUpdateConversationAssigneeLastSeen))
	g.POST("/api/conversations/{uuid}/tags", perm(handleAddConversationTags))
	g.GET("/api/conversations/{uuid}/messages", perm(handleGetMessages))
	g.POST("/api/conversations/{uuid}/messages", perm(handleSendMessage))
	g.GET("/api/message/{uuid}/retry", perm(handleRetryMessage))
	g.GET("/api/message/{uuid}", perm(handleGetMessage))

	// Conversation statuses.
	g.GET("/api/statuses", perm(handleGetStatuses))
	g.POST("/api/statuses", perm(handleCreateStatus, "statuses:manage"))
	g.DELETE("/api/statuses/{id}", perm(handleDeleteStatus, "statuses:manage"))
	g.PUT("/api/statuses/{id}", perm(handleUpdateStatus, "statuses:manage"))

	// Conversation priorities.
	g.GET("/api/priorities", perm(handleGetPriorities))

	// Conversation tags.
	g.GET("/api/tags", perm(handleGetTags))
	g.POST("/api/tags", perm(handleCreateTag, "tags:manage"))
	g.DELETE("/api/tags/{id}", perm(handleDeleteTag, "tags:manage"))
	g.PUT("/api/tags/{id}", perm(handleUpdateTag, "tags:manage"))

	// Media.
	g.POST("/api/media", perm(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", perm(handleGetCannedResponses))
	g.POST("/api/canned-responses", perm(handleCreateCannedResponse, "canned_responses:manage"))
	g.PUT("/api/canned-responses/{id}", perm(handleUpdateCannedResponse, "canned_responses:manage"))
	g.DELETE("/api/canned-responses/{id}", perm(handleDeleteCannedResponse, "canned_responses:manage"))

	// User.
	g.GET("/api/users/me", perm(handleGetCurrentUser))
	g.PUT("/api/users/me", perm(handleUpdateCurrentUser))
	g.DELETE("/api/users/me/avatar", perm(handleDeleteAvatar))
	g.GET("/api/users", perm(handleGetUsers, "users:manage"))
	g.GET("/api/users/{id}", perm(handleGetUser, "users:manage"))
	g.POST("/api/users", perm(handleCreateUser, "users:manage"))
	g.PUT("/api/users/{id}", perm(handleUpdateUser, "users:manage"))

	// Team.
	g.GET("/api/teams", perm(handleGetTeams, "teams:manage"))
	g.GET("/api/teams/{id}", perm(handleGetTeam, "teams:manage"))
	g.PUT("/api/teams/{id}", perm(handleUpdateTeam, "teams:manage"))
	g.POST("/api/teams", perm(handleCreateTeam, "teams:manage"))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Automation rules.
	g.GET("/api/automation/rules", perm(handleGetAutomationRules, "automations:manage"))
	g.GET("/api/automation/rules/{id}", perm(handleGetAutomationRule, "automations:manage"))
	g.POST("/api/automation/rules", perm(handleCreateAutomationRule, "automations:manage"))
	g.PUT("/api/automation/rules/{id}/toggle", perm(handleToggleAutomationRule, "automations:manage"))
	g.PUT("/api/automation/rules/{id}", perm(handleUpdateAutomationRule, "automations:manage"))
	g.DELETE("/api/automation/rules/{id}", perm(handleDeleteAutomationRule, "automations:manage"))

	// Inboxes.
	g.GET("/api/inboxes", perm(handleGetInboxes, "inboxes:manage"))
	g.GET("/api/inboxes/{id}", perm(handleGetInbox, "inboxes:manage"))
	g.POST("/api/inboxes", perm(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/inboxes/{id}/toggle", perm(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/inboxes/{id}", perm(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/inboxes/{id}", perm(handleDeleteInbox, "inboxes:manage"))

	// Roles.
	g.GET("/api/roles", perm(handleGetRoles, "roles:manage"))
	g.GET("/api/roles/{id}", perm(handleGetRole, "roles:manage"))
	g.POST("/api/roles", perm(handleCreateRole, "roles:manage"))
	g.PUT("/api/roles/{id}", perm(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/roles/{id}", perm(handleDeleteRole, "roles:manage"))

	// Dashboard.
	g.GET("/api/dashboard/global/counts", perm(handleDashboardCounts, "dashboard:view_global"))
	g.GET("/api/dashboard/global/charts", perm(handleDashboardCharts, "dashboard:view_global"))
	g.GET("/api/dashboard/me/counts", perm(handleUserDashboardCounts))
	g.GET("/api/dashboard/me/charts", perm(handleUserDashboardCharts))

	// Templates.
	g.GET("/api/templates", perm(handleGetTemplates, "templates:manage"))
	g.GET("/api/templates/{id}", perm(handleGetTemplate, "templates:manage"))
	g.POST("/api/templates", perm(handleCreateTemplate, "templates:manage"))
	g.PUT("/api/templates/{id}", perm(handleUpdateTemplate, "templates:manage"))

	// Websocket.
	g.GET("/api/ws", perm(func(r *fastglue.Request) error {
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

// handleServeUploadedFiles serves uploaded files.
func handleServeUploadedFiles(r *fastglue.Request) error {
	filePath := string(r.RequestCtx.URI().Path())
	fasthttp.ServeFile(r.RequestCtx, "./"+filePath)
	return nil
}

// serveIndexPage serves app's default index page.
func serveIndexPage(r *fastglue.Request) error {
	app := r.Context.(*App)

	// Add no-caching headers.
	r.RequestCtx.Response.Header.Add("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")

	// Serve the index.html file from stuffbin fs.
	file, err := app.fs.Get("/frontend/dist/index.html")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "Page not found", nil, "InputException")
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// serveStaticFiles serves static files from stuffbin fs.
func serveStaticFiles(r *fastglue.Request) error {
	var app = r.Context.(*App)

	// Get the requested path
	filePath := string(r.RequestCtx.Path())

	// Serve the file from stuffbin fs.
	finalPath := filepath.Join("frontend/dist", filePath)
	file, err := app.fs.Get(finalPath)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "File not found", nil, "InputException")
	}

	// Detect and set the appropriate Content-Type
	ext := filepath.Ext(filePath)
	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		contentType = http.DetectContentType(file.ReadBytes())
	}
	r.RequestCtx.Response.Header.Set("Content-Type", contentType)
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

func sendErrorEnvelope(r *fastglue.Request, err error) error {
	e, ok := err.(envelope.Error)
	if !ok {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error interface conversion failed", nil, fastglue.ErrorType(envelope.GeneralError))
	}
	return r.SendErrorEnvelope(e.Code, e.Error(), e.Data, fastglue.ErrorType(e.ErrorType))
}

// handleHealthCheck handles the health check.
func handleHealthCheck(r *fastglue.Request) error {
	return r.SendEnvelope(true)
}
