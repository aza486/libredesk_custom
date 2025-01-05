package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	auth_ "github.com/abhinavxd/artemis/internal/auth"
	"github.com/abhinavxd/artemis/internal/authz"
	businesshours "github.com/abhinavxd/artemis/internal/business_hours"
	"github.com/abhinavxd/artemis/internal/colorlog"
	"github.com/abhinavxd/artemis/internal/csat"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	"github.com/abhinavxd/artemis/internal/sla"
	"github.com/abhinavxd/artemis/internal/view"

	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/conversation/priority"
	"github.com/abhinavxd/artemis/internal/conversation/status"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/media"
	"github.com/abhinavxd/artemis/internal/oidc"
	"github.com/abhinavxd/artemis/internal/role"
	"github.com/abhinavxd/artemis/internal/setting"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/template"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/knadh/go-i18n"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko          = koanf.New(".")
	frontendDir = "frontend/dist"
	ctx         = context.Background()
	buildString string
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	consts        constants
	fs            stuffbin.FileSystem
	auth          *auth_.Auth
	authz         *authz.Enforcer
	i18n          *i18n.I18n
	lo            *logf.Logger
	oidc          *oidc.Manager
	media         *media.Manager
	setting       *setting.Manager
	role          *role.Manager
	user          *user.Manager
	team          *team.Manager
	status        *status.Manager
	priority      *priority.Manager
	tag           *tag.Manager
	inbox         *inbox.Manager
	tmpl          *template.Manager
	cannedResp    *cannedresp.Manager
	conversation  *conversation.Manager
	automation    *automation.Engine
	businessHours *businesshours.Manager
	sla           *sla.Manager
	csat          *csat.Manager
	view          *view.Manager
	notifier      *notifier.Service
}

func main() {
	// Set up signal handler.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load command line flags into Koanf.
	initFlags()

	// Load the config files into Koanf.
	initConfig(ko)

	// Init stuffbin fs.
	fs := initFS()

	// Init DB.
	db := initDB()

	// Version flag.
	if ko.Bool("version") {
		fmt.Println(buildString)
		os.Exit(0)
	}

	log.Printf("Build: %s", buildString)

	// Installer.
	if ko.Bool("install") {
		install(ctx, db, fs)
		os.Exit(0)
	}

	// Set system user password.
	if ko.Bool("set-system-user-password") {
		setSystemUserPass(ctx, db)
		os.Exit(0)
	}

	// Check if schema is installed.
	installed, err := checkSchema(db)
	if err != nil {
		log.Fatalf("error checking db schema: %v", err)
	}
	if !installed {
		log.Println("Database tables are missing. Use the `--install` flag to set up the database schema.")
		os.Exit(0)
	}

	// Load app settings from DB into the Koanf instance.
	settings := initSettings(db)
	loadSettings(settings)

	var (
		automationWrk               = ko.MustInt("automation.worker_count")
		messageDispatchWrk          = ko.MustInt("message.dispatch_workers")
		messageDispatchScanInterval = ko.MustDuration("message.dispatch_scan_interval")
		lo                          = initLogger("artemis")
		wsHub                       = ws.NewHub()
		rdb                         = initRedis()
		constants                   = initConstants()
		i18n                        = initI18n(fs)
		oidc                        = initOIDC(db)
		auth                        = initAuth(oidc, rdb)
		template                    = initTemplate(db, fs, constants)
		media                       = initMedia(db)
		inbox                       = initInbox(db)
		team                        = initTeam(db)
		businessHours               = initBusinessHours(db)
		user                        = initUser(i18n, db)
		notifier                    = initNotifier(user)
		automation                  = initAutomationEngine(db, user)
		sla                         = initSLA(db, team, settings, businessHours)
		conversation                = initConversations(i18n, wsHub, notifier, db, inbox, user, team, media, automation, template)
		autoassigner                = initAutoAssigner(team, user, conversation)
	)

	// Set stores.
	wsHub.SetConversationStore(conversation)
	automation.SetConversationStore(conversation, sla)

	// Start inbox receivers.
	startInboxes(ctx, inbox, conversation)

	// Start evaluating automation rules.
	go automation.Run(ctx, automationWrk)

	// Start conversation auto assigner.
	go autoassigner.Run(ctx)

	// Start processing incoming and outgoing messages.
	go conversation.Run(ctx, messageDispatchWrk, messageDispatchScanInterval)

	// Start notifier.
	go notifier.Run(ctx)

	// Start SLA monitor.
	go sla.Run(ctx)

	// Purge unlinked message media.
	go media.DeleteUnlinkedMessageMedia(ctx)

	// Init the app
	var app = &App{
		lo:            lo,
		fs:            fs,
		sla:           sla,
		oidc:          oidc,
		i18n:          i18n,
		auth:          auth,
		media:         media,
		setting:       settings,
		inbox:         inbox,
		user:          user,
		team:          team,
		tmpl:          template,
		notifier:      notifier,
		consts:        constants,
		conversation:  conversation,
		automation:    automation,
		businessHours: businessHours,
		view:          initView(db),
		csat:          initCSAT(db),
		authz:         initAuthz(),
		status:        initStatus(db),
		priority:      initPriority(db),
		role:          initRole(db),
		tag:           initTag(db),
		cannedResp:    initCannedResponse(db),
	}

	// Init fastglue and set app in ctx.
	g := fastglue.NewGlue()

	// Set the app in context.
	g.SetContext(app)

	// Init HTTP handlers.
	initHandlers(g, wsHub)

	s := &fasthttp.Server{
		Name:                 "server",
		ReadTimeout:          ko.MustDuration("app.server.read_timeout"),
		WriteTimeout:         ko.MustDuration("app.server.write_timeout"),
		MaxRequestBodySize:   ko.MustInt("app.server.max_body_size"),
		MaxKeepaliveDuration: ko.MustDuration("app.server.keepalive_timeout"),
		ReadBufferSize:       ko.MustInt("app.server.max_body_size"),
	}

	go func() {
		if err := g.ListenAndServe(ko.String("app.server.address"), ko.String("server.socket"), s); err != nil {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	colorlog.Green("🚀 server listening on %s %s", ko.String("app.server.address"), ko.String("app.server.socket"))

	// Wait for shutdown signal.
	<-ctx.Done()
	colorlog.Red("Shutting down the server. Please wait....")
	// Shutdown HTTP server.
	s.Shutdown()
	colorlog.Red("Server shutdown complete.")
	colorlog.Red("Shutting down services. Please wait....")
	// Shutdown services.
	inbox.Close()
	colorlog.Red("Inbox shutdown complete.")
	automation.Close()
	colorlog.Red("Automation shutdown complete.")
	autoassigner.Close()
	colorlog.Red("Autoassigner shutdown complete.")
	notifier.Close()
	colorlog.Red("Notifier shutdown complete.")
	conversation.Close()
	colorlog.Red("Conversation shutdown complete.")
	sla.Close()
	colorlog.Red("SLA shutdown complete.")
	db.Close()
	colorlog.Red("Database shutdown complete.")
	rdb.Close()
	colorlog.Red("Redis shutdown complete.")
	colorlog.Green("Shutdown complete.")
}
