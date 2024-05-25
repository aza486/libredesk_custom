package main

import (
	"log"

	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/conversations"
	"github.com/abhinavxd/artemis/internal/initz"
	"github.com/abhinavxd/artemis/internal/media"
	"github.com/abhinavxd/artemis/internal/tags"
	user "github.com/abhinavxd/artemis/internal/userdb"
	"github.com/knadh/koanf/v2"
	"github.com/valyala/fasthttp"
	"github.com/vividvilla/simplesessions"
	"github.com/zerodha/fastcache/v4"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko = koanf.New(".")
)

// App is the global app context which is passed and injected everywhere.
type App struct {
	constants     consts
	lo            *logf.Logger
	conversations *conversations.Conversations
	userDB        *user.UserDB
	sess          *simplesessions.Manager
	mediaManager  *media.Manager
	tags          *tags.Tags
	cannedResp    *cannedresp.CannedResp
	fc            *fastcache.FastCache
}

func main() {
	// Command line flags.
	initFlags()

	// Load the config file into Koanf.
	initz.Config(ko)

	lo := initz.Logger(ko.MustString("app.log_level"), ko.MustString("app.env"))
	rd := initz.Redis(ko)
	db := initz.DB(ko)

	// Init the app.
	var app = &App{
		lo:            &lo,
		constants:     initConstants(ko),
		conversations: initConversations(db, &lo, ko),
		sess:          initSessionManager(rd, ko),
		userDB:        initUserDB(db, &lo, ko),
		mediaManager:  initMediaManager(ko, db),
		tags:          initTags(db, &lo),
		cannedResp:    initCannedResponse(db, &lo),
	}

	// HTTP server.
	g := fastglue.NewGlue()
	g.SetContext(app)

	// Handlers.
	initHandlers(g, app, ko)

	s := &fasthttp.Server{
		Name:                 ko.MustString("app.server.name"),
		ReadTimeout:          ko.MustDuration("app.server.read_timeout"),
		WriteTimeout:         ko.MustDuration("app.server.write_timeout"),
		MaxRequestBodySize:   ko.MustInt("app.server.max_body_size"),
		MaxKeepaliveDuration: ko.MustDuration("app.server.keepalive_timeout"),
		ReadBufferSize:       ko.MustInt("app.server.max_body_size"),
	}

	// Start the HTTP server
	log.Printf("server listening on %s %s", ko.String("app.server.address"), ko.String("app.server.socket"))
	if err := g.ListenAndServe(ko.String("app.server.address"), ko.String("server.socket"), s); err != nil {
		log.Fatalf("error starting frontend server: %v", err)
	}
	log.Println("bye")
}
