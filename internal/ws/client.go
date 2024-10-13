package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/ws/models"
	"github.com/fasthttp/websocket"
)

const (
	maxConversationsPagesToSub = 10
	maxConversationsPageSize   = 50
)

// SafeBool is a thread-safe boolean.
type SafeBool struct {
	flag bool
	mu   sync.Mutex
}

// Set sets the value of the SafeBool.
func (b *SafeBool) Set(value bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flag = value
}

// Get returns the value of the SafeBool.
func (b *SafeBool) Get() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.flag
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// Client ID.
	ID int

	// Hub.
	Hub *Hub

	// WebSocket connection.
	Conn *websocket.Conn

	// To prevent pushes to the channel.
	Closed SafeBool

	// Currently opened conversation UUID.
	CurrentConversationUUID string

	// Buffered channel of outbound ws messages.
	Send chan models.WSMessage
}

// Serve handles heartbeats and sending messages to the client.
func (c *Client) Serve() {
	var heartBeatTicker = time.NewTicker(2 * time.Second)
	defer heartBeatTicker.Stop()

Loop:
	for {
		select {
		case <-heartBeatTicker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("error writing message", err)
				return
			}
		case msg, ok := <-c.Send:
			if !ok {
				break Loop
			}
			c.Conn.WriteMessage(msg.MessageType, msg.Data)
		}
	}
	c.Conn.Close()
}

// Listen listens for incoming messages from the client.
func (c *Client) Listen() {
	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage {
			c.processIncomingMessage(msg)
		} else {
			c.Hub.RemoveClient(c)
			c.close()
			return
		}
	}
	c.Hub.RemoveClient(c)
	c.close()
}

// processIncomingMessage processes incoming messages from the client.
func (c *Client) processIncomingMessage(data []byte) {
	var req models.IncomingReq
	if err := json.Unmarshal(data, &req); err != nil {
		c.SendError("error unmarshalling request")
		return
	}

	switch req.Action {
	case models.ActionConversationsListSub:
		var subReq models.ConversationsListSubscribe
		if err := json.Unmarshal(data, &subReq); err != nil {
			c.SendError("error unmarshalling request")
			return
		}

		// First remove all user conversation subscriptions.
		c.RemoveAllUserConversationSubscriptions(c.ID)

		// Add the new subscriptions.
		for page := 1; page <= maxConversationsPagesToSub; page++ {
			conversationUUIDs, err := c.Hub.conversationStore.GetConversationsListUUIDs(c.ID, page, maxConversationsPageSize, subReq.Type)
			if err != nil {
				continue
			}
			c.SubscribeConversations(c.ID, conversationUUIDs)
		}
	case models.ActionSetCurrentConversation:
		var subReq models.ConversationCurrentSet
		if err := json.Unmarshal(data, &subReq); err != nil {
			c.SendError("error unmarshalling request")
			return
		}

		if c.CurrentConversationUUID != subReq.UUID {
			c.UnsubscribeConversation(c.ID, c.CurrentConversationUUID)
			c.CurrentConversationUUID = subReq.UUID
			c.SubscribeConversations(c.ID, []string{subReq.UUID})
		}
	case models.ActionUnsetCurrentConversation:
		c.UnsubscribeConversation(c.ID, c.CurrentConversationUUID)
		c.CurrentConversationUUID = ""
	default:
		c.SendError("unknown action")
	}
}

// close closes the client connection and removes all subscriptions.
func (c *Client) close() {
	c.RemoveAllUserConversationSubscriptions(c.ID)
	c.Closed.Set(true)
	close(c.Send)
}

// SubscribeConversations subscribes the client to the specified conversations.
func (c *Client) SubscribeConversations(userID int, conversationUUIDs []string) {
	for _, conversationUUID := range conversationUUIDs {
		// Initialize the slice if it doesn't exist
		if c.Hub.conversationSubs[conversationUUID] == nil {
			c.Hub.conversationSubs[conversationUUID] = []int{}
		}

		// Check if userID already exists
		exists := false
		for _, id := range c.Hub.conversationSubs[conversationUUID] {
			if id == userID {
				exists = true
				break
			}
		}

		// Add userID if it doesn't exist
		if !exists {
			c.Hub.conversationSubs[conversationUUID] = append(c.Hub.conversationSubs[conversationUUID], userID)
		}
	}
}

// RemoveAllUserConversationSubscriptions removes all conversation subscriptions for the user.
func (c *Client) RemoveAllUserConversationSubscriptions(userID int) {
	for conversationUUID, userIDs := range c.Hub.conversationSubs {
		for i, id := range userIDs {
			if id == userID {
				c.Hub.conversationSubs[conversationUUID] = append(userIDs[:i], userIDs[i+1:]...)
				break
			}
		}
		if len(c.Hub.conversationSubs[conversationUUID]) == 0 {
			delete(c.Hub.conversationSubs, conversationUUID)
		}
	}
}

// UnsubscribeConversation unsubscribes the user from the specified conversation.
func (c *Client) UnsubscribeConversation(userID int, conversationUUID string) {
	if userIDs, ok := c.Hub.conversationSubs[conversationUUID]; ok {
		for i, id := range userIDs {
			if id == userID {
				c.Hub.conversationSubs[conversationUUID] = append(userIDs[:i], userIDs[i+1:]...)
				break
			}
		}
		// Remove the conversation from the map if no users are subscribed
		if len(c.Hub.conversationSubs[conversationUUID]) == 0 {
			delete(c.Hub.conversationSubs, conversationUUID)
		}
	}
}

// SendError sends an error message to client.
func (c *Client) SendError(msg string) {
	out := models.Message{
		Type: models.MessageTypeError,
		Data: msg,
	}
	b, _ := json.Marshal(out)

	// Try to send the error message over the Send channel
	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
		log.Println("Client send channel is full. Could not send error message.")
		c.Hub.RemoveClient(c)
		c.close()
	}
}

// SendMessage sends a message to client.
func (c *Client) SendMessage(b []byte, typ byte) {
	if c.Closed.Get() {
		log.Println("Attempted to send message to closed client")
		return
	}
	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
	}
}
