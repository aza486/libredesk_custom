// Package ws handles WebSocket connections and broadcasting messages to clients.
package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// Hub maintains the set of registered clients and their subscriptions.
type Hub struct {
	clients          map[int][]*Client
	clientsMutex     sync.Mutex
	conversationSubs map[string][]int
	subMutex         sync.Mutex
	conversationStore ConversationStore
}

// ConversationStore defines the interface for retrieving conversation UUIDs.
type ConversationStore interface {
	GetConversationUUIDs(userID, page, pageSize int, typ, predefinedFilter string) ([]string, error)
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients:          make(map[int][]*Client, 10000),
		clientsMutex:     sync.Mutex{},
		conversationSubs: make(map[string][]int),
		subMutex:         sync.Mutex{},
	}
}

// Message represents a WebSocket message.
type Message struct {
	messageType int
	data        []byte
}

// PushMessage represents a message to be pushed to clients.
type PushMessage struct {
	Data     []byte `json:"data"`
	Users    []int  `json:"users"`
	MaxUsers int    `json:"max_users"`
}

// SetConversationStore sets the conversation store for the hub.
func (h *Hub) SetConversationStore(store ConversationStore) {
	h.conversationStore = store
}

// AddClient adds a new client to the hub.
func (h *Hub) AddClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	h.clients[client.ID] = append(h.clients[client.ID], client)
}

// RemoveClient removes a client from the hub.
func (h *Hub) RemoveClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	if clients, ok := h.clients[client.ID]; ok {
		for i, c := range clients {
			if c == client {
				h.clients[client.ID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}
}

// ClientAlreadyConnected returns true if the client with this ID is already connected.
func (h *Hub) ClientAlreadyConnected(id int) bool {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	_, ok := h.clients[id]
	return ok
}

// PushMessage broadcasts a JSON data packet to specified userIDs (with an optional max cap).
// If userIDs is empty, the broadcast goes to all users.
func (h *Hub) PushMessage(msg PushMessage) {
	if len(msg.Users) != 0 {
		h.clientsMutex.Lock()
		for _, userID := range msg.Users {
			for _, client := range h.clients[userID] {
				client.Conn.WriteMessage(websocket.TextMessage, msg.Data)
			}
		}
		h.clientsMutex.Unlock()
	} else {
		n := 0
		h.clientsMutex.Lock()
		for _, clients := range h.clients {
			for _, client := range clients {
				client.Conn.WriteMessage(websocket.TextMessage, msg.Data)
				if msg.MaxUsers > 0 && n >= msg.MaxUsers {
					break
				}
			}
			n++
		}
		h.clientsMutex.Unlock()
	}
}

// BroadcastNewConversationMessage broadcasts a new conversation message.
func (h *Hub) BroadcastNewConversationMessage(conversationUUID, content, messageUUID, lastMessageAt string, private bool) {
	userIDs, ok := h.conversationSubs[conversationUUID]
	if !ok || len(userIDs) == 0 {
		return
	}

	message := models.Message{
		Type: models.MessageTypeNewMessage,
		Data: map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"last_message":      content,
			"uuid":              messageUUID,
			"last_message_at":   lastMessageAt,
			"private":           private,
		},
	}

	h.marshalAndPush(message, userIDs)
}

// BroadcastMessagePropUpdate broadcasts an update to a message property.
func (h *Hub) BroadcastMessagePropUpdate(conversationUUID, messageUUID, prop, value string) {
	userIDs, ok := h.conversationSubs[conversationUUID]
	if !ok || len(userIDs) == 0 {
		return
	}

	message := models.Message{
		Type: models.MessageTypeMessagePropUpdate,
		Data: map[string]interface{}{
			"uuid": messageUUID,
			"prop": prop,
			"val":  value,
		},
	}

	h.marshalAndPush(message, userIDs)
}

// BroadcastConversationAssignment broadcasts the assignment of a conversation.
func (h *Hub) BroadcastConversationAssignment(userID int, conversationUUID, avatarURL, firstName, lastName, lastMessage, inboxName string, lastMessageAt time.Time, unreadMessageCount int) {
	message := models.Message{
		Type: models.MessageTypeNewConversation,
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
	h.marshalAndPush(message, []int{userID})
}

// BroadcastConversationPropertyUpdate broadcasts an update to a conversation property.
func (h *Hub) BroadcastConversationPropertyUpdate(conversationUUID, prop, value string) {
	userIDs, ok := h.conversationSubs[conversationUUID]
	if !ok || len(userIDs) == 0 {
		return
	}

	message := models.Message{
		Type: models.MessageTypeConversationPropertyUpdate,
		Data: map[string]interface{}{
			"uuid": conversationUUID,
			"prop": prop,
			"val":  value,
		},
	}

	h.marshalAndPush(message, userIDs)
}

// marshalAndPush marshals a message and pushes it to the specified user IDs.
func (h *Hub) marshalAndPush(message models.Message, userIDs []int) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return
	}

	fmt.Println("pushing ws msg", string(messageBytes), "type", message.Type, "to_user_ids", userIDs, "connected_userIds", len(h.clients))

	h.PushMessage(PushMessage{
		Data:  messageBytes,
		Users: userIDs,
	})
}
