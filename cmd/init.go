package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"html/template"

	auth_ "github.com/abhinavxd/artemis/internal/auth"
	"github.com/abhinavxd/artemis/internal/authz"
	"github.com/abhinavxd/artemis/internal/autoassigner"
	"github.com/abhinavxd/artemis/internal/automation"
	businesshours "github.com/abhinavxd/artemis/internal/business_hours"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/conversation/priority"
	"github.com/abhinavxd/artemis/internal/conversation/status"
	"github.com/abhinavxd/artemis/internal/csat"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/inbox/channel/email"
	imodels "github.com/abhinavxd/artemis/internal/inbox/models"
	"github.com/abhinavxd/artemis/internal/media"
	fs "github.com/abhinavxd/artemis/internal/media/stores/localfs"
	"github.com/abhinavxd/artemis/internal/media/stores/s3"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	emailnotifier "github.com/abhinavxd/artemis/internal/notification/providers/email"
	"github.com/abhinavxd/artemis/internal/oidc"
	"github.com/abhinavxd/artemis/internal/role"
	"github.com/abhinavxd/artemis/internal/setting"
	"github.com/abhinavxd/artemis/internal/sla"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	tmpl "github.com/abhinavxd/artemis/internal/template"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/view"
	"github.com/abhinavxd/artemis/internal/workerpool"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	kjson "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	flag "github.com/spf13/pflag"
	"github.com/zerodha/logf"
)

// constants holds the app constants.
type constants struct {
	AppBaseURL                  string
	LogoURL                     string
	SiteName                    string
	UploadProvider              string
	AllowedUploadFileExtensions []string
	MaxFileUploadSizeMB         float64
}

// Config loads config files into koanf.
func initConfig(ko *koanf.Koanf) {
	for _, f := range ko.Strings("config") {
		log.Println("reading config file:", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				log.Fatal("error config file not found.")
			}
			log.Fatalf("error loading config from file: %v.", err)
		}
	}
}

