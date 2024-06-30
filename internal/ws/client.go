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
	// SubscribeConversations to last 1000 conversations.
	// TODO: Move to config.
	maxConversationsPagesToSub = 10
	maxConversationsPageSize   = 100
)

type SafeBool struct {
	flag bool
	mu   sync.Mutex
}

func (b *SafeBool) Set(value bool) {
	b.mu.Lock()
	b.flag = value
	b.mu.Unlock()
}

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

	// Ws Conn.
	Conn *websocket.Conn

	// To prevent pushes to the channel.
	Closed SafeBool

	// Buffered channel of outbound messages.
	Send chan Message
}

func (c *Client) Serve(heartBeatDuration time.Duration) {
	var heartBeatTicker = time.NewTicker(heartBeatDuration)
	defer heartBeatTicker.Stop()
Loop:
	for {
		select {
		case <-heartBeatTicker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("error writing msg", err)
				return
			}
		case o, ok := <-c.Send:
			if !ok {
				break Loop
			}
			c.Conn.WriteMessage(o.messageType, o.data)
		}
	}
	c.Conn.Close()
}

func (c *Client) Listen() {
	for {
		t, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if t == websocket.TextMessage {
			c.processIncomingMessage(msg)
		} else {
			fmt.Printf("closing chan invalid msg type. %d \n", t)
			c.Hub.RemoveClient(c)
			c.close()
			return
		}
	}
	c.Hub.RemoveClient(c)
	c.close()
}

// processIncomingMessage processes incoming messages from client.
func (c *Client) processIncomingMessage(b []byte) {
	var r models.IncomingReq

	if err := json.Unmarshal(b, &r); err != nil {
		return
	}

	switch r.Action {
	case models.ActionConversationsSub:
		var req = models.ConversationsSubscribe{}
		if err := json.Unmarshal(b, &req); err != nil {
			return
		}

		// First remove all user conversation subscriptions.
		c.RemoveAllUserConversationSubscriptions(c.ID)

		// Add the new subcriptions.
		for page := range maxConversationsPagesToSub {
			page++
			conversationUUIDs, err := c.Hub.conversationStore.GetConversationUUIDs(c.ID, page, maxConversationsPageSize, req.Type, req.PreDefinedFilter)
			if err != nil {
				log.Println("error fetching convesation ids", err)
				continue
			}
			c.SubscribeConversations(c.ID, conversationUUIDs)
		}
	case models.ActionConversationSub:
		var req = models.ConversationSubscribe{}
		if err := json.Unmarshal(b, &req); err != nil {
			return
		}
		c.SubscribeConversations(c.ID, []string{req.UUID})
	case models.ActionConversationUnSub:
		var req = models.ConversationUnsubscribe{}
		if err := json.Unmarshal(b, &req); err != nil {
			return
		}
		c.UnsubscribeConversation(c.ID, req.UUID)
	default:
		fmt.Println("new incoming ws msg ", string(b))
	}
}

func (c *Client) close() {
	c.RemoveAllUserConversationSubscriptions(c.ID)
	c.Closed.Set(true)
	close(c.Send)
}

func (c *Client) SubscribeConversations(userID int, conversationUUIDs []string) {
	for _, conversationUUID := range conversationUUIDs {
		// Initialize the slice if it doesn't exist
		if c.Hub.ConversationSubs[conversationUUID] == nil {
			c.Hub.ConversationSubs[conversationUUID] = []int{}
		}

		// Check if userID already exists
		exists := false
		for _, id := range c.Hub.ConversationSubs[conversationUUID] {
			if id == userID {
				exists = true
				break
			}
		}

		// Add userID if it doesn't exist
		if !exists {
			c.Hub.ConversationSubs[conversationUUID] = append(c.Hub.ConversationSubs[conversationUUID], userID)
		}
	}
}

func (c *Client) UnsubscribeConversation(userID int, conversationUUID string) {
	if userIDs, ok := c.Hub.ConversationSubs[conversationUUID]; ok {
		for i, id := range userIDs {
			if id == userID {
				c.Hub.ConversationSubs[conversationUUID] = append(userIDs[:i], userIDs[i+1:]...)
				break
			}
		}
		if len(c.Hub.ConversationSubs[conversationUUID]) == 0 {
			delete(c.Hub.ConversationSubs, conversationUUID)
		}
	}
}

func (c *Client) RemoveAllUserConversationSubscriptions(userID int) {
	for conversationID, userIDs := range c.Hub.ConversationSubs {
		for i, id := range userIDs {
			if id == userID {
				c.Hub.ConversationSubs[conversationID] = append(userIDs[:i], userIDs[i+1:]...)
				break
			}
		}
		if len(c.Hub.ConversationSubs[conversationID]) == 0 {
			delete(c.Hub.ConversationSubs, conversationID)
		}
	}
}
