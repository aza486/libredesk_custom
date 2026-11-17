package main

import (
	"context"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func validateCasePublisherConfig(enabled bool, dsn string, embed learning.Embedder) error {
	if enabled && (strings.TrimSpace(dsn) == "" || embed == nil) {
		return learning.ErrUnavailable
	}
	return nil
}

// No connection or goroutine is created unless explicitly enabled. The embedding
// closure is the existing text-embedding-3-small / 1536 client, not a new transport.
func initLearningCasePublisher(ctx context.Context, lo *logf.Logger, embed learning.Embedder) {
	enabled := ko.Bool("ai_learning.case_publisher_enabled")
	if !enabled {
		return
	}
	dsn := ko.String("ai_learning.case_publisher_database_dsn")
	if err := validateCasePublisherConfig(enabled, dsn, embed); err != nil {
		logLearningFatal(lo, "case_config", err)
	}
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		logLearningFatal(lo, "case_database", err)
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err = db.PingContext(timeout); err != nil {
		db.Close()
		logLearningFatal(lo, "case_database", err)
	}
	publisher := learning.NewCasePublisher(db, embed, true)
	publisher.ReportFailure = func(err *learning.CasePublishError) { reportCasePublisherFailure(lo, err) }
	go func() { defer db.Close(); publisher.Run(ctx) }()
}

func reportCasePublisherFailure(lo *logf.Logger, failure *learning.CasePublishError) {
	if lo == nil {
		return
	}
	code := "case_internal_failure"
	switch failure.Code {
	case "case_configuration_invalid", "case_database_failed", "case_revision_conflict", "case_validation_failed", "case_embedding_failed", "case_vector_invalid":
		code = failure.Code
	}
	fields := []any{"stage", "case_publish", "cause", code}
	// Only UUIDs can enter logs, even if a caller supplies a malformed diagnostic.
	if _, err := uuid.Parse(failure.ID); err == nil {
		fields = append(fields, "item_id", failure.ID)
	}
	lo.Error("AI case publication failed", fields...)
}
