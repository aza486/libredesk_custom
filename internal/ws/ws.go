package ws

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/abhinavxd/artemis/internal/ws/models"
	"github.com/fasthttp/websocket"
)

// Hub maintains the set of registered clients.
type Hub struct {
	clients      map[int][]*Client
	clientsMutex sync.Mutex

	// Map of conversation uuid to slice of subbed userids.
	Csubs  map[string][]int
	SubMut sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[int][]*Client, 100000),
		clientsMutex: sync.Mutex{},
		Csubs:        map[string][]int{},
		SubMut:       sync.Mutex{},
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

// ClientAlreadyConnected checks if a user id is already connected or not.
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
				fmt.Printf("Pushing msg to %d", userID)
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

func (c *Hub) BroadcastNewMsg(convUUID string, msg map[string]interface{}) {
	// clientIDs, ok := c.Csubs[convUUID]
	// if !ok || len(clientIDs) == 0 {
	// 	return
	// }

	clientIDs := []int{1, 2}

	data := map[string]interface{}{
		"ev": models.EventNewMsg,
		"d":  msg,
	}

	// Marshal.
	dataB, err := json.Marshal(data)
	if err != nil {
		return
	}

	c.PushMessage(PushMessage{
		Data:  dataB,
		Users: clientIDs,
	})
}

func (c *Hub) BroadcastMsgStatus(convUUID string, msg map[string]interface{}) {
	// clientIDs, ok := c.Csubs[convUUID]
	// if !ok || len(clientIDs) == 0 {
	// 	return
	// }

	clientIDs := []int{1, 2}

	data := map[string]interface{}{
		"ev": models.EventMsgStatusUpdate,
		"d":  msg,
	}

	// Marshal.
	dataB, err := json.Marshal(data)
	if err != nil {
		return
	}

	c.PushMessage(PushMessage{
		Data:  dataB,
		Users: clientIDs,
	})
}
