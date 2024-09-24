package conversation

import (
	"encoding/json"
	"time"

	wsmodels "github.com/abhinavxd/artemis/internal/ws/models"
)

func (m *Manager) BroadcastNewConversationMessage(conversationUUID, content, messageUUID, lastMessageAt, typ string, private bool) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeNewMessage,
		Data: map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"content":           content,
			"created_at":        lastMessageAt,
			"uuid":              messageUUID,
			"private":           private,
			"type":              typ,
		},
	}
	m.broadcastToConversation(conversationUUID, message)
}

func (m *Manager) BroadcastMessagePropUpdate(conversationUUID, messageUUID, prop, value string) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeMessagePropUpdate,
		Data: map[string]interface{}{
			"uuid":  messageUUID,
			"prop":  prop,
			"value": value,
		},
	}
	m.broadcastToConversation(conversationUUID, message)
}

func (m *Manager) BroadcastNewConversation(userID int, conversationUUID, avatarURL, firstName, lastName, lastMessage, inboxName string, lastMessageAt time.Time, unreadMessageCount int) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeNewConversation,
		Data: map[string]interface{}{
			"uuid":                 conversationUUID,
			"avatar_url":           avatarURL,
			"first_name":           firstName,
			"last_name":            lastName,
			"last_message":         lastMessage,
			"last_message_at":      lastMessageAt.Format(time.RFC3339),
			"inbox_name":           inboxName,
			"unread_message_count": unreadMessageCount,
		},
	}
	m.broadcastToUsers([]int{userID}, message)
}

func (m *Manager) BroadcastConversationPropertyUpdate(conversationUUID, prop, value string) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeConversationPropertyUpdate,
		Data: map[string]interface{}{
			"uuid":  conversationUUID,
			"prop":  prop,
			"value": value,
		},
	}
	m.broadcastToConversation(conversationUUID, message)
}

func (m *Manager) broadcastToConversation(conversationUUID string, message wsmodels.Message) {
	userIDs := m.wsHub.GetConversationSubscribers(conversationUUID)
	m.broadcastToUsers(userIDs, message)
}

func (m *Manager) broadcastToUsers(userIDs []int, message wsmodels.Message) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		m.lo.Error("error marshlling message", "error", err)
		return
	}
	m.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{
		Data:  messageBytes,
		Users: userIDs,
	})
}
