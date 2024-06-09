package main

import (
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/zerodha/fastglue"
)

func initHandlers(g *fastglue.Fastglue, hub *ws.Hub) {
	g.POST("/api/login", handleLogin)
	g.GET("/api/logout", handleLogout)
	g.GET("/api/conversations", auth(handleGetConversations))
	g.GET("/api/conversation/{conversation_uuid}", auth(handleGetConversation))
	g.PUT("/api/conversation/{conversation_uuid}/last-seen", auth(handleUpdateAssigneeLastSeen))
	g.GET("/api/conversation/{conversation_uuid}/participants", auth(handleGetConversationParticipants))
	g.PUT("/api/conversation/{conversation_uuid}/assignee/{assignee_type}", auth(handleUpdateAssignee))
	g.PUT("/api/conversation/{conversation_uuid}/priority", auth(handleUpdatePriority))
	g.PUT("/api/conversation/{conversation_uuid}/status", auth(handleUpdateStatus))
	g.POST("/api/conversation/{conversation_uuid}/tags", auth(handlAddConversationTags))
	g.GET("/api/conversation/{conversation_uuid}/messages", auth(handleGetMessages))
	g.GET("/api/message/{message_uuid}", auth(handleGetMessage))
	g.GET("/api/message/{message_uuid}/retry", auth(handleRetryMessage))
	g.POST("/api/conversation/{conversation_uuid}/message", auth(handleSendMessage))
	g.GET("/api/canned-responses", auth(handleGetCannedResponses))
	g.POST("/api/attachment", auth(handleAttachmentUpload))
	g.GET("/api/attachment/{conversation_uuid}", auth(handleGetAttachment))
	g.GET("/api/users/me", auth(handleGetCurrentUser))
	g.GET("/api/users", auth(handleGetUsers))
	g.GET("/api/teams", auth(handleGetTeams))
	g.GET("/api/tags", auth(handleGetTags))
	g.GET("/api/ws", auth(func(r *fastglue.Request) error {
		return handleWS(r, hub)
	}))
}
