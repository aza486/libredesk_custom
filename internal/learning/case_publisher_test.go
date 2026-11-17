package learning

import (
	"context"
	"errors"
	"math"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func caseVector() []float64 { v := make([]float64, 1536); v[0] = 1; return v }
func approvedPublisherCase(t *testing.T, m *Manager) (string, string) {
	t.Helper()
	c := stagedCaseV11()
	c["feedback_reason"] = ""
	c["has_explicit_feedback"] = false
	fid := caseFeedbackV11(t, m, c)
	var id string
	if err := m.DB.Get(&id, `SELECT learning_case_stage_v11($1)`, fid); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReviewCase(context.Background(), id, "42", CaseDecision{Action: "approve", Checks: CaseChecks{Privacy: true, Quality: true, Generalization: true}}); err != nil {
		t.Fatal(err)
	}
	return id, fid
}
func requireCaseFailure(t *testing.T, err error, code string) {
	t.Helper()
	var failure *CasePublishError
	if !errors.As(err, &failure) || failure.Code != code || strings.Contains(err.Error(), "SENSITIVE") {
		t.Fatal("unsafe or incorrect diagnosis", err)
	}
}
func TestCasePublisherDefaultDisabled(t *testing.T) {
	p := NewCasePublisher(nil, func(context.Context, string) ([]float64, error) { t.Fatal("disabled embedder called"); return nil, nil }, false)
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { p.Run(context.Background()); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("disabled worker did not exit")
	}
	requireCaseFailure(t, NewCasePublisher(nil, nil, true).PublishOne(context.Background()), "case_configuration_invalid")
}

func TestPostgresCasePublisherOnlyApprovedCases(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	// None of these states may reach the embedder.
	for _, decision := range []string{"staged", "rejected", "blocked", "disabled"} {
		fid := caseFeedbackV11(t, m, stagedCaseV11())
		var id string
		m.DB.Get(&id, `SELECT learning_case_stage_v11($1)`, fid)
		if decision != "staged" {
			m.DB.MustExec(`SELECT learning_case_review_v11($1,0,$2,'42')`, id, decision)
		}
	}
	m.DB.MustExec(`INSERT INTO rag_items(id,source_type,quality_status,approved,active,retrieval_text) VALUES(gen_random_uuid(),'employee_fact','approved',false,false,'A different fact question'),(gen_random_uuid(),'response_example','approved',false,false,'A different style question'),(gen_random_uuid(),'classification_example','approved',false,false,'A different category question')`)
	var calls atomic.Int32
	p := NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) { calls.Add(1); return caseVector(), nil }, true)
	if err := p.PublishOne(context.Background()); err != nil || calls.Load() != 0 {
		t.Fatal("unapproved item embedded", err, calls.Load())
	}
	id, _ := approvedPublisherCase(t, m)
	var retrieval string
	m.DB.Get(&retrieval, `SELECT retrieval_text FROM rag_items WHERE id=$1`, id)
	p.embed = func(_ context.Context, text string) ([]float64, error) {
		calls.Add(1)
		if text != retrieval {
			t.Fatal("embedded something other than retrieval_text")
		}
		return caseVector(), nil
	}
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := p.PublishOne(context.Background()); err != nil || calls.Load() != 1 {
		t.Fatal("duplicate embedding", calls.Load(), err)
	}
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, id)
	if n != 1 {
		t.Fatal("published case missing")
	}
	m.DB.Get(&n, `SELECT count(*) FROM learning_case_audit_v11 WHERE item_id=$1 AND action='publish'`, id)
	if n != 1 {
		t.Fatal("duplicate publication audit")
	}
	var reviewer string
	m.DB.Get(&reviewer, `SELECT learning_data->>'reviewer' FROM rag_items WHERE id=$1`, id)
	if reviewer != "42" {
		t.Fatal("publisher changed human approval")
	}
	m.DB.Get(&n, `SELECT count(*) FROM learning_reviews WHERE item_id=$1`, id)
	if n != 0 {
		t.Fatal("case entered fact path")
	}
}

func TestPostgresCasePublisherRetryAndValidation(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	id, _ := approvedPublisherCase(t, m)
	now := time.Now()
	var calls atomic.Int32
	p := NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) {
		calls.Add(1)
		return nil, errors.New("SENSITIVE_customer_text_and_token")
	}, true)
	p.now = func() time.Time { return now }
	err := p.PublishOne(context.Background())
	requireCaseFailure(t, err, "case_embedding_failed")
	if strings.Contains(err.Error(), "SENSITIVE") || strings.Contains(errors.Unwrap(err).Error(), "SENSITIVE") {
		t.Fatal("upstream body retained")
	}
	if err := p.PublishOne(context.Background()); err != nil || calls.Load() != 1 {
		t.Fatal("retry without cooldown", err)
	}
	// A failing old item must not starve another approved case.
	other, _ := approvedPublisherCase(t, m)
	p.embed = func(context.Context, string) ([]float64, error) { calls.Add(1); return caseVector(), nil }
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, other)
	if n != 1 {
		t.Fatal("other case starved")
	}
	now = now.Add(caseRetryDelay)
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, id)
	if n != 1 || calls.Load() != 3 {
		t.Fatal("retry did not recover", n, calls.Load())
	}
}

