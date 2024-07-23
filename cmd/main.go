package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/message"
	"github.com/abhinavxd/artemis/internal/role"
	"github.com/abhinavxd/artemis/internal/setting"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/upload"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/knadh/go-i18n"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
	"github.com/zerodha/simplesessions/v3"
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
	constants           consts
	fs                  stuffbin.FileSystem
	rdb                 *redis.Client
	i18n                *i18n.I18n
	lo                  *logf.Logger
	settingsManager     *setting.Manager
	roleManager         *role.Manager
	contactManager      *contact.Manager
	userManager         *user.Manager
	teamManager         *team.Manager
	sessManager         *simplesessions.Manager
	tagManager          *tag.Manager
	messageManager      *message.Manager
	inboxManager        *inbox.Manager
	uploadManager       *upload.Manager
	attachmentManager   *attachment.Manager
	cannedRespManager   *cannedresp.Manager
	conversationManager *conversation.Manager
	automationEngine    *automation.Engine
}

func main() {
	// Load command line flags into Koanf.
	initFlags()

	// Load the config files into Koanf.
	initConfig(ko)

	// Load app settings into Koanf.
	db := initDB()
	settingManager := initSettingsManager(db)
	loadSettings(settingManager)

	var (
		shutdownCh          = make(chan struct{})
		ctx, stop           = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		wsHub               = ws.NewHub()
		fs                  = initFS()
		i18n                = initI18n(fs)
		lo                  = initLogger("artemis")
		rdb                 = initRedis()
		templateManager     = initTemplateManager(db)
		attachmentManager   = initAttachmentsManager(db)
		contactManager      = initContactManager(db)
		inboxManager        = initInboxManager(db)
		teamManager         = initTeamManager(db)
		userManager         = initUserManager(i18n, db)
		notifier            = initNotifier(userManager, templateManager)
		conversationManager = initConversations(i18n, wsHub, notifier, db)
		automationEngine    = initAutomationEngine(db, userManager)
		messageManager      = initMessages(db, wsHub, userManager, teamManager, contactManager, attachmentManager, conversationManager, inboxManager, automationEngine, templateManager)
		autoassigner        = initAutoAssigner(teamManager, conversationManager)
	)

	// Set message store for conversation manager.
	conversationManager.SetMessageStore(messageManager)

	// Register all inboxes with inbox manager & start receiving messages.
	registerInboxes(inboxManager, messageManager)
	inboxManager.Receive(ctx)

	// Set conversation store for websocket hub.
	wsHub.SetConversationStore(conversationManager)

	// Set stores for automation engine & start the evaluating rules.
	automationEngine.SetConversationStore(conversationManager)
	go automationEngine.Serve(ctx)

	// Start conversation auto assigner.
	go autoassigner.Serve(ctx, ko.MustDuration("autoassigner.assign_interval"))

	// Start inserting incoming messages from all active inboxes and dispatch pending outgoing messages.
	go messageManager.StartDBInserts(ctx, ko.MustInt("message.reader_concurrency"))
	go messageManager.StartDispatcher(ctx, ko.MustInt("message.dispatch_concurrency"), ko.MustDuration("message.dispatch_read_interval"))

	// Init the app
	var app = &App{
		lo:                  lo,
		rdb:                 rdb,
		fs:                  fs,
		i18n:                i18n,
		settingsManager:     settingManager,
		contactManager:      contactManager,
		inboxManager:        inboxManager,
		userManager:         userManager,
		teamManager:         teamManager,
		attachmentManager:   attachmentManager,
		conversationManager: conversationManager,
		messageManager:      messageManager,
		automationEngine:    automationEngine,
		roleManager:         initRoleManager(db),
		constants:           initConstants(),
		tagManager:          initTags(db),
		sessManager:         initSessionManager(rdb),
		cannedRespManager:   initCannedResponse(db),
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

	// Graceful shutdown
	go func() {
		// Wait for the interruption signal
		<-ctx.Done()

		log.Printf("%sShutting down the server. Please wait.\x1b[0m", colourRed)

		time.Sleep(1 * time.Second)

		// Signal to shutdown the server
		shutdownCh <- struct{}{}
		stop()
	}()

	log.Printf("%s🚀 server listening on %s %s\x1b[0m", colourGreen, ko.String("app.server.address"), ko.String("app.server.socket"))

	if err := g.ListenServeAndWaitGracefully(ko.String("app.server.address"), ko.String("server.socket"), s, shutdownCh); err != nil {
		log.Fatalf("error starting frontend server: %v", err)
	}
}
