package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/abhinavxd/artemis/internal/cannedresp"
	"github.com/abhinavxd/artemis/internal/conversations"
	"github.com/abhinavxd/artemis/internal/media"
	"github.com/abhinavxd/artemis/internal/media/stores/s3"
	"github.com/abhinavxd/artemis/internal/tags"
	user "github.com/abhinavxd/artemis/internal/userdb"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	flag "github.com/spf13/pflag"
	"github.com/vividvilla/simplesessions"
	sessredisstore "github.com/vividvilla/simplesessions/stores/goredis"
	"github.com/zerodha/logf"
)

// consts holds the app constants.
type consts struct {
	ChatReferenceNumberPattern   string
	AllowedMediaUploadExtensions []string
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

func initConstants(ko *koanf.Koanf) consts {
	return consts{
		ChatReferenceNumberPattern:   ko.String("app.constants.chat_reference_number_pattern"),
		AllowedMediaUploadExtensions: ko.Strings("app.constants.allowed_media_upload_extensions"),
	}
}

// initSessionManager initializes and returns a simplesessions.Manager instance.
func initSessionManager(rd *redis.Client, ko *koanf.Koanf) *simplesessions.Manager {
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

func initUserDB(DB *sqlx.DB, lo *logf.Logger, ko *koanf.Koanf) *user.UserDB {
	udb, err := user.New(user.Opts{
		DB:         DB,
		Lo:         lo,
		BcryptCost: ko.MustInt("app.user.password_bcypt_cost"),
	})
	if err != nil {
		log.Fatalf("error initializing userdb: %v", err)
	}
	return udb
}

func initConversations(db *sqlx.DB, lo *logf.Logger, ko *koanf.Koanf) *conversations.Conversations {
	c, err := conversations.New(conversations.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing conversations: %v", err)
	}
	return c
}

func initTags(db *sqlx.DB, lo *logf.Logger) *tags.Tags {
	t, err := tags.New(tags.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing tags: %v", err)
	}
	return t
}

func initCannedResponse(db *sqlx.DB, lo *logf.Logger) *cannedresp.CannedResp {
	c, err := cannedresp.New(cannedresp.Opts{
		DB: db,
		Lo: lo,
	})
	if err != nil {
		log.Fatalf("error initializing canned responses: %v", err)
	}
	return c
}

func initMediaManager(ko *koanf.Koanf, db *sqlx.DB) *media.Manager {
	var (
		manager *media.Manager
		store   media.Store
		err     error
	)
	// First init the store.
	switch s := ko.MustString("app.media_store"); s {
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
		log.Fatal("media store not available.")
	}

	manager, err = media.New(store, db)
	if err != nil {
		log.Fatalf("initializing media manager %v", err)
	}

	return manager
}