func TestPostgresCasePublisherHumanApprovedPlaceholder(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	var calls atomic.Int32
	p := NewCasePublisher(m.DB, nil, true)
	// Human-approved placeholders must pass both review and publication unchanged.
	// Each of these labels triggered the former automatic privacy guard.
	c := stagedCaseV11()
	c["customer_question"] = "Wie wurde die Anfrage zur Bestellnummer: [Bestellnummer] gelöst?"
	c["human_answer"] = "Im historischen Fall wurde Bestellnummer: [Bestellnummer] geprüft. Daraus folgt keine heutige Zusage."
	c["feedback_reason"] = "Bestellnummer: [Bestellnummer] wurde verallgemeinert."
	fid := caseFeedbackV11(t, m, c)
	var placeholderID string
	if err := m.DB.Get(&placeholderID, `SELECT learning_case_stage_v11($1)`, fid); err != nil {
		t.Fatal(err)
	}
	if _, err := m.ReviewCase(context.Background(), placeholderID, "42", CaseDecision{Action: "approve", Checks: CaseChecks{Privacy: true, Generalization: true, Quality: true}}); err != nil {
		t.Fatal("placeholder review", err)
	}
	p.embed = func(_ context.Context, text string) ([]float64, error) {
		calls.Add(1)
		if text != c["customer_question"] {
			t.Fatal("approved retrieval text changed")
		}
		return caseVector(), nil
	}
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal("placeholder publication", err)
	}
	var published bool
	if err := m.DB.Get(&published, `SELECT active AND approved AND embedding IS NOT NULL
 AND learning_data->>'reviewer'='42' AND learning_data->>'case_revision'='2'
 AND customer_question=$2 AND human_answer=$3 AND feedback_reason=$4
 FROM rag_items WHERE id=$1`, placeholderID, c["customer_question"], c["human_answer"], c["feedback_reason"]); err != nil || !published || calls.Load() != 1 {
		t.Fatal("approved placeholder or human approval changed", err, calls.Load())
	}
	var n int
	if err := m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, placeholderID); err != nil || n != 1 {
		t.Fatal("published placeholder missing from retrieval", err)
	}
}

func TestPostgresCasePublisherInvalidVectors(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	id, _ := approvedPublisherCase(t, m)
	for _, vector := range [][]float64{nil, {1, 2}, make([]float64, 1536), func() []float64 { v := caseVector(); v[0] = math.NaN(); return v }(), func() []float64 { v := caseVector(); v[0] = math.Inf(1); return v }()} {
		p := NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) { return vector, nil }, true)
		requireCaseFailure(t, p.PublishOne(context.Background()), "case_vector_invalid")
	}
	var active bool
	m.DB.Get(&active, `SELECT active OR embedding IS NOT NULL FROM rag_items WHERE id=$1`, id)
	if active {
		t.Fatal("invalid vector published")
	}
}

func TestPostgresCasePublisherParallelChanges(t *testing.T) {
	for _, decision := range []string{"rejected", "blocked", "disabled", "source_blocked", "other_publisher"} {
		t.Run(decision, func(t *testing.T) {
			m := isolated(t)
			m.DB.MustExec(CaseV11Schema)
			m.DB.MustExec(CasePublisherV11Schema)
			id, fid := approvedPublisherCase(t, m)
			started, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			p := NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) {
				calls.Add(1)
				close(started)
				<-release
				return caseVector(), nil
			}, true)
			result := make(chan error, 1)
			go func() { result <- p.PublishOne(context.Background()) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("embedding did not start")
			}
			// A different worker obeying the same advisory lock cannot duplicate work.
			second := NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) { calls.Add(1); return caseVector(), nil }, true)
			if err := second.PublishOne(context.Background()); err != nil || calls.Load() != 1 {
				t.Fatal("duplicate parallel embed", err, calls.Load())
			}
			// Review must finish before release: no row lock across the external call.
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var err error
			switch decision {
			case "source_blocked":
				_, err = m.DB.ExecContext(ctx, `UPDATE ai_feedback SET feedback_data=jsonb_set(feedback_data,'{learning_case_v11}','{"version":11,"status":"blocked","reason_code":"privacy"}') WHERE id=$1`, fid)
			case "other_publisher":
				_, err = m.DB.ExecContext(ctx, `SELECT learning_case_publish_v11($1,1,$2::vector)`, id, vectorLiteral(caseVector()))
			default:
				_, err = m.DB.ExecContext(ctx, `SELECT learning_case_review_v11($1,1,$2,'42')`, id, decision)
			}
			close(release)
			if err != nil {
				t.Fatal("review blocked by embedder", err)
			}
			select {
			case err = <-result:
			case <-time.After(5 * time.Second):
				t.Fatal("publisher did not finish")
			}
			requireCaseFailure(t, err, "case_revision_conflict")
			var n int
			m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, id)
			want := 0
			if decision == "other_publisher" {
				want = 1
			}
			if n != want {
				t.Fatal("stale result published", decision, n)
			}
		})
	}
}

