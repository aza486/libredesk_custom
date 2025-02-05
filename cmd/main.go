package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/abhinavxd/libredesk/internal/ai"
	auth_ "github.com/abhinavxd/libredesk/internal/auth"
	"github.com/abhinavxd/libredesk/internal/authz"
	businesshours "github.com/abhinavxd/libredesk/internal/business_hours"
	"github.com/abhinavxd/libredesk/internal/colorlog"
	"github.com/abhinavxd/libredesk/internal/csat"
	"github.com/abhinavxd/libredesk/internal/macro"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/search"
	"github.com/abhinavxd/libredesk/internal/sla"
	"github.com/abhinavxd/libredesk/internal/view"

	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/conversation/priority"
	"github.com/abhinavxd/libredesk/internal/conversation/status"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/oidc"
	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/tag"
	"github.com/abhinavxd/libredesk/internal/team"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/knadh/go-i18n"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

var (
	ko          = koanf.New(".")
	ctx         = context.Background()
	appName     = "libredesk"
	frontendDir = "frontend/dist"

	// Injected at build time.
	buildString = ""
)

// App is the global app context which is passed and injected in the http handlers.
type App struct {
	fs            stuffbin.FileSystem
	consts        atomic.Value
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
	macro         *macro.Manager
	conversation  *conversation.Manager
	automation    *automation.Engine
	businessHours *businesshours.Manager
	sla           *sla.Manager
	csat          *csat.Manager
	view          *view.Manager
	ai            *ai.Manager
	search        *search.Manager
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

	colorlog.Green("Build: %s", buildString)

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
		autoAssignInterval          = ko.MustDuration("autoassigner.autoassign_interval")
		unsnoozeInterval            = ko.MustDuration("conversation.unsnooze_interval")
		automationWorkers           = ko.MustInt("automation.worker_count")
		messageOutgoingQWorkers     = ko.MustDuration("message.outgoing_queue_workers")
		messageIncomingQWorkers     = ko.MustDuration("message.incoming_queue_workers")
		messageOutgoingScanInterval = ko.MustDuration("message.message_outoing_scan_interval")
		slaEvaluationInterval       = ko.MustDuration("sla.evaluation_interval")
		lo                          = initLogger(appName)
		wsHub                       = ws.NewHub()
		rdb                         = initRedis()
		constants                   = initConstants()
		i18n                        = initI18n(fs)
		csat                        = initCSAT(db)
		oidc                        = initOIDC(db, settings)
		status                      = initStatus(db)
		priority                    = initPriority(db)
		auth                        = initAuth(oidc, rdb)
		template                    = initTemplate(db, fs, constants)
		media                       = initMedia(db)
		inbox                       = initInbox(db)
		team                        = initTeam(db)
		businessHours               = initBusinessHours(db)
		user                        = initUser(i18n, db)
		notifier                    = initNotifier(user)
		automation                  = initAutomationEngine(db)
		sla                         = initSLA(db, team, settings, businessHours)
		conversation                = initConversations(i18n, sla, status, priority, wsHub, notifier, db, inbox, user, team, media, settings, csat, automation, template)
		autoassigner                = initAutoAssigner(team, user, conversation)
	)

	automation.SetConversationStore(conversation)
	startInboxes(ctx, inbox, conversation)
	go automation.Run(ctx, automationWorkers)
	go autoassigner.Run(ctx, autoAssignInterval)
	go conversation.Run(ctx, messageIncomingQWorkers, messageOutgoingQWorkers, messageOutgoingScanInterval)
	go conversation.RunUnsnoozer(ctx, unsnoozeInterval)
	go notifier.Run(ctx)
	go sla.Run(ctx, slaEvaluationInterval)
	go media.DeleteUnlinkedMedia(ctx)

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
		status:        status,
		priority:      priority,
		tmpl:          template,
		notifier:      notifier,
		consts:        atomic.Value{},
		conversation:  conversation,
		automation:    automation,
		businessHours: businessHours,
		authz:         initAuthz(),
		view:          initView(db),
		csat:          initCSAT(db),
		search:        initSearch(db),
		role:          initRole(db),
		tag:           initTag(db),
		macro:         initMacro(db),
		ai:            initAI(db),
	}
	app.consts.Store(constants)

	g := fastglue.NewGlue()
	g.SetContext(app)
	initHandlers(g, wsHub)

	s := &fasthttp.Server{
		Name:                 appName,
		ReadTimeout:          ko.MustDuration("app.server.read_timeout"),
		WriteTimeout:         ko.MustDuration("app.server.write_timeout"),
		MaxRequestBodySize:   ko.MustInt("app.server.max_body_size"),
		MaxKeepaliveDuration: ko.MustDuration("app.server.keepalive_timeout"),
		ReadBufferSize:       ko.MustInt("app.server.max_body_size"),
	}

	go func() {
		colorlog.Green("Server started at %s", ko.String("app.server.address"))
		if ko.String("server.socket") != "" {
			colorlog.Green("Unix socket created at %s", ko.String("server.socket"))
		}
		if err := g.ListenAndServe(ko.String("app.server.address"), ko.String("server.socket"), s); err != nil {
			log.Fatalf("error starting server: %v", err)
		}
	}()

	// Wait for shutdown signal.
	<-ctx.Done()
	colorlog.Red("Shutting down HTTP server...")
	s.Shutdown()
	colorlog.Red("Shutting down inboxes...")
	inbox.Close()
	colorlog.Red("Shutting down automation...")
	automation.Close()
	colorlog.Red("Shutting down autoassigner...")
	autoassigner.Close()
	colorlog.Red("Shutting down notifier...")
	notifier.Close()
	colorlog.Red("Shutting down conversation...")
	conversation.Close()
	colorlog.Red("Shutting down SLA...")
	sla.Close()
	colorlog.Red("Shutting down database...")
	db.Close()
	colorlog.Red("Shutting down redis...")
	rdb.Close()
	colorlog.Green("Shutdown complete.")
}
