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
	clients      map[int][]*Client
	clientsMutex sync.Mutex

	// Map of conversation uuid to a set of subscribed user IDs.
	ConversationSubs map[string][]int

	SubMut            sync.Mutex
	conversationStore ConversationStore
}

type ConversationStore interface {
	GetConversationUUIDs(userID, page, pageSize int, typ, predefinedFilter string) ([]string, error)
}

func NewHub() *Hub {
	return &Hub{
		clients:          make(map[int][]*Client, 10000),
		clientsMutex:     sync.Mutex{},
		ConversationSubs: make(map[string][]int),
		SubMut:           sync.Mutex{},
	}
}

type Message struct {
	messageType int
	data        []byte
}

type PushMessage struct {
	Data     []byte `json:"data"`
	Users    []int  `json:"users"`
	MaxUsers int    `json:"max_users"`
}

func (h *Hub) SetConversationStore(store ConversationStore) {
	h.conversationStore = store
}

func (h *Hub) AddClient(c *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	h.clients[c.ID] = append(h.clients[c.ID], c)
}

func (h *Hub) RemoveClient(client *Client) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	if clients, ok := h.clients[client.ID]; ok {
		for i, c := range clients {
			if c == client {
				// Remove the client from the slice
				h.clients[client.ID] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}
}

// ClientAlreadyConnected returns true if the client with this id is already connected else returns false.
func (h *Hub) ClientAlreadyConnected(id int) bool {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	_, ok := h.clients[id]
	return ok
}

// PushMessage broadcasts a JSON data packet to all or some userIDs (with an optional max cap).
// If userIDs is empty, the broadcast goes to all users.
func (h *Hub) PushMessage(m PushMessage) {
	if len(m.Users) != 0 {
		// The message has to go to specific userIDs.
		h.clientsMutex.Lock()
		for _, userID := range m.Users {
			for _, c := range h.clients[userID] {
				c.Conn.WriteMessage(websocket.TextMessage, m.Data)
			}
		}
		h.clientsMutex.Unlock()
	} else {
		// Message goes to all connected users.
		n := 0
		h.clientsMutex.Lock()
		for _, cls := range h.clients {
			for _, c := range cls {
				c.Conn.WriteMessage(websocket.TextMessage, m.Data)
				if m.MaxUsers > 0 && n >= m.MaxUsers {
					break
				}
			}
			n++
		}
		h.clientsMutex.Unlock()
	}
}

func (c *Hub) BroadcastNewConversationMessage(conversationUUID, trimmedMessage, messageUUID, lastMessageAt string, private bool) {
	userIDs, ok := c.ConversationSubs[conversationUUID]
	if !ok || len(userIDs) == 0 {
		return
	}

	message := models.Message{
		Type: models.MessageTypeNewMessage,
		Data: map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"last_message":      trimmedMessage,
			"uuid":              messageUUID,
			"last_message_at":   lastMessageAt,
			"private":           private,
		},
	}

	c.marshalAndPush(message, userIDs)
}

func (c *Hub) BroadcastMessagePropUpdate(conversationUUID, messageUUID, prop, value string) {
	userIDs, ok := c.ConversationSubs[conversationUUID]
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

	c.marshalAndPush(message, userIDs)
}

func (c *Hub) BroadcastConversationAssignment(userID int, conversationUUID string, avatarUrl string, firstName, lastName, lastMessage, inboxName string, lastMessageAt time.Time, unreadMessageCount int) {
	message := models.Message{
		Type: models.MessageTypeNewConversation,
		Data: map[string]interface{}{
			"uuid":                 conversationUUID,
			"avatar_url":           avatarUrl,
			"first_name":           firstName,
			"last_name":            lastName,
			"last_message":         lastMessage,
			"last_message_at":      time.Now().Format(time.RFC3339),
			"inbox_name":           inboxName,
			"unread_message_count": unreadMessageCount,
		},
	}
	c.marshalAndPush(message, []int{userID})
}

func (c *Hub) BroadcastConversationPropertyUpdate(conversationUUID, prop string, val string) {
	userIDs, ok := c.ConversationSubs[conversationUUID]
	if !ok || len(userIDs) == 0 {
		return
	}

	message := models.Message{
		Type: models.MessageTypeConversationPropertyUpdate,
		Data: map[string]interface{}{
			"uuid": conversationUUID,
			"prop": prop,
			"val":  val,
		},
	}

	c.marshalAndPush(message, userIDs)
}

func (c *Hub) marshalAndPush(message models.Message, userIDs []int) {
	messageB, err := json.Marshal(message)
	if err != nil {
		return
	}

	fmt.Println("pushing ws msg", string(messageB), "type", message.Type, "to_user_ids", userIDs, "connected_userIds", len(c.clients))

	c.PushMessage(PushMessage{
		Data:  messageB,
		Users: userIDs,
	})
}
