package main

import (
	"github.com/knadh/koanf/v2"
	"github.com/zerodha/fastglue"
)

func initHandlers(g *fastglue.Fastglue, app *App, ko *koanf.Koanf) {
	g.POST("/api/login", handleLogin)
	g.GET("/api/logout", handleLogout)
	g.GET("/api/conversations", authSession(handleGetConversations))
	g.GET("/api/conversation/{uuid}", authSession(handleGetConversation))
	g.GET("/api/conversation/{uuid}/messages", authSession(handleGetMessages))
	g.PUT("/api/conversation/{uuid}/assignee/{assignee_type}", authSession(handleUpdateAssignee))
	g.PUT("/api/conversation/{uuid}/priority", authSession(handleUpdatePriority))
	g.PUT("/api/conversation/{uuid}/status", authSession(handleUpdateStatus))

	g.POST("/api/conversation/{uuid}/tags", authSession(handleUpsertConvTag))

	g.GET("/api/profile", authSession(handleGetAgentProfile))
	g.GET("/api/canned_responses", authSession(handleGetCannedResponses))
	g.POST("/api/media", authSession(handleMediaUpload))
	g.GET("/api/agents", authSession(handleGetAgents))
	g.GET("/api/teams", authSession(handleGetTeams))
	g.GET("/api/tags", authSession(handleGetTags))
}
