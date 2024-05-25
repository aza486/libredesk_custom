package initz

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	_ "github.com/lib/pq"
	"github.com/zerodha/logf"
)

// Config loads config files into koanf.
func Config(ko *koanf.Koanf) {
	for _, f := range ko.Strings("config") {
		log.Println("reading config file: ", f)
		if err := ko.Load(file.Provider(f), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				log.Fatal("config file not found.")
			}
			log.Fatalf("loading config from file: %v.", err)
		}
	}
}

func DB(ko *koanf.Koanf) *sqlx.DB {
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
		log.Fatalf("connecting to DB %v", err)
	}

	db.SetMaxOpenConns(c.MaxOpen)
	db.SetMaxIdleConns(c.MaxIdle)
	db.SetConnMaxLifetime(c.MaxLifetime)

	return db
}

func Redis(ko *koanf.Koanf) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     ko.MustString("redis.address"),
		Password: ko.String("redis.password"),
		DB:       ko.Int("redis.db"),
	})
}

// Logger initialies a logf logger.
func Logger(lvl string, env string) logf.Logger {
	lo := logf.New(logf.Opts{
		Level:                getLogLevel(lvl),
		EnableColor:          getColor(env),
		EnableCaller:         true,
		CallerSkipFrameCount: 3,
		DefaultFields:        []any{"sc", "artemis"},
	})
	return lo
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
