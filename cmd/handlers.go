package main

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/zerodha/fastglue"
)

func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	g.POST("/api/login", handleLogin)
	g.GET("/api/logout", handleLogout)

	g.GET("/api/conversations/all", auth(handleGetAllConversations, "conversations.all"))
	g.GET("/api/conversations/assigned", auth(handleGetAssignedConversations, "conversations.assigned"))
	g.GET("/api/conversations/unassigned", auth(handleGetUnassignedConversations, "conversations.unassigned"))
	g.GET("/api/conversations/assignee/stats", auth(handleAssigneeStats))
	g.GET("/api/conversations/new/stats", auth(handleNewConversationsStats))
	g.GET("/api/conversation/{conversation_uuid}", auth(handleGetConversation))
	g.PUT("/api/conversation/{conversation_uuid}/last-seen", auth(handleUpdateAssigneeLastSeen))
	g.GET("/api/conversation/{conversation_uuid}/participants", auth(handleGetConversationParticipants))
	g.PUT("/api/conversation/{conversation_uuid}/assignee/{assignee_type}", auth(handleUpdateAssignee))
	g.PUT("/api/conversation/{conversation_uuid}/priority", auth(handleUpdatePriority))
	g.PUT("/api/conversation/{conversation_uuid}/status", auth(handleUpdateStatus))
	g.POST("/api/conversation/{conversation_uuid}/tags", auth(handlAddConversationTags))
	g.GET("/api/conversation/{conversation_uuid}/messages", auth(handleGetMessages))
	g.POST("/api/conversation/{conversation_uuid}/message", auth(handleSendMessage))
	g.POST("/api/attachment", auth(handleAttachmentUpload))
	g.GET("/api/message/{message_uuid}/retry", auth(handleRetryMessage))
	g.GET("/api/message/{message_uuid}", auth(handleGetMessage))
	g.GET("/api/canned-responses", auth(handleGetCannedResponses))
	g.POST("/api/upload", auth(handleFileUpload))
	g.POST("/api/upload/view/{file_uuid}", auth(handleViewFile))
	g.GET("/api/users/me", auth(handleGetCurrentUser))
	g.GET("/api/users", auth(handleGetUsers))
	g.GET("/api/teams", auth(handleGetTeams))
	g.GET("/api/tags", auth(handleGetTags))
	g.GET("/api/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))


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

	// Serve the index.html file from the Stuffbin archive.
	file, err := app.fs.Get("/frontend/dist/index.html")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusNotFound, "Page not found", nil, "InputException")
	}
	r.RequestCtx.Response.Header.Set("Content-Type", "text/html")
	r.RequestCtx.SetBody(file.ReadBytes())
	return nil
}

// serveStaticFiles serves static files from the Stuffbin archive.
func serveStaticFiles(r *fastglue.Request) error {
	fmt.Println("static here.")
	app := r.Context.(*App)

	// Get the requested path
	filePath := string(r.RequestCtx.Path())

	fmt.Println("file path ", filePath)

	// Serve the file from the Stuffbin archive
	finalPath := filepath.Join("frontend/dist", filePath)
	fmt.Println("final path ", finalPath)
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
