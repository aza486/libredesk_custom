package main

import (
	"fmt"
	"time"

	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func ErrHandler(ctx *fasthttp.RequestCtx, status int, reason error) {
	fmt.Printf("error status %d - error %d", status, reason)
}

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true // Allow all origins in development
	},
	Error: ErrHandler,
}

func handleWS(r *fastglue.Request, hub *ws.Hub) error {
	var (
		userID = r.RequestCtx.UserValue("user_id").(int)
		app    = r.Context.(*App)
	)

	err := upgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		c := ws.Client{
			ID:   userID,
			Hub:  hub,
			Conn: conn,
			Send: make(chan ws.Message, 100000),
		}

		// Sub this client to all assigned conversations.
		convs, err := app.conversationMgr.GetAssignedConversations(userID)
		if err != nil {
			return
		}
		// Extract  uuids.
		uuids := make([]string, len(convs))
		for i, conv := range convs {
			uuids[i] = conv.UUID
		}
		c.SubConv(userID, uuids...)

		hub.AddClient(&c)

		go c.Listen()
		c.Serve(2 * time.Second)
	})
	if err != nil {
		app.lo.Error("error upgrading tcp connection", "error", err)
	}
	return nil
}
