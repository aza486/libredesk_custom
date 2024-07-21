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

	// Conversation.
	g.GET("/api/conversations/all", auth(handleGetAllConversations, "conversations:all"))
	g.GET("/api/conversations/team", auth(handleGetTeamConversations, "conversations:team"))
	g.GET("/api/conversations/assigned", auth(handleGetAssignedConversations, "conversations:assigned"))	

	g.GET("/api/conversations/{uuid}", auth(handleGetConversation))
	g.GET("/api/conversations/{uuid}/participants", auth(handleGetConversationParticipants))
	g.PUT("/api/conversations/{uuid}/last-seen", auth(handleUpdateAssigneeLastSeen))
	g.PUT("/api/conversations/{uuid}/assignee/user", auth(handleUpdateUserAssignee))
	g.PUT("/api/conversations/{uuid}/assignee/team", auth(handleUpdateTeamAssignee))
	g.PUT("/api/conversations/{uuid}/priority", auth(handleUpdatePriority))
	g.PUT("/api/conversations/{uuid}/status", auth(handleUpdateStatus))
	g.POST("/api/conversations/{uuid}/tags", auth(handleAddConversationTags))

	// Message.
	g.GET("/api/conversations/{uuid}/messages", auth(handleGetMessages))
	g.GET("/api/message/{uuid}/retry", auth(handleRetryMessage))
	g.GET("/api/message/{uuid}", auth(handleGetMessage))
	g.POST("/api/conversations/{uuid}/messages", auth(handleSendMessage))

	// Attachment.
	g.POST("/api/attachment", auth(handleAttachmentUpload))

	// Canned response.
	g.GET("/api/canned-responses", auth(handleGetCannedResponses))

	// File upload.
	g.POST("/api/file/upload", auth(handleFileUpload))

	// User.
	g.GET("/api/users/me", auth(handleGetCurrentUser))
	g.GET("/api/users", auth(handleGetUsers))
	g.GET("/api/users/{id}", auth(handleGetUser))
	g.PUT("/api/users/{id}", auth(handleUpdateUser))
	g.POST("/api/users", auth(handleCreateUser))

	// Team.
	g.GET("/api/teams", auth(handleGetTeams))
	g.GET("/api/teams/{id}", auth(handleGetTeam))
	g.PUT("/api/teams/{id}", auth(handleUpdateTeam))
	g.POST("/api/teams", auth(handleCreateTeam))

	// Tags.
	g.GET("/api/tags", auth(handleGetTags))

	// i18n.
	g.GET("/api/lang/{lang}", handleGetI18nLang)

	// Websocket.
	g.GET("/api/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))

	// Automation rules.
	g.GET("/api/automation/rules", handleGetAutomationRules)
	g.GET("/api/automation/rules/{id}", handleGetAutomationRule)
	g.POST("/api/automation/rules", handleCreateAutomationRule)
	g.PUT("/api/automation/rules/{id}/toggle", handleToggleAutomationRule)
	g.PUT("/api/automation/rules/{id}", handleUpdateAutomationRule)
	g.DELETE("/api/automation/rules/{id}", handleDeleteAutomationRule)

	// Inboxes.
	g.GET("/api/inboxes", handleGetInboxes)
	g.GET("/api/inboxes/{id}", handleGetInbox)
	g.POST("/api/inboxes", handleCreateInbox)
	g.PUT("/api/inboxes/{id}/toggle", handleToggleInbox)
	g.PUT("/api/inboxes/{id}", handleUpdateInbox)
	g.DELETE("/api/inboxes/{id}", handleDeleteInbox)

	// Roles.
	g.GET("/api/roles", handleGetRoles)
	g.GET("/api/roles/{id}", handleGetRole)
	g.POST("/api/roles", handleCreateRole)
	g.PUT("/api/roles/{id}", handleUpdateRole)
	g.DELETE("/api/roles/{id}", handleDeleteRole)

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
