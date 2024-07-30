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
	// SubscribeConversations to the last 1000 conversations.
	// TODO: Move to config.
	maxConversationsPagesToSub = 10
	maxConversationsPageSize   = 100
)

// SafeBool is a thread-safe boolean.
type SafeBool struct {
	flag bool
	mu   sync.Mutex
}

// Set sets the value of the SafeBool.
func (b *SafeBool) Set(value bool) {
	b.mu.Lock()
	b.flag = value
	b.mu.Unlock()
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

	// Buffered channel of outbound messages.
	Send chan Message
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
			c.Conn.WriteMessage(msg.messageType, msg.data)
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
			fmt.Printf("closing channel due to invalid message type. %d \n", msgType)
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
		return
	}

	switch req.Action {
	case models.ActionConversationsSub:
		var subReq models.ConversationsSubscribe
		if err := json.Unmarshal(data, &subReq); err != nil {
			return
		}

		// First remove all user conversation subscriptions.
		c.RemoveAllUserConversationSubscriptions(c.ID)

		// Add the new subscriptions.
		for page := 1; page <= maxConversationsPagesToSub; page++ {
			conversationUUIDs, err := c.Hub.conversationStore.GetConversationUUIDs(c.ID, page, maxConversationsPageSize, subReq.Type, subReq.Filter)
			if err != nil {
				log.Println("error fetching conversation ids", err)
				continue
			}
			c.SubscribeConversations(c.ID, conversationUUIDs)
		}
	case models.ActionConversationSub:
		var subReq models.ConversationSubscribe
		if err := json.Unmarshal(data, &subReq); err != nil {
			return
		}
		c.SubscribeConversations(c.ID, []string{subReq.UUID})
	case models.ActionConversationUnSub:
		var unsubReq models.ConversationUnsubscribe
		if err := json.Unmarshal(data, &unsubReq); err != nil {
			return
		}
		c.UnsubscribeConversation(c.ID, unsubReq.UUID)
	default:
		fmt.Println("new incoming websocket message ", string(data))
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

// UnsubscribeConversation unsubscribes the client from the specified conversation.
func (c *Client) UnsubscribeConversation(userID int, conversationUUID string) {
	if userIDs, ok := c.Hub.conversationSubs[conversationUUID]; ok {
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
