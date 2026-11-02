package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func validContent() Content {
	return Content{Type: "employee_fact", Category: "firmenfragen", Topic: "Öffnungszeiten", Retrieval: "Wann ist unser Büro geöffnet?", Question: "Wann ist das Büro geöffnet?", Rule: "Das Büro ist samstags geschlossen.", Evidence: "Das Büro ist samstags geschlossen.", Source: "Internes geprüftes Firmenhandbuch", Limits: "Nur Büro, nicht Online-Bestellungen"}
}
func allChecks() Checks {
	return Checks{Privacy: true, Content: true, General: true, Correction: true, StyleOnly: true}
}
func TestValidation(t *testing.T) {
	c := validContent()
	if err := Validate(c, allChecks(), true); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"Stornierung ihrer Bestellung", "Wir erstatten den Betrag", "Garantiert morgen geliefert", "Produktion stoppen", "Ihre Bestellung wurde versendet", "Wir können die Rücksendung bearbeiten", "Sicher bis Freitag geliefert"} {
		bad := c
		bad.Rule = field
		bad.Evidence = field
		if Validate(bad, allChecks(), true) == nil {
			t.Fatal("unsafe claim accepted", field)
		}
	}
	if Validate(c, Checks{}, true) == nil {
		t.Fatal("unreviewed fact")
	}
	c.Type = "factual_review"
	if !errors.Is(Validate(c, allChecks(), true), ErrConflict) {
		t.Fatal("blind factual review approval")
	}
	c = validContent()
	c.Type = "classification_example"
	c.OriginalCategories = []string{"bestellung"}
	c.CorrectedCategories = []string{"stornierung"}
	if Validate(c, allChecks(), true) == nil {
		t.Fatal("unknown category")
	}
	c.CorrectedCategories = []string{"bestellung"}
	if Validate(c, allChecks(), true) == nil {
		t.Fatal("no correction")
	}
	c.CorrectedCategories = []string{"human_needed"}
	if Validate(c, allChecks(), true) != nil {
		t.Fatal("explicit human correction blocked")
	}
	c = validContent()
	c.Retrieval = "Contact person@example.com"
	if Validate(c, allChecks(), false) == nil {
		t.Fatal("PII allowed")
	}
	for _, v := range [][]float64{nil, make([]float64, 1536), {math.NaN()}} {
		if ValidVector(v) {
			t.Fatal("invalid vector")
		}
	}
}
func isolated(t *testing.T) *Manager {
	t.Helper()
	dsn := os.Getenv("LIBREDESK_LEARNING_TEST_DSN")
	if dsn == "" {
		t.Skip("set explicit isolated LIBREDESK_LEARNING_TEST_DSN")
	}
	target, parseErr := url.Parse(dsn)
	if parseErr != nil || target.Hostname() != "127.0.0.1" || target.Port() != "55437" || target.Path != "/learning_test" {
		t.Fatal("refusing schema reset outside explicitly isolated learning_test on 127.0.0.1:55437")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.MustExec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public; CREATE EXTENSION vector;
 CREATE TABLE mail_events(message_uuid text,clean_text text,categories text[]);
 CREATE TABLE ai_feedback(id uuid PRIMARY KEY,mail_event_message_uuid text,outgoing_message_uuid text,agent_id text,feedback_reason text,human_answer text,feedback_type text,quality_status text DEFAULT 'pending',created_at timestamp DEFAULT now(),updated_at timestamp DEFAULT now(),feedback_data jsonb DEFAULT '{}',learning_status text,learning_processed_at timestamp);
 CREATE TABLE rag_items(id uuid PRIMARY KEY,source_type text,scope text,agent_id text,category text,subcategory text,topic text,customer_question text,ai_answer text,human_answer text,feedback_reason text,retrieval_text text,embedding vector(1536),quality_status text DEFAULT 'pending',approved boolean DEFAULT false,active boolean DEFAULT false,source_id text,created_at timestamp DEFAULT now(),updated_at timestamp DEFAULT now(),feedback_id uuid REFERENCES ai_feedback(id),learning_data jsonb DEFAULT '{}');`)
	if _, err = db.Exec(Schema); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(Schema); err != nil {
		t.Fatal("non-idempotent schema", err)
	}
	return &Manager{DB: db, Anonymize: func(_ context.Context, s string) (string, error) {
		return strings.ReplaceAll(s, "Private Name", "[NAME]"), nil
	}, Embed: func(context.Context, string) ([]float64, error) { v := make([]float64, 1536); v[0] = 1; return v, nil }}
}
func candidate(t *testing.T, m *Manager, c Content) string {
	t.Helper()
	id := uuid.NewString()
	m.DB.MustExec(`INSERT INTO rag_items(id,source_type,category,topic,retrieval_text,customer_question,human_answer) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, c.Type, c.Category, c.Topic, c.Retrieval, c.Question, c.Answer)
	tx := m.DB.MustBegin()
	if e := writeContent(context.Background(), tx, id, c); e != nil {
		t.Fatal(e)
	}
	if e := tx.Commit(); e != nil {
		t.Fatal(e)
	}
	return id
}
func TestPostgresReviewLifecycle(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := validContent()
	id := candidate(t, m, c)
	req := Mutation{Action: "approve", Reason: "Handbuch und allgemeine Gültigkeit geprüft", Checks: allChecks()}
	if _, e := m.Mutate(ctx, id, "42", req); e != nil {
		t.Fatal(e)
	}
	item, e := m.Get(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if item.Active || item.Approved || item.Publish != "unpublished" || item.Decision != "approved" {
		t.Fatalf("premature activation %+v", item)
	}
	var calls atomic.Int32
	originalEmbed := m.Embed
	m.Embed = func(c context.Context, s string) ([]float64, error) { calls.Add(1); return originalEmbed(c, s) }
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := m.PublishOne(ctx); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	item, e = m.Get(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if !item.Active || !item.Approved || item.Publish != "published" || calls.Load() != 1 {
		t.Fatalf("publication %+v calls=%d", item, calls.Load())
	}
	var verified, approved string
	if e = m.DB.QueryRow(`SELECT learning_data->>'verification_status',learning_data->>'factual_use_approved' FROM rag_items WHERE id=$1`, id).Scan(&verified, &approved); e != nil || verified != "human_verified" || approved != "true" {
		t.Fatal("fact guards", e, verified, approved)
	}
	_, e = m.Mutate(ctx, id, "42", Mutation{Action: "edit", Revision: item.Revision, Reason: "Edit published content"})
	if !errors.Is(e, ErrInput) {
		t.Fatal("removed editor accepted", e)
	}
	_, e = m.Mutate(ctx, id, "42", Mutation{Action: "deactivate", Revision: item.Revision, Reason: "Richtlinie nicht mehr aktuell"})
	if e != nil {
		t.Fatal(e)
	}
	item, e = m.Get(ctx, id)
	if e != nil || item.Active || item.Approved || item.Decision != "disabled" {
		t.Fatal("disable", e, item)
	}
	if _, e = m.Mutate(ctx, id, "42", Mutation{Action: "approve", Revision: item.Revision, Reason: "Erneut geprüft und bestätigt", Checks: Checks{Privacy: true, Content: true}}); !errors.Is(e, ErrInput) {
		t.Fatal("reactivation bypass", e)
	}
	// Original and audit survive, but go through the same privacy boundary.
	if len(item.History) != 3 || len(item.Original) == 0 {
		t.Fatalf("audit missing %+v", item)
	}
	if _, e = m.DB.Exec(Schema); e != nil {
		t.Fatal(e)
	}
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM learning_review_audit`)
	if n != 3 {
		t.Fatal("schema reinstall lost audit")
	}
}
func TestPostgresEditRetryConflictAndPrivacy(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := validContent()
	id := candidate(t, m, c)
	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := m.Mutate(ctx, id, "42", Mutation{Action: "approve", Reason: "Geprüftes Firmenhandbuch", Checks: allChecks()})
			if e == nil {
				wins.Add(1)
			} else if !errors.Is(e, ErrConflict) {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatal("concurrent approve", wins.Load())
	}
	m.Embed = func(context.Context, string) ([]float64, error) { return []float64{1}, nil }
	if e := m.PublishOne(ctx); !errors.Is(e, ErrInput) {
		t.Fatal("invalid vector accepted", e)
	}
	item, e := m.Get(ctx, id)
	if e != nil || item.Active || item.Publish != "failed" {
		t.Fatal("failed gating", e, item)
	}
	var emb sql.NullString
	m.DB.Get(&emb, `SELECT embedding::text FROM rag_items WHERE id=$1`, id)
	if emb.Valid {
		t.Fatal("invalid embedding retained")
	}
	_, e = m.Mutate(ctx, id, "42", Mutation{Action: "reject", Revision: item.Revision, Reason: "Quelle nicht ausreichend belegt"})
	if e != nil {
		t.Fatal(e)
	}
	m.Anonymize = nil
	item, e = m.Get(ctx, id)
	if e != nil || item.Content.Rule != c.Rule {
		t.Fatal("display unnecessarily anonymized", e, item)
	}
}
func TestPostgresPublicationFenceAndViews(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := validContent()
	id := candidate(t, m, c)
	_, e := m.Mutate(ctx, id, "42", Mutation{Action: "approve", Reason: "Quelle geprüft vor Veröffentlichung", Checks: allChecks()})
	if e != nil {
		t.Fatal(e)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	m.Embed = func(context.Context, string) ([]float64, error) {
		close(entered)
		<-release
		v := make([]float64, 1536)
		v[0] = 1
		return v, nil
	}
	done := make(chan error, 1)
	go func() { done <- m.PublishOne(ctx) }()
	<-entered
	_, e = m.Mutate(ctx, id, "43", Mutation{Action: "reject", Revision: 1, Reason: "Concurrent decision"})
	if !errors.Is(e, ErrConflict) {
		t.Fatal("publication race", e)
	}
	close(release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	views, e := os.ReadFile("artifacts/002_style_view.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(views))
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_verified_fact_view`)
	if n != 1 {
		t.Fatal("verified fact not retrieved")
	}
	item, e := m.Get(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(item)
	if strings.Contains(string(b), `"embedding":`) {
		t.Fatal("embedding or raw snapshot leaked")
	}
}

func TestPostgresRetryPreservesApprovalAndContentChange(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := candidate(t, m, validContent())
	_, e := m.Mutate(ctx, id, "42", Mutation{Action: "approve", Reason: "Quellen überprüft und bestätigt", Checks: allChecks()})
	if e != nil {
		t.Fatal(e)
	}
	m.Embed = func(context.Context, string) ([]float64, error) { return nil, errors.New("timeout") }
	if e = m.PublishOne(ctx); e == nil {
		t.Fatal("timeout accepted")
	}
	var first []byte
	m.DB.Get(&first, `SELECT checks FROM learning_reviews WHERE item_id=$1`, id)
	_, e = m.Mutate(ctx, id, "43", Mutation{Action: "retry", Revision: 1, Reason: "Veröffentlichung erneut versuchen"})
	if e != nil {
		t.Fatal(e)
	}
	var after []byte
	m.DB.Get(&after, `SELECT checks FROM learning_reviews WHERE item_id=$1`, id)
	if string(first) != string(after) {
		t.Fatal("retry erased approval checks")
	}
	m.Embed = func(context.Context, string) ([]float64, error) { v := make([]float64, 1536); v[0] = 1; return v, nil }
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	// A stale publication result cannot overwrite a newer lease.
	other := candidate(t, m, validContent())
	_, e = m.Mutate(ctx, other, "42", Mutation{Action: "approve", Reason: "Quelle erneut überprüft", Checks: allChecks()})
	if e != nil {
		t.Fatal(e)
	}
	m.Embed = func(context.Context, string) ([]float64, error) {
		m.DB.MustExec(`UPDATE learning_reviews SET lease=$2::uuid WHERE item_id=$1`, other, uuid.NewString())
		v := make([]float64, 1536)
		v[0] = 1
		return v, nil
	}
	if e = m.PublishOne(ctx); !errors.Is(e, ErrConflict) {
		t.Fatal("stale lease accepted", e)
	}
	var active bool
	m.DB.Get(&active, `SELECT active FROM rag_items WHERE id=$1`, other)
	if active {
		t.Fatal("stale result activated")
	}
}
func TestPostgresRetrievalAndV9PublicationContract(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	views, e := os.ReadFile("testdata/legacy_views.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(views))
	c := validContent()
	c.Type = "response_example"
	c.Answer = "Guten Tag, vielen Dank für Ihre freundliche Nachricht. Hier finden Sie die Bürozeiten."
	id := candidate(t, m, c)
	_, e = m.Mutate(ctx, id, "42", Mutation{Action: "approve", Reason: "Nur Stil und Struktur geprüft", Checks: allChecks()})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	for _, view := range []string{"rag_category_learning_view", "rag_final_style_view"} {
		var count int
		if e = m.DB.Get(&count, `SELECT count(*) FROM `+view+` WHERE id=$1`, id); e != nil || count != 1 {
			t.Fatal("view contract", view, count, e)
		}
	}
	c.Type = "classification_example"
	c.OriginalCategories = []string{"bestellung"}
	c.CorrectedCategories = []string{"firmenfragen"}
	c.Evidence = "Original bestellung; ausdrücklich korrigiert zu firmenfragen, geprüft anhand der Bürozeitenfrage."
	cid := candidate(t, m, c)
	_, e = m.Mutate(ctx, cid, "42", Mutation{Action: "approve", Reason: "Ausdrückliche Korrektur geprüft", Checks: allChecks()})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	var count int
	m.DB.Get(&count, `SELECT count(*) FROM rag_category_learning_view WHERE id=$1`, cid)
	if count != 1 {
		t.Fatal("classifier missing from real view")
	}
	raw, e := os.ReadFile("artifacts/AI_Learning_Processor_v9.json")
	if e != nil {
		t.Fatal(e)
	}
	var workflow struct {
		Nodes []struct {
			Name       string `json:"name"`
			Parameters struct {
				Query string `json:"query"`
			} `json:"parameters"`
		} `json:"nodes"`
	}
	if e = json.Unmarshal(raw, &workflow); e != nil {
		t.Fatal(e)
	}
	queries := map[string]string{}
	for _, node := range workflow.Nodes {
		queries[node.Name] = node.Parameters.Query
	}
	feedback, auto := uuid.NewString(), uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,learning_status) VALUES($1,'prepared')`, feedback)
	m.DB.MustExec(`INSERT INTO rag_items(id,source_type,category,topic,retrieval_text,human_answer,feedback_id,learning_data) VALUES($1,'response_example','firmenfragen','Büro','Wann ist das Büro geöffnet?','Vielen Dank für die Nachricht. Hier finden Sie alle Bürozeiten.',$2,'{"target_agent":"final_answer","example_role":"style_only","factual_authority":false}')`, auto, feedback)
	var claimed struct {
		ID        string `db:"rag_item_id"`
		Retrieval string `db:"retrieval_text"`
		Type      string `db:"source_type"`
		Category  string `db:"category"`
		Data      []byte `db:"learning_data"`
		Feedback  string `db:"feedback_id"`
		Answer    string `db:"human_answer"`
	}
	if e = m.DB.Get(&claimed, queries["Claim RAG Publication"]); e != nil || claimed.ID != auto {
		t.Fatal("v9 claim contract", e, claimed.ID)
	}
	m.DB.MustExec(`CREATE TABLE learning_embedding_stage(id uuid PRIMARY KEY,embedding vector(1536),metadata jsonb)`)
	v := make([]float64, 1536)
	v[0] = 1
	m.DB.MustExec(`INSERT INTO learning_embedding_stage VALUES($1,$2::vector,jsonb_build_object('rag_item_id',$3::text))`, uuid.NewString(), vectorLiteral(v), auto)
	payload, _ := json.Marshal(map[string]string{"rag_item_id": auto})
	rows, e := m.DB.Queryx(queries["Finalize RAG Publication"], string(payload))
	if e != nil {
		t.Fatal("v9 finalize", e)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("missing finalize response")
	}
	result := map[string]any{}
	if e = rows.MapScan(result); e != nil {
		t.Fatal(e)
	}
	var active bool
	m.DB.Get(&active, `SELECT active FROM rag_items WHERE id=$1`, auto)
	if !active {
		t.Fatal("existing automatic non-fact publisher broken")
	}
}
func TestPostgresStyleRuleAndLeastPrivilege(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := validContent()
	c.Type = "style_rule"
	c.Rule = "Antworten kurz und freundlich formulieren."
	c.Evidence = ""
	c.Source = ""
	c.Limits = ""
	id := candidate(t, m, c)
	_, e := m.Mutate(ctx, id, "42", Mutation{Action: "approve", Reason: "Geeigneten Stilhinweis geprüft", Checks: allChecks()})
	if e != nil {
		t.Fatal(e)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	sqlViews, e := os.ReadFile("artifacts/002_style_view.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(sqlViews))
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_final_style_view WHERE id=$1`, id)
	if n != 1 {
		t.Fatal("style rule not retrievable")
	}
	perms, e := os.ReadFile("artifacts/003_permissions.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(perms))
	var canReadRaw, canReadClean, canChangeFeedback, canChangeAudit bool
	e = m.DB.QueryRow(`SELECT has_column_privilege('libredesk_learning','mail_events','clean_text','SELECT'),has_table_privilege('libredesk_learning','ai_feedback','UPDATE'),has_table_privilege('libredesk_learning','learning_review_audit','UPDATE')`).Scan(&canReadClean, &canChangeFeedback, &canChangeAudit)
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(`ALTER TABLE mail_events ADD COLUMN raw_text text`)
	m.DB.Get(&canReadRaw, `SELECT has_column_privilege('libredesk_learning','mail_events','raw_text','SELECT')`)
	if !canReadClean || canReadRaw || canChangeFeedback || canChangeAudit {
		t.Fatal("least privilege failed", canReadClean, canReadRaw, canChangeFeedback, canChangeAudit)
	}
}
func TestPostgresDedicatedRoleCanUseReviewAPI(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := candidate(t, m, validContent())
	perms, e := os.ReadFile("artifacts/003_permissions.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(perms))
	m.DB.SetMaxOpenConns(1)
	m.DB.MustExec(`SET ROLE libredesk_learning`)
	defer m.DB.MustExec(`RESET ROLE`)
	if _, e = m.List(ctx, Filter{Tab: "pending", Page: 1}); e != nil {
		t.Fatal("role cannot list", e)
	}
	if _, e = m.List(ctx, Filter{Tab: "skipped", Page: 1}); e != nil {
		t.Fatal("role cannot read skipped context", e)
	}
	if _, e = m.Mutate(ctx, id, "42", Mutation{Action: "approve", Reason: "Getrennter Review-Zugang geprüft", Checks: allChecks()}); e != nil {
		t.Fatal("role cannot approve", e)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal("role cannot publish", e)
	}
	if _, e = m.Get(ctx, id); e != nil {
		t.Fatal("role cannot read audit", e)
	}
}
func TestPostgresStatusFiltersAndPagination(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	blocked := candidate(t, m, validContent())
	legacy := candidate(t, m, validContent())
	m.DB.MustExec(`UPDATE rag_items SET learning_data=learning_data||'{"publish_state":"blocked","proposal_version":2,"review_required":true,"rationale":"Allgemeine Firmeninformation prüfen"}' WHERE id=$1`, blocked)
	m.DB.MustExec(`UPDATE rag_items SET approved=true,active=false,quality_status='approved' WHERE id=$1`, legacy)
	m.DB.MustExec(`UPDATE rag_items SET learning_data=learning_data||'{"evidence_source":"human_answer"}' WHERE id=$1`, blocked)
	m.DB.MustExec(`INSERT INTO learning_reviews(item_id,original) SELECT id,to_jsonb(r)-'embedding' FROM rag_items r WHERE id=$1`, blocked)
	page, e := m.List(ctx, Filter{Tab: "pending", Status: "blocked", Page: 1})
	if e != nil || len(page.Items) != 1 || page.Items[0].ID != blocked {
		t.Fatal("blocked filter", e, page)
	}
	page, e = m.List(ctx, Filter{Tab: "history", Page: 1})
	if e != nil || len(page.Items) != 1 || page.Items[0].ID != legacy {
		t.Fatal("legacy proposals require migration, not empty forms", e, page)
	}
	for i := 0; i < 25; i++ {
		id := candidate(t, m, validContent())
		m.DB.MustExec(`UPDATE rag_items SET learning_data=learning_data||'{"proposal_version":2,"review_required":true,"rationale":"Allgemeine Firmeninformation prüfen"}' WHERE id=$1`, id)
		m.DB.MustExec(`UPDATE rag_items SET learning_data=learning_data||'{"evidence_source":"human_answer"}' WHERE id=$1`, id)
		m.DB.MustExec(`INSERT INTO learning_reviews(item_id,original) SELECT id,to_jsonb(r)-'embedding' FROM rag_items r WHERE id=$1`, id)
	}
	page, e = m.List(ctx, Filter{Tab: "pending", Type: "employee_fact", Page: 1})
	if e != nil || len(page.Items) != 25 || !page.HasMore {
		t.Fatal("first page", e, page)
	}
	next, e := m.List(ctx, Filter{Tab: "pending", Page: 2})
	if e != nil || len(next.Items) != 1 || next.HasMore {
		t.Fatal("second page", e, next)
	}
	seen := map[string]bool{}
	for _, item := range page.Items {
		seen[item.ID] = true
	}
	for _, item := range next.Items {
		if seen[item.ID] {
			t.Fatal("pagination duplicates")
		}
	}
}

func TestDisplayMetadataSkippedSources(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	// These columns are deliberately not granted to the application role.
	m.DB.MustExec(`ALTER TABLE mail_events ADD COLUMN raw_text text, ADD COLUMN conversation_uuid text`)
	m.DB.MustExec(`INSERT INTO mail_events(message_uuid,clean_text,categories,raw_text,conversation_uuid) VALUES
 ('source-a','Private Name fragt nach Bürozeiten',ARRAY['bestellung','danksagung'],'RAW_CUSTOMER_SECRET_A','conversation-a'),
 ('foreign-source','FOREIGN_CONVERSATION_QUESTION',ARRAY['rechnung'],'RAW_CUSTOMER_SECRET_B','conversation-b'),
 ('ambiguous','DO_NOT_CHOOSE_FIRST',ARRAY['produktfragen'],'RAW_1','conversation-c'),
 ('ambiguous','DO_NOT_CHOOSE_SECOND',ARRAY['spam'],'RAW_2','conversation-d')`)
	ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
	for i, source := range []string{"source-a", "missing-source", "ambiguous"} {
		m.DB.MustExec(`INSERT INTO ai_feedback(id,mail_event_message_uuid,learning_status,feedback_reason,human_answer,feedback_data) VALUES($1,$2,'skipped','Private Name bittet um Kürzung','Private Name antwortet', '{"learning_skip_reason":"no_valid_learning_candidate"}')`, ids[i], source)
	}
	perms, e := os.ReadFile("artifacts/003_permissions.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(perms))
	m.DB.SetMaxOpenConns(1)
	m.DB.MustExec(`SET ROLE libredesk_learning`)
	defer m.DB.MustExec(`RESET ROLE`)
	page, e := m.List(ctx, Filter{Tab: "skipped", Page: 1})
	if e != nil {
		t.Fatal(e)
	}
	byID := map[string]Item{}
	for _, item := range page.Items {
		byID[item.ID] = item
	}
	matched := byID[ids[0]]
	if matched.RecordType != "feedback" || matched.LearningStatus != "skipped" || matched.CandidateCount != 0 || matched.Publish != "" {
		t.Fatalf("invented candidate state: %+v", matched)
	}
	if strings.Join(matched.Content.OriginalCategories, ",") != "bestellung,danksagung" || matched.Content.Category != "" || matched.Content.Topic != "" {
		t.Fatalf("missing or invented category/topic %+v", matched.Content)
	}
	if matched.Content.Question != "Private Name fragt nach Bürozeiten" || matched.Reason != "Private Name bittet um Kürzung" || matched.BlockReason != "no_valid_learning_candidate" {
		t.Fatalf("missing safe context %+v", matched)
	}
	for _, id := range ids[1:] {
		if byID[id].Content.Question != "" || len(byID[id].Content.OriginalCategories) != 0 {
			t.Fatal("missing or ambiguous source was guessed", byID[id])
		}
	}
	raw, _ := json.Marshal(page)
	for _, forbidden := range []string{"RAW_CUSTOMER_SECRET", "FOREIGN_CONVERSATION", "DO_NOT_CHOOSE", "conversation-b"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatal("unsafe/foreign source leaked", forbidden)
		}
	}

}
func TestDisplayMetadataCandidateAndAnonymization(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := validContent()
	id := candidate(t, m, c)
	m.DB.MustExec(`UPDATE rag_items SET source_type='manual',category='special_unknown',ai_answer='Private Name erhält einen KI-Vorschlag' WHERE id=$1`, id)
	item, e := m.Get(ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if item.RecordType != "candidate" || item.Content.Type != "manual" || item.Content.Category != "special_unknown" || item.AISuggestion != "Private Name erhält einen KI-Vorschlag" || item.ReviewExists {
		t.Fatalf("normal fields incorrectly replaced: %+v", item)
	}
	page, e := m.List(ctx, Filter{Tab: "history", Type: "manual", Page: 1})
	if e != nil || len(page.Items) != 1 {
		t.Fatal("manual read filter", e, page)
	}
	// Display-only support must not expand the mutation/type/category contract.
	c.Type = "manual"
	if Validate(c, allChecks(), true) == nil {
		t.Fatal("manual display type enabled new approvals")
	}
	c = validContent()
	c.Category = "danksagung"
	if Validate(c, allChecks(), false) == nil {
		t.Fatal("display category changed write policy")
	}
	m.Anonymize = nil
	item, e = m.Get(ctx, id)
	if e != nil || item.AISuggestion != "Private Name erhält einen KI-Vorschlag" || item.Content.Rule != c.Rule {
		t.Fatal("display still requires anonymizer", e, item)
	}
}
