package main

import (
	"fmt"

	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/abhinavxd/artemis/internal/ws"
	wsmodels "github.com/abhinavxd/artemis/internal/ws/models"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func ErrHandler(ctx *fasthttp.RequestCtx, status int, reason error) {
	fmt.Printf("error status %d: %s", status, reason)
}

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
	Error: ErrHandler,
}

func handleWS(r *fastglue.Request, hub *ws.Hub) error {
	var (
		user = r.RequestCtx.UserValue("user").(umodels.User)
		app  = r.Context.(*App)
	)
	err := upgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		c := ws.Client{
			ID:   user.ID,
			Hub:  hub,
			Conn: conn,
			Send: make(chan wsmodels.WSMessage, 1000),
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
