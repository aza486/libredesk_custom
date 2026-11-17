package learning

import (
	"context"
	_ "embed"
	"errors"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// CasePublisherV11Schema is an explicit pg-n8n installation artifact after SQL005.
// It is embedded for installation-contract tests, never applied on startup.
//
//go:embed artifacts/006_case_publisher_v11.sql
var CasePublisherV11Schema string

const (
	casePollInterval   = 15 * time.Second
	caseRetryDelay     = time.Minute
	caseAttemptTimeout = 90 * time.Second
	caseRetryLimit     = 1024
)

// CasePublishError contains only a fixed code and an item UUID. Original database
// and upstream errors are intentionally not retained: they may contain content.
type CasePublishError struct {
	ID    string
	Code  string
	class error
}

func (e *CasePublishError) Error() string { return "case publication failed" }
func (e *CasePublishError) Unwrap() error { return e.class }

// CasePublisher never creates reviews. It uses SQL005 as its only write path.
// A separate instance/DB pool keeps the fact publisher independent.
type CasePublisher struct {
	db            *sqlx.DB
	embed         Embedder
	enabled       bool
	mu            sync.Mutex
	retryAfter    map[string]time.Time
	now           func() time.Time
	ReportFailure func(*CasePublishError)
}

func NewCasePublisher(db *sqlx.DB, embed Embedder, enabled bool) *CasePublisher {
	return &CasePublisher{db: db, embed: embed, enabled: enabled, retryAfter: make(map[string]time.Time), now: time.Now}
}

type caseCandidate struct {
	ID       string
	Revision int64
	Item     Item
}

// PublishOne processes at most one item. No row lock is held across the external
// embedding request, so human rejection/blocking can finish immediately.
func (p *CasePublisher) PublishOne(ctx context.Context) error {
	if !p.enabled {
		return nil
	}
	if !p.mu.TryLock() {
		return nil
	}
	defer p.mu.Unlock()
	fail := func(id, code string, class error) error {
		if id != "" {
			p.retryAfter[id] = p.now().Add(caseRetryDelay)
		}
		return &CasePublishError{ID: id, Code: code, class: class}
	}
	if p.db == nil || p.embed == nil {
		return fail("", "case_configuration_invalid", ErrUnavailable)
	}
	excluded := []string{}
	for id, until := range p.retryAfter {
		if !p.now().Before(until) {
			delete(p.retryAfter, id)
		} else {
			excluded = append(excluded, id)
		}
	}
	// Bound retry memory without dropping exclusions and immediately retrying a
	// known bad item. Once full, resume as soon as the first cooldown expires.
	if len(excluded) >= caseRetryLimit {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, caseAttemptTimeout)
	defer cancel()
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fail("", "case_database_failed", ErrUnavailable)
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT r.id::text,r.revision,
 r.topic,r.customer_question,r.human_answer,r.feedback_reason,r.retrieval_text
 FROM public.learning_case_publisher_candidates_v11 r WHERE r.id <> ALL($1::uuid[])
 ORDER BY r.updated_at,r.id LIMIT 16`, pq.Array(excluded))
	if err != nil {
		return fail("", "case_database_failed", ErrUnavailable)
	}
	candidates := []caseCandidate{}
	for rows.Next() {
		c := caseCandidate{}
		c.Item.Content.Type = "case_example"
		err = rows.Scan(&c.ID, &c.Revision, &c.Item.Content.Topic, &c.Item.Content.Question, &c.Item.Content.Answer, &c.Item.Reason, &c.Item.Content.Retrieval)
		if err != nil {
			rows.Close()
			return fail("", "case_database_failed", ErrUnavailable)
		}
		candidates = append(candidates, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return fail("", "case_database_failed", ErrUnavailable)
	}
	for _, c := range candidates {
		var locked bool
		if err = tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock(hashtextextended('libredesk-case-v11:'||$1::text,0))`, c.ID).Scan(&locked); err != nil {
			return fail(c.ID, "case_database_failed", ErrUnavailable)
		}
		if !locked {
			continue
		}
		// Another publisher/reviewer may have finished between selection and lock.
		var current bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM public.learning_case_publisher_candidates_v11 r WHERE r.id=$1::uuid AND r.revision=$2)`, c.ID, c.Revision).Scan(&current); err != nil {
			return fail(c.ID, "case_database_failed", ErrUnavailable)
		}
		if !current {
			return fail(c.ID, "case_revision_conflict", ErrConflict)
		}
		if err = ValidateCaseExample(c.Item); err != nil {
			return fail(c.ID, "case_validation_failed", ErrInput)
		}
		vector, err := p.embed(ctx, c.Item.Content.Retrieval)
		if err != nil {
			return fail(c.ID, "case_embedding_failed", ErrUnavailable)
		}
		if !ValidVector(vector) {
			return fail(c.ID, "case_vector_invalid", ErrInput)
		}
		var revision int64
		err = tx.QueryRowContext(ctx, `SELECT public.learning_case_publish_v11($1::uuid,$2::bigint,$3::public.vector)`, c.ID, c.Revision, vectorLiteral(vector)).Scan(&revision)
		if err != nil {
			var dbErr *pq.Error
			if errors.As(err, &dbErr) && (dbErr.Code == "P0001" || dbErr.Code == "40001") {
				return fail(c.ID, "case_revision_conflict", ErrConflict)
			}
			return fail(c.ID, "case_database_failed", ErrUnavailable)
		}
		if err = tx.Commit(); err != nil {
			return fail(c.ID, "case_database_failed", ErrUnavailable)
		}
		delete(p.retryAfter, c.ID)
		return nil
	}
	return nil
}

func (p *CasePublisher) Run(ctx context.Context) {
	if !p.enabled {
		return
	}
	ticker := time.NewTicker(casePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := p.PublishOne(ctx)
			if ctx.Err() != nil {
				return
			}
			var failure *CasePublishError
			if errors.As(err, &failure) && p.ReportFailure != nil {
				p.ReportFailure(failure)
			}
		}
	}
}
