package main

import (
	"fmt"

	umodels "github.com/abhinavxd/artemis/internal/user/models"
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
		user = r.RequestCtx.UserValue("user").(umodels.User)
		app    = r.Context.(*App)
	)

	err := upgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		c := ws.Client{
			ID:   user.ID,
			Hub:  hub,
			Conn: conn,
			Send: make(chan ws.Message, 10000),
		}
		hub.AddClient(&c)
		go c.Listen()
		c.Serve()
	})
	if err != nil {
		app.lo.Error("error upgrading tcp connection", "error", err)
	}
	return nil
}
