package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/attachment/stores/s3"
	"github.com/abhinavxd/artemis/internal/autoassigner"
	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/inbox/channel/email"
	imodels "github.com/abhinavxd/artemis/internal/inbox/models"
	"github.com/abhinavxd/artemis/internal/message"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	emailnotifier "github.com/abhinavxd/artemis/internal/notification/providers/email"
	"github.com/abhinavxd/artemis/internal/role"
	"github.com/abhinavxd/artemis/internal/setting"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/template"
	"github.com/abhinavxd/artemis/internal/user"
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
	sessredisstore "github.com/zerodha/simplesessions/stores/redis/v3"
	"github.com/zerodha/simplesessions/v3"
)

// consts holds the app constants.
type consts struct {
	AppBaseURL                  string
	AllowedFileUploadExtensions []string
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

	if err := f.Parse(os.Args[1:]); err != nil {
		log.Fatalf("loading flags: %v", err)
	}

	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		log.Fatalf("loading config: %v", err)
	}
}

func initConstants() consts {
	return consts{
		AppBaseURL:                  ko.String("app.constants.base_url"),
		AllowedFileUploadExtensions: ko.Strings("app.constants.allowed_file_upload_extensions"),
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

// loadSettings loads settings from the DB into the given Koanf map.
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

func initSettingsManager(db *sqlx.DB) *setting.Manager {
	s, err := setting.New(setting.Opts{
		DB: db,
	})
	if err != nil {
		log.Fatalf("error initializing setting manager: %v", err)
	}
	return s
}

// initSessionManager initializes and returns a simplesessions.Manager instance.
func initSessionManager(rd *redis.Client) *simplesessions.Manager {
	maxAge := ko.Duration("app.session.cookie_max_age")
	if maxAge.Seconds() == 0 {
		maxAge = time.Hour * 12
	}
	s := simplesessions.New(simplesessions.Options{
		EnableAutoCreate: true,
		SessionIDLength:  64,
		Cookie: simplesessions.CookieOptions{
			IsHTTPOnly: true,
			IsSecure:   true,
			MaxAge:     maxAge,
		},
	})
	st := sessredisstore.New(context.TODO(), rd)
	s.UseStore(st)
	s.SetCookieHooks(simpleSessGetCookieCB, simpleSessSetCookieCB)
	return s
}

func initUserManager(i18n *i18n.I18n, DB *sqlx.DB) *user.Manager {
	mgr, err := user.New(i18n, user.Opts{
		DB:         DB,
		Lo:         initLogger("user_manager"),
		BcryptCost: ko.MustInt("app.user.password_bcypt_cost"),
	})
	if err != nil {
		log.Fatalf("error initializing user manager: %v", err)
	}
	return mgr
}

func initConversations(i18n *i18n.I18n, hub *ws.Hub, n notifier.Notifier, db *sqlx.DB) *conversation.Manager {
	c, err := conversation.New(hub, i18n, n, conversation.Opts{
		DB:                  db,
		Lo:                  initLogger("conversation_manager"),
		ReferenceNumPattern: ko.String("app.constants.conversation_reference_number_pattern"),
	})
	if err != nil {
		log.Fatalf("error initializing conversation manager: %v", err)
	}
	return c
}

func initTags(db *sqlx.DB) *tag.Manager {
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

func initCannedResponse(db *sqlx.DB) *cannedresp.Manager {
	var lo = initLogger("canned_response_manager")
	c, err := cannedresp.New(cannedresp.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing canned responses manager: %v", err)
	}
	return c
}

func initContactManager(db *sqlx.DB) *contact.Manager {
	var lo = initLogger("contact_manager")
	m, err := contact.New(contact.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing contact manager: %v", err)
	}
	return m
}

func initTemplateManager(db *sqlx.DB) *template.Manager {
	m, err := template.New(db)
	if err != nil {
		log.Fatalf("error initializing template manager: %v", err)
	}
	return m
}

func initMessages(db *sqlx.DB,
	wsHub *ws.Hub,
	userMgr *user.Manager,
	teaMgr *team.Manager,
	contactMgr *contact.Manager,
	attachmentMgr *attachment.Manager,
	conversationMgr *conversation.Manager,
	inboxMgr *inbox.Manager,
	automationEngine *automation.Engine,
	templateManager *template.Manager,
) *message.Manager {
	var lo = initLogger("message_manager")
	mgr, err := message.New(
		wsHub,
		userMgr,
		teaMgr,
		contactMgr,
		attachmentMgr,
		inboxMgr,
		conversationMgr,
		automationEngine,
		templateManager,
		message.Opts{
			DB:                   db,
			Lo:                   lo,
			OutgoingMsgQueueSize: ko.MustInt("message.outgoing_queue_size"),
			IncomingMsgQueueSize: ko.MustInt("message.incoming_queue_size"),
		})
	if err != nil {
		log.Fatalf("error initializing message manager: %v", err)
	}
	return mgr
}

func initTeamManager(db *sqlx.DB) *team.Manager {
	var lo = initLogger("team_manager")
	mgr, err := team.New(team.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing team manager: %v", err)
	}
	return mgr
}

func initAttachmentsManager(db *sqlx.DB) *attachment.Manager {
	var (
		mgr   *attachment.Manager
		store attachment.Store
		err   error
		lo    = initLogger("attachments_manager")
	)
	switch s := ko.MustString("app.attachment_store"); s {
	case "s3":
		store, err = s3.New(s3.Opt{
			URL:        ko.String("s3.url"),
			PublicURL:  ko.String("s3.public_url"),
			AccessKey:  ko.String("s3.access_key"),
			SecretKey:  ko.String("s3.secret_key"),
			Region:     ko.String("s3.region"),
			Bucket:     ko.String("s3.bucket"),
			BucketPath: ko.String("s3.bucket_path"),
			BucketType: ko.String("s3.bucket_type"),
			Expiry:     ko.Duration("s3.expiry"),
		})
		if err != nil {
			log.Fatalf("error initializing s3 %v", err)
		}
	default:
		log.Fatalf("media store: %s not available", s)
	}

	mgr, err = attachment.New(attachment.Opts{
		Store:      store,
		Lo:         lo,
		DB:         db,
		AppBaseURL: ko.String("app.constants.base_url"),
	})
	if err != nil {
		log.Fatalf("initializing attachments manager %v", err)
	}
	return mgr
}

// initInboxManager initializes the inbox manager without registering inboxes.
func initInboxManager(db *sqlx.DB) *inbox.Manager {
	var lo = initLogger("inbox_manager")
	mgr, err := inbox.New(lo, db)
	if err != nil {
		log.Fatalf("error initializing inbox manager: %v", err)
	}
	return mgr
}

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

func initAutoAssigner(teamManager *team.Manager, conversationManager *conversation.Manager) *autoassigner.Engine {
	e, err := autoassigner.New(teamManager, conversationManager, initLogger("autoassigner"))
	if err != nil {
		log.Fatalf("error initializing auto assigner engine: %v", err)
	}
	return e
}

func initNotifier(userStore notifier.UserStore, templateRenderer notifier.TemplateRenderer) notifier.Notifier {
	var smtpCfg email.SMTPConfig
	if err := ko.UnmarshalWithConf("notification.provider.email", &smtpCfg, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		log.Fatalf("error unmarshalling email notification provider config: %v", err)
	}
	notifier, err := emailnotifier.New([]email.SMTPConfig{smtpCfg}, userStore, templateRenderer, emailnotifier.Opts{
		Lo:        initLogger("email-notifier"),
		FromEmail: ko.String("notification.provider.email.email_address"),
	})
	if err != nil {
		log.Fatalf("error initializing email notifier: %v", err)
	}
	return notifier
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

// registerInboxes registers the active inboxes with the inbox manager.
func registerInboxes(mgr *inbox.Manager, store inbox.MessageStore) {
	inboxRecords, err := mgr.GetActive()
	if err != nil {
		log.Fatalf("error fetching active inboxes %v", err)
	}

	for _, inboxR := range inboxRecords {
		switch inboxR.Channel {
		case "email":
			log.Printf("initializing `Email` inbox: %s", inboxR.Name)
			inbox, err := initEmailInbox(inboxR, store)
			if err != nil {
				log.Fatalf("error initializing email inbox: %v", err)
			}
			mgr.Register(inbox)
		default:
			log.Printf("WARNING: Unknown inbox channel: %s", inboxR.Name)
		}
	}
}

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

func initRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     ko.MustString("redis.address"),
		Password: ko.String("redis.password"),
		DB:       ko.Int("redis.db"),
	})
}

func initDB() *sqlx.DB {
	var c struct {
		Host        string        `koanf:"host"`
		Port        int           `koanf:"port"`
		User        string        `koanf:"user"`
		Password    string        `koanf:"password"`
		DBName      string        `koanf:"database"`
		SSLMode     string        `koanf:"ssl_mode"`
		Params      string        `koanf:"params"` // Extra params.
		MaxOpen     int           `koanf:"max_open"`
		MaxIdle     int           `koanf:"max_idle"`
		MaxLifetime time.Duration `koanf:"max_lifetime"`
	}
	if err := ko.Unmarshal("db", &c); err != nil {
		log.Fatalf("loading db config: %v", err)
	}

	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s %s", c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.Params))
	if err != nil {
		log.Fatalf("error connecting to DB %v", err)
	}

	db.SetMaxOpenConns(c.MaxOpen)
	db.SetMaxIdleConns(c.MaxIdle)
	db.SetConnMaxLifetime(c.MaxLifetime)

	return db
}

func initRoleManager(db *sqlx.DB) *role.Manager {
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
