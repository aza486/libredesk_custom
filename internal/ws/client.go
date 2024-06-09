package ws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/ws/models"
	"github.com/fasthttp/websocket"
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
				// Disconnected.
				fmt.Println("Client disconnected, breaking serve lop.")
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

	fmt.Println("loop broke closing")
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
	// Sub to conversation updates.
	case models.ActionConvSub:
		var subR = models.ConvSubUnsubReq{}
		if err := json.Unmarshal(b, &subR); err != nil {
			return
		}
		c.SubConv(int(c.ID), subR.UUIDs...)
	case models.ActionConvUnsub:
		var subR = models.ConvSubUnsubReq{}
		if err := json.Unmarshal(b, &subR); err != nil {
			return
		}
		c.UnsubConv(int(c.ID), subR.UUIDs...)
	case models.ActionAssignedConvSub:
		// Fetch all assigned conversation & sub.
	case models.ActionAssignedConvUnSub:
		// Fetch all unassigned conversation and sub.
	}
}

func (c *Client) close() {
	c.Closed.Set(true)
	close(c.Send)
}

func (c *Client) SubConv(userID int, uuids ...string) {
	c.Hub.SubMut.Lock()
	defer c.Hub.SubMut.Unlock()

	for _, uuid := range uuids {
		// Initialize the slice if this is the first subscription for this UUID
		if _, ok := c.Hub.Csubs[uuid]; !ok {
			c.Hub.Csubs[uuid] = []int{}
		}
		// Append the user ID to the slice of subscribed user IDs
		c.Hub.Csubs[uuid] = append(c.Hub.Csubs[uuid], userID)
	}
}

func (c *Client) UnsubConv(userID int, uuids ...string) {
	c.Hub.SubMut.Lock()
	defer c.Hub.SubMut.Unlock()

	for _, uuid := range uuids {
		currentSubs, ok := c.Hub.Csubs[uuid]
		if !ok {
			continue // No subscriptions for this UUID
		}
		j := 0
		for _, sub := range currentSubs {
			if sub != userID {
				currentSubs[j] = sub
				j++
			}
		}
		currentSubs = currentSubs[:j] // Update the slice in-place
		if len(currentSubs) == 0 {
			delete(c.Hub.Csubs, uuid) // Remove key if no more subscriptions
		} else {
			c.Hub.Csubs[uuid] = currentSubs
		}
	}
}
