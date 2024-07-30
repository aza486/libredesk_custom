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
	g.GET("/auth/oidc/login", handleOIDCLogin)
	g.GET("/auth/oidc/finish", handleOIDCCallback)

	g.GET("/api/settings", aauth(handleGetSettings))

	// Conversation.
	g.GET("/api/conversations/all", aauth(handleGetAllConversations, "conversation:all"))
	g.GET("/api/conversations/team", aauth(handleGetTeamConversations, "conversation:team"))
	g.GET("/api/conversations/assigned", aauth(handleGetAssignedConversations, "conversation:assigned"))
	g.GET("/api/conversations/{uuid}", aauth(handleGetConversation))
	g.GET("/api/conversations/{uuid}/participants", aauth(handleGetConversationParticipants))
	g.PUT("/api/conversations/{uuid}/last-seen", aauth(handleUpdateAssigneeLastSeen))
	g.PUT("/api/conversations/{uuid}/assignee/user", aauth(handleUpdateUserAssignee))
	g.PUT("/api/conversations/{uuid}/assignee/team", aauth(handleUpdateTeamAssignee))
	g.PUT("/api/conversations/{uuid}/priority", aauth(handleUpdatePriority, "conversation:edit_priority"))
	g.PUT("/api/conversations/{uuid}/status", aauth(handleUpdateStatus, "conversation:edit_status"))
	g.POST("/api/conversations/{uuid}/tags", aauth(handleAddConversationTags))
	g.GET("/api/conversations/{uuid}/messages", aauth(handleGetMessages))
	g.POST("/api/conversations/{uuid}/messages", aauth(handleSendMessage))
	g.GET("/api/message/{uuid}/retry", aauth(handleRetryMessage))
	g.GET("/api/message/{uuid}", aauth(handleGetMessage))

	// Media.
	g.POST("/api/media", aauth(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", aauth(handleGetCannedResponses))

	// User.
	g.GET("/api/users/me", aauth(handleGetCurrentUser, "users:manage"))
	g.GET("/api/users", aauth(handleGetUsers, "users:manage"))
	g.GET("/api/users/{id}", aauth(handleGetUser, "users:manage"))
	g.PUT("/api/users/{id}", aauth(handleUpdateUser, "users:manage"))
	g.POST("/api/users", aauth(handleCreateUser, "users:manage"))

	// Team.
	g.GET("/api/teams", aauth(handleGetTeams, "teams:manage"))
	g.GET("/api/teams/{id}", aauth(handleGetTeam, "teams:manage"))
	g.PUT("/api/teams/{id}", aauth(handleUpdateTeam, "teams:manage"))
	g.POST("/api/teams", aauth(handleCreateTeam, "teams:manage"))

	// Tags.
	g.GET("/api/tags", aauth(handleGetTags))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Websocket.
	g.GET("/api/ws", aauth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Automation rules.
	g.GET("/api/automation/rules", aauth(handleGetAutomationRules, "automations:manage"))
	g.GET("/api/automation/rules/{id}", aauth(handleGetAutomationRule, "automations:manage"))
	g.POST("/api/automation/rules", aauth(handleCreateAutomationRule, "automations:manage"))
	g.PUT("/api/automation/rules/{id}/toggle", aauth(handleToggleAutomationRule, "automations:manage"))
	g.PUT("/api/automation/rules/{id}", aauth(handleUpdateAutomationRule, "automations:manage"))
	g.DELETE("/api/automation/rules/{id}", aauth(handleDeleteAutomationRule, "automations:manage"))

	// Inboxes.
	g.GET("/api/inboxes", aauth(handleGetInboxes, "inboxes:manage"))
	g.GET("/api/inboxes/{id}", aauth(handleGetInbox, "inboxes:manage"))
	g.POST("/api/inboxes", aauth(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/inboxes/{id}/toggle", aauth(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/inboxes/{id}", aauth(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/inboxes/{id}", aauth(handleDeleteInbox, "inboxes:manage"))

	// Roles.
	g.GET("/api/roles", aauth(handleGetRoles, "roles:manage"))
	g.GET("/api/roles/{id}", aauth(handleGetRole, "roles:manage"))
	g.POST("/api/roles", aauth(handleCreateRole, "roles:manage"))
	g.PUT("/api/roles/{id}", aauth(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/roles/{id}", aauth(handleDeleteRole, "roles:manage"))

	// Dashboard.
	g.GET("/api/dashboard/me/counts", aauth(handleUserDashboardCounts))
	g.GET("/api/dashboard/me/charts", aauth(handleUserDashboardCharts))
	// g.GET("/api/dashboard/team/:teamName/counts", aauth(handleTeamCounts))
	// g.GET("/api/dashboard/team/:teamName/charts", aauth(handleTeamCharts))
	// g.GET("/api/dashboard/global/counts", aauth(handleGlobalCounts))
	// g.GET("/api/dashboard/global/charts", aauth(handleGlobalCharts))

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
