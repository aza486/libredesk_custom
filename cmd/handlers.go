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
	g.GET("/api/oidc/login", handleOIDCLogin)
	g.GET("/api/oidc/finish", handleOIDCCallback)

	// Health check.
	g.GET("/api/health", handleHealthCheck)

	// Settings.
	g.GET("/api/settings", perm(handleGetSettings))

	// Conversation.
	g.GET("/api/conversations/all", perm(handleGetAllConversations, "conversation:all"))
	g.GET("/api/conversations/team", perm(handleGetTeamConversations, "conversation:team"))
	g.GET("/api/conversations/assigned", perm(handleGetAssignedConversations, "conversation:assigned"))
	g.GET("/api/conversations/{uuid}", perm(handleGetConversation))
	g.GET("/api/conversations/{uuid}/participants", perm(handleGetConversationParticipants))
	g.PUT("/api/conversations/{uuid}/last-seen", perm(handleUpdateAssigneeLastSeen))
	g.PUT("/api/conversations/{uuid}/assignee/user", perm(handleUpdateUserAssignee))
	g.PUT("/api/conversations/{uuid}/assignee/team", perm(handleUpdateTeamAssignee))
	g.PUT("/api/conversations/{uuid}/priority", perm(handleUpdatePriority, "conversation:edit_priority"))
	g.PUT("/api/conversations/{uuid}/status", perm(handleUpdateStatus, "conversation:edit_status"))
	g.POST("/api/conversations/{uuid}/tags", perm(handleAddConversationTags))
	g.GET("/api/conversations/{uuid}/messages", perm(handleGetMessages))
	g.POST("/api/conversations/{uuid}/messages", perm(handleSendMessage))
	g.GET("/api/message/{uuid}/retry", perm(handleRetryMessage))
	g.GET("/api/message/{uuid}", perm(handleGetMessage))

	// Media.
	g.POST("/api/media", perm(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", perm(handleGetCannedResponses))

	// User.
	g.GET("/api/users/me", perm(handleGetCurrentUser))
	g.GET("/api/users", perm(handleGetUsers, "users:manage"))
	g.GET("/api/users/{id}", perm(handleGetUser, "users:manage"))
	g.POST("/api/users", perm(handleCreateUser, "users:manage"))
	g.PUT("/api/users/{id}", perm(handleUpdateUser, "users:manage"))
	g.PUT("/api/users/me", perm(handleUpdateCurrentUser))
	g.DELETE("/api/users/me/avatar", perm(handleDeleteAvatar))

	// Team.
	g.GET("/api/teams", perm(handleGetTeams, "teams:manage"))
	g.GET("/api/teams/{id}", perm(handleGetTeam, "teams:manage"))
	g.PUT("/api/teams/{id}", perm(handleUpdateTeam, "teams:manage"))
	g.POST("/api/teams", perm(handleCreateTeam, "teams:manage"))

	// Tags.
	g.GET("/api/tags", perm(handleGetTags))

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
	g.GET("/api/dashboard/global/counts", perm(handleDashboardCounts))
	g.GET("/api/dashboard/global/charts", perm(handleDashboardCharts))
	g.GET("/api/dashboard/me/counts", perm(handleUserDashboardCounts))
	g.GET("/api/dashboard/me/charts", perm(handleUserDashboardCharts))

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

// handleHealthCheck handles the health check endpoint by pinging the PostgreSQL and Redis.
func handleHealthCheck(r *fastglue.Request) error {
	var app = r.Context.(*App)

	// Ping DB.
	if err := app.db.Ping(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "DB ping failed.", nil, envelope.GeneralError)
	}

	// Ping Redis.
	if err := app.rdb.Ping(r.RequestCtx).Err(); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Redis ping failed.", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}