func TestPostgresCasePublisherDatabaseFailureAndCancellation(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	id, _ := approvedPublisherCase(t, m)
	db, err := sqlx.Open("postgres", os.Getenv("LIBREDESK_LEARNING_TEST_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	p := NewCasePublisher(db, func(context.Context, string) ([]float64, error) {
		t.Fatal("database failure reached embedder")
		return nil, nil
	}, true)
	requireCaseFailure(t, p.PublishOne(context.Background()), "case_database_failed")
	ctx, cancel := context.WithCancel(context.Background())
	p = NewCasePublisher(m.DB, func(ctx context.Context, _ string) ([]float64, error) { cancel(); <-ctx.Done(); return nil, ctx.Err() }, true)
	requireCaseFailure(t, p.PublishOne(ctx), "case_embedding_failed")
	// Advisory lock/transaction must be released after cancellation.
	p = NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) { return caseVector(), nil }, true)
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal(err)
	}
	var active bool
	m.DB.Get(&active, `SELECT active FROM rag_items WHERE id=$1`, id)
	if !active {
		t.Fatal("lock leaked after cancellation")
	}
}

func TestPostgresCasePublisherDatabaseLostAfterEmbedding(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	id, _ := approvedPublisherCase(t, m)
	target, _ := url.Parse(os.Getenv("LIBREDESK_LEARNING_TEST_DSN"))
	q := target.Query()
	q.Set("application_name", "case_publisher_disconnect_test")
	target.RawQuery = q.Encode()
	db, err := sqlx.Connect("postgres", target.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	p := NewCasePublisher(db, func(context.Context, string) ([]float64, error) {
		var killed bool
		if err := m.DB.Get(&killed, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name='case_publisher_disconnect_test' AND pid<>pg_backend_pid()`); err != nil || !killed {
			t.Fatal("could not interrupt synthetic publisher", err)
		}
		return caseVector(), nil
	}, true)
	requireCaseFailure(t, p.PublishOne(context.Background()), "case_database_failed")
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM learning_case_audit_v11 WHERE item_id=$1 AND action='publish'`, id)
	if n != 0 {
		t.Fatal("partial publication committed")
	}
	// A fresh attempt safely reconnects; no intermediate state repair is needed.
	p = NewCasePublisher(db, func(context.Context, string) ([]float64, error) { return caseVector(), nil }, true)
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal("reconnect", err)
	}
}

func TestPostgresCasePublisherRestrictedRole(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	id, fid := approvedPublisherCase(t, m)
	m.DB.MustExec(`DO $$ BEGIN IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='case_v11_technical_publisher') THEN CREATE ROLE case_v11_technical_publisher LOGIN PASSWORD 'isolated_case_role_only' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION; END IF; END $$;
 GRANT USAGE ON SCHEMA public TO case_v11_technical_publisher;
 GRANT SELECT ON public.learning_case_publisher_candidates_v11 TO case_v11_technical_publisher;
 GRANT EXECUTE ON FUNCTION learning_case_publish_v11(uuid,bigint,vector) TO case_v11_technical_publisher;`)
	target, _ := url.Parse(os.Getenv("LIBREDESK_LEARNING_TEST_DSN"))
	target.User = url.UserPassword("case_v11_technical_publisher", "isolated_case_role_only")
	restricted, err := sqlx.Connect("postgres", target.String())
	if err != nil {
		t.Fatal(err)
	}
	defer restricted.Close()
	for _, q := range []string{`SELECT * FROM public.rag_items`, `SELECT id FROM public.rag_items`, `SELECT * FROM public.ai_feedback`, `SELECT id FROM public.ai_feedback`, `SELECT human_answer FROM ai_feedback`, `SELECT learning_case_stage_v11('` + fid + `')`, `UPDATE rag_items SET active=true WHERE id='` + id + `'`, `SELECT learning_case_review_v11('` + id + `',1,'approved','spoof',true,true,true)`} {
		if _, err := restricted.Exec(q); err == nil {
			t.Fatal("publisher role can read raw source or review/write", q)
		}
	}
	var baseRead bool
	if err := restricted.Get(&baseRead, `SELECT has_any_column_privilege(current_user,'public.rag_items','SELECT') OR has_any_column_privilege(current_user,'public.ai_feedback','SELECT')`); err != nil || baseRead {
		t.Fatal("unexpected base-column privilege", baseRead, err)
	}
	var selectedID string
	if err := restricted.Get(&selectedID, `SELECT id FROM public.learning_case_publisher_candidates_v11`); err != nil || selectedID != id {
		t.Fatal("view SELECT", selectedID, err)
	}
	// Rerunning SQL006 preserves explicit role SELECT and keeps PUBLIC revoked.
	m.DB.MustExec(CasePublisherV11Schema)
	p := NewCasePublisher(restricted, func(context.Context, string) ([]float64, error) { return caseVector(), nil }, true)
	if err := p.PublishOne(context.Background()); err != nil {
		t.Fatal("minimum role could not publish", err)
	}
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, id)
	if n != 1 {
		t.Fatal("restricted publication missing")
	}
}
