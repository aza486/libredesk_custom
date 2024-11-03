package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	auth_ "github.com/abhinavxd/artemis/internal/auth"
	"github.com/abhinavxd/artemis/internal/authz"
	notifier "github.com/abhinavxd/artemis/internal/notification"

	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
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
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	constant     constants
	fs           stuffbin.FileSystem
	auth         *auth_.Auth
	authz        *authz.Enforcer
	i18n         *i18n.I18n
	lo           *logf.Logger
	oidc         *oidc.Manager
	media        *media.Manager
	setting      *setting.Manager
	role         *role.Manager
	contact      *contact.Manager
	user         *user.Manager
	team         *team.Manager
	status       *status.Manager
	priority     *priority.Manager
	tag          *tag.Manager
	inbox        *inbox.Manager
	tmpl         *template.Manager
	cannedResp   *cannedresp.Manager
	conversation *conversation.Manager
	automation   *automation.Engine
	notifier     *notifier.Service
}

func main() {
	// Load command line flags into Koanf.
	initFlags()

	// Load the config files into Koanf.
	initConfig(ko)

	// Init stuffbin fs.
	fs := initFS()

	// Init DB.
	db := initDB()

	// Installer.
	if ko.Bool("install") {
		install(db, fs)
		os.Exit(0)
	}

	// Set system user password.
	if ko.Bool("set-system-user-password") {
		setSystemUserPass(db)
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
		ctx, _                      = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		wsHub                       = ws.NewHub()
		rdb                         = initRedis()
		constants                   = initConstants()
		i18n                        = initI18n(fs)
		lo                          = initLogger("artemis")
		oidc                        = initOIDC(db)
		auth                        = initAuth(oidc, rdb)
		template                    = initTemplate(db, fs, constants)
		media                       = initMedia(db)
		contact                     = initContact(db)
		inbox                       = initInbox(db)
		team                        = initTeam(db)
		user                        = initUser(i18n, db)
		notifier                    = initNotifier(user)
		automation                  = initAutomationEngine(db, user)
		conversation                = initConversations(i18n, wsHub, notifier, db, contact, inbox, user, team, media, automation, template)
		autoassigner                = initAutoAssigner(team, user, conversation)
	)

	// Register all active inboxes with inbox manager.
	registerInboxes(inbox, conversation)

	// Set stores.
	wsHub.SetConversationStore(conversation)
	automation.SetConversationStore(conversation)

	// Start receivers for each inbox.
	go inbox.Start(ctx)

	// Start evaluating automation rules.
	go automation.Run(ctx, automationWrk)

	// Start conversation auto assigner.
	go autoassigner.Run(ctx)

	// Start listening and dispatching messages.
	go conversation.Run(ctx, messageDispatchWrk, messageDispatchScanInterval)

	// Start notification service.
	go notifier.Run(ctx)

	// Init the app
	var app = &App{
		lo:           lo,
		auth:         auth,
		fs:           fs,
		i18n:         i18n,
		media:        media,
		setting:      settings,
		contact:      contact,
		inbox:        inbox,
		user:         user,
		team:         team,
		tmpl:         template,
		conversation: conversation,
		automation:   automation,
		oidc:         oidc,
		constant:     constants,
		notifier:     notifier,
		authz:        initAuthz(),
		status:       initStatus(db),
		priority:     initPriority(db),
		role:         initRole(db),
		tag:          initTags(db),
		cannedResp:   initCannedResponse(db),
	}

	// Init fastglue and set app in ctx.
	g := fastglue.NewGlue()
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

	log.Printf("%s🚀 server listening on %s %s\x1b[0m", "\x1b[32m", ko.String("app.server.address"), ko.String("app.server.socket"))

	go func() {
		if err := g.ListenAndServe(ko.String("app.server.address"), ko.String("server.socket"), s); err != nil {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("%sShutting down the server. Please wait.\x1b[0m", "\x1b[31m")
	// Shutdown HTTP server.
	s.Shutdown()
	// Shutdown services.
	inbox.Close()
	automation.Close()
	autoassigner.Close()
	notifier.Close()
	conversation.Close()
	db.Close()
	rdb.Close()
}
