// Package ws handles WebSocket connections and broadcasting messages to clients.
package ws

import (
	"sync"

	"github.com/abhinavxd/artemis/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// Hub maintains the set of registered clients and their subscriptions.
type Hub struct {
	// Client ID to WS Client map.
	clients      map[int][]*Client
	clientsMutex sync.Mutex

	// Map of conversation UUID to client id list.
	conversationSubs map[string][]int
	subMutex         sync.Mutex

	// Store to fetch conversation UUIDs for subscriptions.
	conversationStore ConversationStore
}

// ConversationStore defines the interface for retrieving conversation UUIDs.
type ConversationStore interface {
	GetConversationsListUUIDs(userID, page, pageSize int, typ string) ([]string, error)
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
func (h *Hub) BroadcastMessage(msg models.BroadcastMessage) {
	h.clientsMutex.Lock()
	defer h.clientsMutex.Unlock()
	for _, userID := range msg.Users {
		for _, client := range h.clients[userID] {
			client.SendMessage(msg.Data, websocket.TextMessage)
		}
	}
}

func (h *Hub) GetConversationSubscribers(conversationUUID string) []int {
	h.subMutex.Lock()
	defer h.subMutex.Unlock()
	return h.conversationSubs[conversationUUID]
}
