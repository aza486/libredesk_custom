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

	g.GET("/api/settings", auth(handleGetSettings))

	// Conversation.
	g.GET("/api/conversations/all", auth(handleGetAllConversations, "conversation:all"))
	g.GET("/api/conversations/team", auth(handleGetTeamConversations, "conversation:team"))
	g.GET("/api/conversations/assigned", auth(handleGetAssignedConversations, "conversation:assigned"))
	g.GET("/api/conversations/{uuid}", auth(handleGetConversation))
	g.GET("/api/conversations/{uuid}/participants", auth(handleGetConversationParticipants))
	g.PUT("/api/conversations/{uuid}/last-seen", auth(handleUpdateAssigneeLastSeen))
	g.PUT("/api/conversations/{uuid}/assignee/user", auth(handleUpdateUserAssignee))
	g.PUT("/api/conversations/{uuid}/assignee/team", auth(handleUpdateTeamAssignee))
	g.PUT("/api/conversations/{uuid}/priority", auth(handleUpdatePriority, "conversation:edit_priority"))
	g.PUT("/api/conversations/{uuid}/status", auth(handleUpdateStatus, "conversation:edit_status"))
	g.POST("/api/conversations/{uuid}/tags", auth(handleAddConversationTags))
	g.GET("/api/conversations/{uuid}/messages", auth(handleGetMessages))
	g.GET("/api/message/{uuid}/retry", auth(handleRetryMessage))
	g.GET("/api/message/{uuid}", auth(handleGetMessage))
	g.POST("/api/conversations/{uuid}/messages", auth(handleSendMessage))

	// Media.
	g.POST("/api/media", auth(handleMediaUpload))

	// Canned response.
	g.GET("/api/canned-responses", auth(handleGetCannedResponses))

	// User.
	g.GET("/api/users/me", auth(handleGetCurrentUser, "users:manage"))
	g.GET("/api/users", auth(handleGetUsers, "users:manage"))
	g.GET("/api/users/{id}", auth(handleGetUser, "users:manage"))
	g.PUT("/api/users/{id}", auth(handleUpdateUser, "users:manage"))
	g.POST("/api/users", auth(handleCreateUser, "users:manage"))

	// Team.
	g.GET("/api/teams", auth(handleGetTeams, "teams:manage"))
	g.GET("/api/teams/{id}", auth(handleGetTeam, "teams:manage"))
	g.PUT("/api/teams/{id}", auth(handleUpdateTeam, "teams:manage"))
	g.POST("/api/teams", auth(handleCreateTeam, "teams:manage"))

	// Tags.
	g.GET("/api/tags", auth(handleGetTags))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Websocket.
	g.GET("/api/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Automation rules.
	g.GET("/api/automation/rules", auth(handleGetAutomationRules, "automations:manage"))
	g.GET("/api/automation/rules/{id}", auth(handleGetAutomationRule, "automations:manage"))
	g.POST("/api/automation/rules", auth(handleCreateAutomationRule, "automations:manage"))
	g.PUT("/api/automation/rules/{id}/toggle", auth(handleToggleAutomationRule, "automations:manage"))
	g.PUT("/api/automation/rules/{id}", auth(handleUpdateAutomationRule, "automations:manage"))
	g.DELETE("/api/automation/rules/{id}", auth(handleDeleteAutomationRule, "automations:manage"))

	// Inboxes.
	g.GET("/api/inboxes", auth(handleGetInboxes, "inboxes:manage"))
	g.GET("/api/inboxes/{id}", auth(handleGetInbox, "inboxes:manage"))
	g.POST("/api/inboxes", auth(handleCreateInbox, "inboxes:manage"))
	g.PUT("/api/inboxes/{id}/toggle", auth(handleToggleInbox, "inboxes:manage"))
	g.PUT("/api/inboxes/{id}", auth(handleUpdateInbox, "inboxes:manage"))
	g.DELETE("/api/inboxes/{id}", auth(handleDeleteInbox, "inboxes:manage"))

	// Roles.
	g.GET("/api/roles", auth(handleGetRoles, "roles:manage"))
	g.GET("/api/roles/{id}", auth(handleGetRole, "roles:manage"))
	g.POST("/api/roles", auth(handleCreateRole, "roles:manage"))
	g.PUT("/api/roles/{id}", auth(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/roles/{id}", auth(handleDeleteRole, "roles:manage"))

	// Dashboard.
	g.GET("/api/dashboard/me/counts", auth(handleUserDashboardCounts))
	g.GET("/api/dashboard/me/charts", auth(handleUserDashboardCharts))
	// g.GET("/api/dashboard/team/:teamName/counts", auth(handleTeamCounts))
	// g.GET("/api/dashboard/team/:teamName/charts", auth(handleTeamCharts))
	// g.GET("/api/dashboard/global/counts", auth(handleGlobalCounts))
	// g.GET("/api/dashboard/global/charts", auth(handleGlobalCharts))

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