// initFlags initializes the commandline flags.
func initFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)

	// Registering `--help` handler.
	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		f.PrintDefaults()
	}

	// Register the commandline flags and parse them.
	f.StringSlice("config", []string{"config.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("version", false, "show current version of the build")
	f.Bool("install", false, "setup database")
	f.Bool("set-system-user-password", false, "set password for the system user")

	if err := f.Parse(os.Args[1:]); err != nil {
		log.Fatalf("loading flags: %v", err)
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		log.Fatalf("loading config: %v", err)
	}
}

// initConstants initializes the app constants.
func initConstants() constants {
	return constants{
		AppBaseURL:                  ko.String("app.root_url"),
		LogoURL:                     ko.String("app.logo_url"),
		SiteName:                    ko.String("app.site_name"),
		UploadProvider:              ko.MustString("upload.provider"),
		AllowedUploadFileExtensions: ko.Strings("app.allowed_file_upload_extensions"),
		MaxFileUploadSizeMB:         ko.Float64("app.max_file_upload_size"),
	}
}

// initFS initializes the stuffbin FileSystem.
func initFS() stuffbin.FileSystem {
	var files = []string{
		"frontend/dist",
		"i18n",
	}

	// Get self executable path.
	path, err := os.Executable()
	if err != nil {
		log.Fatalf("error initializing FS: %v", err)
	}

	// Load embedded files in the executable.
	fs, err := stuffbin.UnStuff(path)

	if err != nil {
		if err == stuffbin.ErrNoID {
			// The embed failed or the binary's already unstuffed or running in local / dev mode, use the local filesystem.
			log.Println("unstuff failed, using local FS")
			fs, err = stuffbin.NewLocalFS("/", files...)
			if err != nil {
				log.Fatalf("error initializing local FS: %v", err)
			}
		} else {
			log.Fatalf("error initializing FS: %v", err)
		}
	}
	return fs
}

// loadSettings loads settings from the DB into Koanf map.
func loadSettings(m *setting.Manager) {
	j, err := m.GetAllJSON()
	if err != nil {
		log.Fatalf("error parsing settings from DB: %v", err)
	}

	// Setting keys are dot separated, eg: app.favicon_url. Unflatten them into
	// nested maps {app: {favicon_url}}.
	var out map[string]interface{}

	if err := json.Unmarshal(j, &out); err != nil {
		log.Fatalf("error unmarshalling settings from DB: %v", err)
	}
	if err := ko.Load(confmap.Provider(out, "."), nil); err != nil {
		log.Fatalf("error parsing settings from DB: %v", err)
	}
}

// initSettings inits setting manager.
func initSettings(db *sqlx.DB) *setting.Manager {
	s, err := setting.New(setting.Opts{
		DB: db,
		Lo: initLogger("settings"),
	})
	if err != nil {
		log.Fatalf("error initializing setting manager: %v", err)
	}
	return s
}

// initUser inits user manager.
func initUser(i18n *i18n.I18n, DB *sqlx.DB) *user.Manager {
	mgr, err := user.New(i18n, user.Opts{
		DB: DB,
		Lo: initLogger("user_manager"),
	})
	if err != nil {
		log.Fatalf("error initializing user manager: %v", err)
	}
	return mgr
}

// initConversations inits conversation manager.
func initConversations(
	i18n *i18n.I18n,
	hub *ws.Hub,
	n *notifier.Service,
	db *sqlx.DB,
	inboxStore *inbox.Manager,
	userStore *user.Manager,
	teamStore *team.Manager,
	mediaStore *media.Manager,
	automationEngine *automation.Engine,
	template *tmpl.Manager,
) *conversation.Manager {
	c, err := conversation.New(hub, i18n, n, inboxStore, userStore, teamStore, mediaStore, automationEngine, template, conversation.Opts{
		DB:                       db,
		Lo:                       initLogger("conversation_manager"),
		OutgoingMessageQueueSize: ko.MustInt("message.outgoing_queue_size"),
		IncomingMessageQueueSize: ko.MustInt("message.incoming_queue_size"),
	})
	if err != nil {
		log.Fatalf("error initializing conversation manager: %v", err)
	}
	return c
}

// initTag inits tag manager.
func initTag(db *sqlx.DB) *tag.Manager {
	var lo = initLogger("tag_manager")
	mgr, err := tag.New(tag.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing tags: %v", err)
	}
	return mgr
}

// initViews inits view manager.
func initView(db *sqlx.DB) *view.Manager {
	var lo = initLogger("view_manager")
	m, err := view.New(view.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing view manager: %v", err)
	}
	return m
}

// initCannedResponse inits canned response manager.
func initCannedResponse(db *sqlx.DB) *cannedresp.Manager {
	var lo = initLogger("canned-response")
	c, err := cannedresp.New(cannedresp.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing canned responses manager: %v", err)
	}
	return c
}

// initBusinessHours inits business hours manager.
func initBusinessHours(db *sqlx.DB) *businesshours.Manager {
	var lo = initLogger("business-hours")
	m, err := businesshours.New(businesshours.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing business hours manager: %v", err)
	}
	return m
}

// initSLA inits SLA manager.
func initSLA(db *sqlx.DB, teamManager *team.Manager, settings *setting.Manager, businessHours *businesshours.Manager) *sla.Manager {
	var lo = initLogger("sla")
	m, err := sla.New(sla.Opts{
		DB:              db,
		Lo:              lo,
		ScannerInterval: ko.MustDuration("sla.scanner_interval"),
	}, workerpool.New(ko.MustInt("sla.worker_count"), ko.MustInt("sla.queue_size")), teamManager, settings, businessHours)
	if err != nil {
		log.Fatalf("error initializing SLA manager: %v", err)
	}
	return m
}

// initCSAT inits CSAT manager.
func initCSAT(db *sqlx.DB) *csat.Manager {
	var lo = initLogger("csat")
	m, err := csat.New(csat.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing CSAT manager: %v", err)
	}
	return m
}

// initTemplates inits template manager.
func initTemplate(db *sqlx.DB, fs stuffbin.FileSystem, consts constants) *tmpl.Manager {
	var (
		lo      = initLogger("template")
		funcMap = getTmplFuncs(consts)
	)
	tpls, err := stuffbin.ParseTemplatesGlob(funcMap, fs, "/static/email-templates/*.html")
	if err != nil {
		log.Fatalf("error parsing e-mail templates: %v", err)
	}

	webTpls, err := stuffbin.ParseTemplatesGlob(funcMap, fs, "/static/public/web-templates/*.html")
	if err != nil {
		log.Fatalf("error parsing web templates: %v", err)
	}
	m, err := tmpl.New(lo, db, webTpls, tpls, funcMap)
	if err != nil {
		log.Fatalf("error initializing template manager: %v", err)
	}
	return m
}

// getTmplFuncs returns the template functions.
func getTmplFuncs(consts constants) template.FuncMap {
	return template.FuncMap{
		"RootURL": func() string {
			return consts.AppBaseURL
		},
		"Date": func(layout string) string {
			if layout == "" {
				layout = time.ANSIC
			}
			return time.Now().Format(layout)
		},
		"LogoURL": func() string {
			return consts.LogoURL
		},
		"SiteName": func() string {
			return consts.SiteName
		},
	}
}

// initTeam inits team manager.
func initTeam(db *sqlx.DB) *team.Manager {
	var lo = initLogger("team-manager")
	mgr, err := team.New(team.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing team manager: %v", err)
	}
	return mgr
}

// initMedia inits media manager.
func initMedia(db *sqlx.DB) *media.Manager {
	var (
		store      media.Store
		err        error
		appRootURL = ko.String("app.root_url")
		lo         = initLogger("media")
	)
	switch s := ko.MustString("upload.provider"); s {
	case "s3":
		store, err = s3.New(s3.Opt{
			URL:        ko.String("upload.s3.url"),
			PublicURL:  ko.String("upload.s3.public_url"),
			AccessKey:  ko.String("upload.s3.access_key"),
			SecretKey:  ko.String("upload.s3.secret_key"),
			Region:     ko.String("upload.s3.region"),
			Bucket:     ko.String("upload.s3.bucket"),
			BucketPath: ko.String("upload.s3.bucket_path"),
			BucketType: ko.String("upload.s3.bucket_type"),
			Expiry:     ko.Duration("upload.s3.expiry"),
		})
		if err != nil {
			log.Fatalf("error initializing s3 media store: %v", err)
		}
	case "fs":
		store, err = fs.New(fs.Opts{
			UploadURI:  "/uploads",
			UploadPath: filepath.Clean(ko.String("upload.fs.upload_path")),
			RootURL:    appRootURL,
		})
		if err != nil {
			log.Fatalf("error initializing fs media store: %v", err)
		}
	default:
		log.Fatalf("unknown media store: %s", s)
	}

	media, err := media.New(media.Opts{
		Store: store,
		Lo:    lo,
		DB:    db,
	})
	if err != nil {
		log.Fatalf("error initializing media: %v", err)
	}
	return media
}

// initInbox initializes the inbox manager without registering inboxes.
func initInbox(db *sqlx.DB) *inbox.Manager {
	var lo = initLogger("inbox-manager")
	mgr, err := inbox.New(lo, db)
	if err != nil {
		log.Fatalf("error initializing inbox manager: %v", err)
	}
	return mgr
}

// initAutomationEngine initializes the automation engine.
func initAutomationEngine(db *sqlx.DB, userManager *user.Manager) *automation.Engine {
	var lo = initLogger("automation_engine")

	systemUser, err := userManager.GetSystemUser()
	if err != nil {
		log.Fatalf("error fetching system user: %v", err)
	}

	engine, err := automation.New(systemUser, automation.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing automation engine: %v", err)
	}
	return engine
}

// initAutoAssigner initializes the auto assigner.
func initAutoAssigner(teamManager *team.Manager, userManager *user.Manager, conversationManager *conversation.Manager) *autoassigner.Engine {
	systemUser, err := userManager.GetSystemUser()
	if err != nil {
		log.Fatalf("error fetching system user: %v", err)
	}
	e, err := autoassigner.New(teamManager, conversationManager, systemUser, initLogger("autoassigner"))
	if err != nil {
		log.Fatalf("error initializing auto assigner: %v", err)
	}
	return e
}

// initNotifier initializes the notifier service with available providers.
func initNotifier(userStore notifier.UserStore) *notifier.Service {
	smtpCfg := email.SMTPConfig{}
	if err := ko.UnmarshalWithConf("notification.email", &smtpCfg, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		log.Fatalf("error unmarshalling email notification provider config: %v", err)
	}

	emailNotifier, err := emailnotifier.New([]email.SMTPConfig{smtpCfg}, userStore, emailnotifier.Opts{
		Lo:        initLogger("email-notifier"),
		FromEmail: ko.String("notification.email.email_address"),
	})
	if err != nil {
		log.Fatalf("error initializing email notifier: %v", err)
	}

	notifierProviders := map[string]notifier.Notifier{
		emailNotifier.Name(): emailNotifier,
	}

	return notifier.NewService(notifierProviders, ko.MustInt("notification.concurrency"), ko.MustInt("notification.queue_size"), initLogger("notifier"))
}

// initEmailInbox initializes the email inbox.
func initEmailInbox(inboxRecord imodels.Inbox, store inbox.MessageStore) (inbox.Inbox, error) {
	var config email.Config

	// Load JSON data into Koanf.
	if err := ko.Load(rawbytes.Provider([]byte(inboxRecord.Config)), kjson.Parser()); err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	if err := ko.UnmarshalWithConf("", &config, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		log.Fatalf("error unmarshalling `%s` %s config: %v", inboxRecord.Channel, inboxRecord.Name, err)
	}

	if len(config.SMTP) == 0 {
		log.Printf("WARNING: Zero SMTP servers configured for `%s` inbox: Name: `%s`", inboxRecord.Channel, inboxRecord.Name)
	}

	if len(config.IMAP) == 0 {
		log.Printf("WARNING: Zero IMAP clients configured for `%s` inbox: Name: `%s`", inboxRecord.Channel, inboxRecord.Name)
	}

	config.From = inboxRecord.From

	if len(config.From) == 0 {
		log.Printf("WARNING: No `from` email address set for `%s` inbox: Name: `%s`", inboxRecord.Channel, inboxRecord.Name)
	}

	inbox, err := email.New(store, email.Opts{
		ID:     inboxRecord.ID,
		Config: config,
		Lo:     initLogger("email_inbox"),
	})

	if err != nil {
		log.Fatalf("ERROR: initalizing `%s` inbox: `%s` error : %v", inboxRecord.Channel, inboxRecord.Name, err)
		return nil, err
	}

	log.Printf("`%s` inbox successfully initalized. %d smtp servers. %d imap clients.", inboxRecord.Name, len(config.SMTP), len(config.IMAP))

	return inbox, nil
}

// initializeInboxes handles inbox initialization.
func initializeInboxes(inboxR imodels.Inbox, store inbox.MessageStore) (inbox.Inbox, error) {
	switch inboxR.Channel {
	case "email":
		return initEmailInbox(inboxR, store)
	default:
		return nil, fmt.Errorf("unknown inbox channel: %s", inboxR.Channel)
	}
}

// reloadInboxes reloads all inboxes.
func reloadInboxes(app *App) error {
	app.lo.Info("reloading inboxes")
	return app.inbox.Reload(ctx, initializeInboxes)
}

// startInboxes registers the active inboxes and starts receiver for each.
func startInboxes(ctx context.Context, mgr *inbox.Manager, store inbox.MessageStore) {
	mgr.SetMessageStore(store)

	if err := mgr.InitInboxes(initializeInboxes); err != nil {
		log.Fatalf("error initializing inboxes: %v", err)
	}

	if err := mgr.Start(ctx); err != nil {
		log.Fatalf("error starting inboxes: %v", err)
	}
}

// initAuthz initializes authorization enforcer.
func initAuthz() *authz.Enforcer {
	enforcer, err := authz.NewEnforcer(initLogger("authz"))
	if err != nil {
		log.Fatalf("error initializing authz: %v", err)
	}
	return enforcer
}

// initAuth initializes the authentication manager.
func initAuth(o *oidc.Manager, rd *redis.Client) *auth_.Auth {
	lo := initLogger("auth")

	providers, err := buildProviders(o)
	if err != nil {
		log.Fatalf("error initializing auth: %v", err)
	}

	auth, err := auth_.New(auth_.Config{Providers: providers}, rd, lo)
	if err != nil {
		log.Fatalf("error initializing auth: %v", err)
	}

	return auth
}

// reloadAuth reloads the auth providers.
func reloadAuth(app *App) error {
	app.lo.Info("reloading auth manager")

	providers, err := buildProviders(app.oidc)
	if err != nil {
		log.Fatalf("error reloading auth: %v", err)
	}

	if err := app.auth.Reload(auth_.Config{Providers: providers}); err != nil {
		app.lo.Error("error reloading auth", "error", err)
		return err
	}

	return nil
}

// buildProviders creates a list of auth providers from the OIDC manager.
func buildProviders(o *oidc.Manager) ([]auth_.Provider, error) {
	oidcConfigs, err := o.GetAll()
	if err != nil {
		return nil, err
	}

	providers := make([]auth_.Provider, 0, len(oidcConfigs))
	for _, config := range oidcConfigs {
		if config.Disabled {
			continue
		}
		providers = append(providers, auth_.Provider{
			ID:           config.ID,
			Provider:     config.Provider,
			ProviderURL:  config.ProviderURL,
			RedirectURL:  config.RedirectURI,
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
		})
	}
	return providers, nil
}

// initOIDC initializes open id connect config manager.
func initOIDC(db *sqlx.DB) *oidc.Manager {
	lo := initLogger("oidc")
	o, err := oidc.New(oidc.Opts{
		DB: db,
		Lo: lo,
	})

	if err != nil {
		log.Fatalf("error initializing oidc: %v", err)
	}
	return o
}

// initI18n inits i18n.
func initI18n(fs stuffbin.FileSystem) *i18n.I18n {
	file, err := fs.Get("i18n/" + cmp.Or(ko.String("app.lang"), defLang) + ".json")
	if err != nil {
		log.Fatalf("error reading i18n language file")
	}
	i18n, err := i18n.New(file.ReadBytes())
	if err != nil {
		log.Fatalf("error initializing i18n: %v", err)
	}
	return i18n
}

// initRedis inits redis DB.
func initRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     ko.MustString("redis.address"),
		Password: ko.String("redis.password"),
		DB:       ko.Int("redis.db"),
	})
}

