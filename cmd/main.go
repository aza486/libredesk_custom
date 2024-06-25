package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	convtag "github.com/abhinavxd/artemis/internal/conversation/tag"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/initz"
	"github.com/abhinavxd/artemis/internal/message"
	"github.com/abhinavxd/artemis/internal/rbac"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/user/filterstore"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/knadh/koanf/v2"
	"github.com/valyala/fasthttp"
	"github.com/vividvilla/simplesessions"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var ko = koanf.New(".")

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	constants           consts
	lo                  *logf.Logger
	cntctMgr            *contact.Manager
	userMgr             *user.Manager
	teamMgr             *team.Manager
	sessMgr             *simplesessions.Manager
	tagMgr              *tag.Manager
	msgMgr              *message.Manager
	rbac                *rbac.Engine
	userFilterMgr       *filterstore.Manager
	inboxMgr            *inbox.Manager
	attachmentMgr       *attachment.Manager
	cannedRespMgr       *cannedresp.Manager
	conversationMgr     *conversation.Manager
	conversationTagsMgr *convtag.Manager
}

func main() {
	// Load command line flags into Koanf.
	initFlags()

	// Load the config files into Koanf.
	initz.Config(ko)

	var (
		shutdownCh = make(chan struct{})
		ctx, stop  = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		lo         = initz.Logger(ko.MustString("app.log_level"), ko.MustString("app.env"), "artemis")
		rd         = initz.Redis(ko)
		db         = initz.DB(ko)

		wsHub              = ws.NewHub()
		attachmentMgr      = initAttachmentsManager(db, lo)
		cntctMgr           = initContactManager(db, lo)
		inboxMgr           = initInboxManager(db, lo)
		teamMgr            = initTeamMgr(db, lo)
		userMgr            = initUserDB(db, lo)
		conversationMgr    = initConversations(db, lo)
		automationEngine   = initAutomationEngine(db, lo)
		msgMgr             = initMessages(db, lo, wsHub, userMgr, teamMgr, cntctMgr, attachmentMgr, conversationMgr, inboxMgr, automationEngine)
		autoAssignerEngine = initAutoAssignmentEngine(teamMgr, conversationMgr, msgMgr, lo)
	)

	// Init the app
	var app = &App{
		lo:                  lo,
		cntctMgr:            cntctMgr,
		inboxMgr:            inboxMgr,
		userMgr:             userMgr,
		teamMgr:             teamMgr,
		attachmentMgr:       attachmentMgr,
		conversationMgr:     conversationMgr,
		msgMgr:              msgMgr,
		constants:           initConstants(),
		rbac:                initRBACEngine(db),
		tagMgr:              initTags(db, lo),
		userFilterMgr:       initUserFilterMgr(db),
		sessMgr:             initSessionManager(rd),
		cannedRespMgr:       initCannedResponse(db, lo),
		conversationTagsMgr: initConversationTags(db, lo),
	}

	// Register all inboxes with the inbox manager.
	registerInboxes(inboxMgr, msgMgr)

	automationEngine.SetMsgRecorder(msgMgr)
	automationEngine.SetConvUpdater(conversationMgr)

	// Start receivers for all active inboxes.
	inboxMgr.Receive(ctx)

	// Start inserting incoming msgs and dispatch pending outgoing messages.
	go app.msgMgr.StartDBInserts(ctx, ko.MustInt("message.reader_concurrency"))
	go app.msgMgr.StartDispatcher(ctx, ko.MustInt("message.dispatch_concurrency"), ko.MustDuration("message.dispatch_read_interval"))

	// Start automation rule engine.
	go automationEngine.Serve(ctx)

	// Start conversation auto assigner engine.
	go autoAssignerEngine.Serve(ctx, ko.MustDuration("autoassigner.assign_interval"))

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

		log.Printf("\x1b[%dm%s\x1b[0m", 31, "Shutting down the server please wait...")

		// Additional grace period before triggering shutdown
		time.Sleep(7 * time.Second)

		// Signal to shutdown the server
		shutdownCh <- struct{}{}
		stop()
	}()

	// Starting the server and waiting for the shutdown signal
	log.Printf("🚀 server listening on %s %s", ko.String("app.server.address"), ko.String("app.server.socket"))
	if err := g.ListenServeAndWaitGracefully(ko.String("app.server.address"), ko.String("server.socket"), s, shutdownCh); err != nil {
		log.Fatalf("error starting frontend server: %v", err)
	}
	log.Println("Server shutdown completed")
}
