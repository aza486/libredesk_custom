package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	uauth "github.com/abhinavxd/artemis/internal/auth"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/initz"
	"github.com/abhinavxd/artemis/internal/message"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/upload"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/vividvilla/simplesessions"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko = koanf.New(".")
)

const (
	// ANSI escape colour codes.
	colourRed   = "\x1b[31m"
	colourGreen = "\x1b[32m"
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	constants       consts
	fs              stuffbin.FileSystem
	lo              *logf.Logger
	cntctMgr        *contact.Manager
	userMgr         *user.Manager
	teamMgr         *team.Manager
	sessMgr         *simplesessions.Manager
	tagMgr          *tag.Manager
	msgMgr          *message.Manager
	rbac            *uauth.Engine
	inboxMgr        *inbox.Manager
	uploadMgr       *upload.Manager
	attachmentMgr   *attachment.Manager
	cannedRespMgr   *cannedresp.Manager
	conversationMgr *conversation.Manager
}

func main() {
	// Load stuffbin fs.
	fs := initFS()

	// Load command line flags into Koanf.
	initFlags()

	// Load the config files into Koanf.
	initz.Config(ko)

	var (
		shutdownCh         = make(chan struct{})
		ctx, stop          = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		lo                 = initz.Logger(ko.MustString("app.log_level"), ko.MustString("app.env"), "artemis")
		rd                 = initz.Redis(ko)
		db                 = initz.DB(ko)
		wsHub              = ws.NewHub()
		templateMgr        = initTemplateMgr(db)
		attachmentMgr      = initAttachmentsManager(db, lo)
		cntctMgr           = initContactManager(db, lo)
		inboxMgr           = initInboxManager(db, lo)
		teamMgr            = initTeamMgr(db, lo)
		userMgr            = initUserDB(db, lo)
		notifier           = initNotifier(userMgr, templateMgr)
		conversationMgr    = initConversations(wsHub, db, lo)
		automationEngine   = initAutomationEngine(db, lo)
		msgMgr             = initMessages(db, lo, wsHub, userMgr, teamMgr, cntctMgr, attachmentMgr, conversationMgr, inboxMgr, automationEngine, templateMgr)
		autoAssignerEngine = initAutoAssignmentEngine(teamMgr, conversationMgr, msgMgr, notifier, wsHub, lo)
	)

	// Register all inboxes with the inbox manager.
	registerInboxes(inboxMgr, msgMgr)

	// Set conversation store for the websocket hub.
	wsHub.SetConversationStore(conversationMgr)

	// Set stores for the automation engine.
	automationEngine.SetMessageStore(msgMgr)
	automationEngine.SetConversationStore(conversationMgr)

	// Start receivers for all active inboxes.
	inboxMgr.Receive(ctx)

	// Start automation rule evaluation engine.
	go automationEngine.Serve(ctx)

	// Start conversation auto assigner engine.
	go autoAssignerEngine.Serve(ctx, ko.MustDuration("autoassigner.assign_interval"))

	// Start inserting incoming messages from all active inboxes and dispatch pending outgoing messages.
	go msgMgr.StartDBInserts(ctx, ko.MustInt("message.reader_concurrency"))
	go msgMgr.StartDispatcher(ctx, ko.MustInt("message.dispatch_concurrency"), ko.MustDuration("message.dispatch_read_interval"))

	// Init the app
	var app = &App{
		lo:              lo,
		fs:              fs,
		cntctMgr:        cntctMgr,
		inboxMgr:        inboxMgr,
		userMgr:         userMgr,
		teamMgr:         teamMgr,
		attachmentMgr:   attachmentMgr,
		conversationMgr: conversationMgr,
		msgMgr:          msgMgr,
		constants:       initConstants(),
		rbac:            initRBACEngine(db),
		tagMgr:          initTags(db, lo),
		sessMgr:         initSessionManager(rd),
		cannedRespMgr:   initCannedResponse(db, lo),
	}

	// Init fastglue http server.
	g := fastglue.NewGlue()

	// Add app the request context.
	g.SetContext(app)

	// Init the handlers.
	initHandlers(g, wsHub)

	s := &fasthttp.Server{
		Name:                 ko.MustString("app.server.name"),
		ReadTimeout:          ko.MustDuration("app.server.read_timeout"),
		WriteTimeout:         ko.MustDuration("app.server.write_timeout"),
		MaxRequestBodySize:   ko.MustInt("app.server.max_body_size"),
		MaxKeepaliveDuration: ko.MustDuration("app.server.keepalive_timeout"),
		ReadBufferSize:       ko.MustInt("app.server.max_body_size"),
	}

	// Handling graceful shutdown with a delay
	go func() {
		// Wait for the interruption signal
		<-ctx.Done()

		log.Printf("%sShutting down the server. Please wait.\x1b[0m", colourRed)

		time.Sleep(5 * time.Second)

		// Signal to shutdown the server
		shutdownCh <- struct{}{}
		stop()
	}()

	log.Printf("%s🚀 server listening on %s %s\x1b[0m", colourGreen, ko.String("app.server.address"), ko.String("app.server.socket"))

	if err := g.ListenServeAndWaitGracefully(ko.String("app.server.address"), ko.String("server.socket"), s, shutdownCh); err != nil {
		log.Fatalf("error starting frontend server: %v", err)
	}
}