// initRedis inits postgres DB.
func initDB() *sqlx.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s %s",
		ko.MustString("db.host"),
		ko.MustInt("db.port"),
		ko.MustString("db.user"),
		ko.MustString("db.password"),
		ko.MustString("db.database"),
		ko.String("db.ssl_mode"),
		ko.String("db.params"),
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatalf("error connecting to DB: %v", err)
	}

	db.SetMaxOpenConns(ko.MustInt("db.max_open"))
	db.SetMaxIdleConns(ko.MustInt("db.max_idle"))
	db.SetConnMaxLifetime(ko.MustDuration("db.max_lifetime"))

	return db
}

// initRedis inits role manager.
func initRole(db *sqlx.DB) *role.Manager {
	var lo = initLogger("role_manager")
	r, err := role.New(role.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing role manager: %v", err)
	}
	return r
}

// initStatus inits conversation status manager.
func initStatus(db *sqlx.DB) *status.Manager {
	manager, err := status.New(status.Opts{
		DB: db,
		Lo: initLogger("status-manager"),
	})
	if err != nil {
		log.Fatalf("error initializing status manager: %v", err)
	}
	return manager
}

// initPriority inits conversation priority manager.
func initPriority(db *sqlx.DB) *priority.Manager {
	manager, err := priority.New(priority.Opts{
		DB: db,
		Lo: initLogger("priority-manager"),
	})
	if err != nil {
		log.Fatalf("error initializing priority manager: %v", err)
	}
	return manager
}

// initLogger initializes a logf logger.
func initLogger(src string) *logf.Logger {
	lvl, env := ko.MustString("app.log_level"), ko.MustString("app.env")
	lo := logf.New(logf.Opts{
		Level:                getLogLevel(lvl),
		EnableColor:          getColor(env),
		EnableCaller:         true,
		CallerSkipFrameCount: 3,
		DefaultFields:        []any{"sc", src},
	})
	return &lo
}

func getColor(env string) bool {
	color := false
	if env == "dev" {
		color = true
	}
	return color
}

func getLogLevel(lvl string) logf.Level {
	switch lvl {
	case "info":
		return logf.InfoLevel
	case "debug":
		return logf.DebugLevel
	case "warn":
		return logf.WarnLevel
	case "error":
		return logf.ErrorLevel
	case "fatal":
		return logf.FatalLevel
	default:
		return logf.InfoLevel
	}
}
