package main

import (
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/jmoiron/sqlx"
)

func initFormRecipients() *conversation.FormRecipients {
	senders := ko.Strings("form_recipients.technical_senders")
	if len(senders) == 0 {
		senders = []string{"bestellung@pixelfotoexpress.de"}
	}
	config := &conversation.FormRecipients{TechnicalSenders: senders}
	if dsn := ko.String("form_recipients.database_dsn"); dsn != "" {
		db, err := sqlx.Open("postgres", dsn)
		if err != nil {
			panic("invalid form_recipients database configuration; credentials withheld")
		}
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(2)
		db.SetConnMaxLifetime(5 * time.Minute)
		// Lazy connection: an unavailable integration only affects suggestions, not sending or startup.
		config.DB = db
	}
	return config
}
