package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/attachment/stores/s3"
	uauth "github.com/abhinavxd/artemis/internal/auth"
	"github.com/abhinavxd/artemis/internal/autoassigner"
	"github.com/abhinavxd/artemis/internal/automation"
	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/contact"
	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/abhinavxd/artemis/internal/inbox/channel/email"
	"github.com/abhinavxd/artemis/internal/initz"
	"github.com/abhinavxd/artemis/internal/message"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	emailnotifier "github.com/abhinavxd/artemis/internal/notification/providers/email"
	"github.com/abhinavxd/artemis/internal/tag"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/template"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
	"github.com/vividvilla/simplesessions"
	sessredisstore "github.com/vividvilla/simplesessions/stores/goredis"
	"github.com/zerodha/logf"
)

// consts holds the app constants.
type consts struct {
	AppBaseURL                  string
	AllowedFileUploadExtensions []string
}

func initFlags() {
	f := flag.NewFlagSet("config", flag.ContinueOnError)

	// Registering `--help` handler.
	f.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
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

// initSessionManager initializes and returns a simplesessions.Manager instance.
func initSessionManager(rd *redis.Client) *simplesessions.Manager {
	ttl := ko.Duration("app.session.cookie_ttl")
	s := simplesessions.New(simplesessions.Options{
		CookieName:       ko.MustString("app.session.cookie_name"),
		CookiePath:       ko.MustString("app.session.cookie_path"),
		IsSecureCookie:   ko.Bool("app.session.cookie_secure"),
		DisableAutoSet:   ko.Bool("app.session.cookie_disable_auto_set"),
		IsHTTPOnlyCookie: true,
		CookieLifetime:   ttl,
	})

	// Initialize a Redis pool for session storage.
	st := sessredisstore.New(context.TODO(), rd)

	// Prefix backend session keys with cookie name.
	st.SetPrefix(ko.MustString("app.session.cookie_name") + ":")
	// Set TTL in backend if its set.
	if ttl > 0 {
		st.SetTTL(ttl)
	}

	s.UseStore(st)
	s.RegisterGetCookie(simpleSessGetCookieCB)
	s.RegisterSetCookie(simpleSessSetCookieCB)
	return s
}

func initUserDB(DB *sqlx.DB, lo *logf.Logger) *user.Manager {
	mgr, err := user.New(user.Opts{
		DB:         DB,
		Lo:         lo,
		BcryptCost: ko.MustInt("app.user.password_bcypt_cost"),
	})
	if err != nil {
		log.Fatalf("error initializing user manager: %v", err)
	}
	return mgr
}

func initConversations(hub *ws.Hub, db *sqlx.DB, lo *logf.Logger) *conversation.Manager {
	c, err := conversation.New(hub, conversation.Opts{
		DB:                  db,
		Lo:                  lo,
		ReferenceNumPattern: ko.String("app.constants.conversation_reference_number_pattern"),
	})
	if err != nil {
		log.Fatalf("error initializing conversation manager: %v", err)
	}
	return c
}

func initTags(db *sqlx.DB, lo *logf.Logger) *tag.Manager {
	mgr, err := tag.New(tag.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing tags: %v", err)
	}
	return mgr
}

func initCannedResponse(db *sqlx.DB, lo *logf.Logger) *cannedresp.Manager {
	c, err := cannedresp.New(cannedresp.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing canned responses manager: %v", err)
	}
	return c
}

func initContactManager(db *sqlx.DB, lo *logf.Logger) *contact.Manager {
	m, err := contact.New(contact.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing contact manager: %v", err)
	}
	return m
}

func initTemplateMgr(db *sqlx.DB) *template.Manager {
	m, err := template.New(db)
	if err != nil {
		log.Fatalf("error initializing template manager: %v", err)
	}
	return m
}

func initMessages(db *sqlx.DB,
	lo *logf.Logger,
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

func initTeamMgr(db *sqlx.DB, lo *logf.Logger) *team.Manager {
	mgr, err := team.New(team.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing team manager: %v", err)
	}
	return mgr
}

func initAttachmentsManager(db *sqlx.DB, lo *logf.Logger) *attachment.Manager {
	var (
		mgr   *attachment.Manager
		store attachment.Store
		err   error
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
func initInboxManager(db *sqlx.DB, lo *logf.Logger) *inbox.Manager {
	mgr, err := inbox.New(lo, db)
	if err != nil {
		log.Fatalf("error initializing inbox manager: %v", err)
	}
	return mgr
}

func initAutomationEngine(db *sqlx.DB, lo *logf.Logger) *automation.Engine {
	engine, err := automation.New(automation.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing automation engine: %v", err)
	}
	return engine
}

func initAutoAssignmentEngine(teamMgr *team.Manager, convMgr *conversation.Manager, msgMgr *message.Manager,
	notifier notifier.Notifier, hub *ws.Hub, lo *logf.Logger) *autoassigner.Engine {
	engine, err := autoassigner.New(teamMgr, convMgr, msgMgr, notifier, hub, lo)
	if err != nil {
		log.Fatalf("error initializing auto assignment engine: %v", err)
	}
	return engine
}

func initRBACEngine(db *sqlx.DB) *uauth.Engine {
	engine, err := uauth.New(db, &logf.Logger{})
	if err != nil {
		log.Fatalf("error initializing rbac enginer: %v", err)
	}
	return engine
}

func initNotifier(userStore notifier.UserStore, templateRenderer notifier.TemplateRenderer) notifier.Notifier {
	var smtpCfg email.SMTPConfig
	if err := ko.UnmarshalWithConf("notification.provider.email", &smtpCfg, koanf.UnmarshalConf{Tag: "json"}); err != nil {
		log.Fatalf("error unmarshalling email notification provider config: %v", err)
	}
	notifier, err := emailnotifier.New([]email.SMTPConfig{smtpCfg}, userStore, templateRenderer, emailnotifier.Opts{
		Lo:        initz.Logger(ko.MustString("app.log_level"), ko.MustString("app.env"), "email-notifier"),
		FromEmail: ko.String("notification.provider.email.email_address"),
	})
	if err != nil {
		log.Fatalf("error initializing email notifier: %v", err)
	}
	return notifier
}

// initEmailInbox initializes the email inbox.
func initEmailInbox(inboxRecord inbox.InboxRecord, store inbox.MessageStore) (inbox.Inbox, error) {
	var config email.Config

	// Load JSON data into Koanf.
	if err := ko.Load(rawbytes.Provider([]byte(inboxRecord.Config)), json.Parser()); err != nil {
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

	// Set from addr.
	config.From = inboxRecord.From

	inbox, err := email.New(store, email.Opts{
		ID:     inboxRecord.ID,
		Config: config,
		Lo:     initz.Logger(ko.MustString("app.log_level"), ko.MustString("app.env"), "email_inbox"),
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
	inboxRecords, err := mgr.GetActiveInboxes()
	if err != nil {
		log.Fatalf("error fetching active inboxes %v", err)
	}

	for _, inboxR := range inboxRecords {
		switch inboxR.Channel {
		case "email":
			log.Printf("initializing `Email` inbox: %s", inboxR.Name)
			inbox, err := initEmailInbox(inboxR, store)
			if err != nil {
				log.Fatalf("error initializing email inbox %v", err)
			}
			mgr.Register(inbox)
		default:
			log.Printf("WARNING: Unknown inbox channel: %s", inboxR.Name)
		}
	}
}
