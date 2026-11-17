package learning

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func stagedCaseV11() map[string]any {
	return map[string]any{"version": 11, "status": "staged", "customer_question": "Wie lässt sich ein beschädigter Artikel reklamieren?", "human_answer": "Im historischen Fall wurde der Schaden geprüft und ein Ersatz angeboten. Daraus folgt kein aktueller Anspruch.", "feedback_reason": "Die Mitarbeiterlösung erklärt den damaligen Lösungsweg.", "feedback_type": "response_feedback", "original_categories": []string{"reklamation"}, "corrected_categories": []string{}, "has_human_resolution": true, "has_explicit_feedback": true, "requires_quality_review": true}
}

func TestCaseV11StructuralValidation(t *testing.T) {
	item := Item{Content: Content{Type: "case_example", Question: "Wie wurde der historische Fall gelöst?", Answer: "Im historischen Fall wurde Bestellnummer: [Bestellnummer] geprüft.", Retrieval: "Frage zur Bestellnummer: [Bestellnummer]"}, Reason: "Bestellnummer: [Bestellnummer]"}
	if err := ValidateCaseExample(item); err != nil {
		t.Fatal("placeholder rejected by automatic privacy check", err)
	}
	// The same content remains subject to automatic privacy checks on the fact path.
	fact := validContent()
	fact.Retrieval = item.Content.Retrieval
	if err := Validate(fact, allChecks(), true); !errors.Is(err, ErrInput) || DiagnosticCode(err) != "privacy_identifier" {
		t.Fatal("fact privacy guard changed", err)
	}
	for _, tt := range []struct {
		name   string
		change func(*Item)
	}{
		{"type", func(i *Item) { i.Content.Type = "employee_fact" }},
		{"question missing", func(i *Item) { i.Content.Question = "" }},
		{"question short", func(i *Item) { i.Content.Question = "Kurz" }},
		{"question long", func(i *Item) { i.Content.Question = strings.Repeat("ä", 6001) }},
		{"answer missing", func(i *Item) { i.Content.Answer = " \n" }},
		{"answer long", func(i *Item) { i.Content.Answer = strings.Repeat("ä", 6001) }},
		{"retrieval missing", func(i *Item) { i.Content.Retrieval = "" }},
		{"retrieval short", func(i *Item) { i.Content.Retrieval = "Kurz" }},
		{"retrieval long", func(i *Item) { i.Content.Retrieval = strings.Repeat("ä", 901) }},
		{"reason long", func(i *Item) { i.Reason = strings.Repeat("ä", 6001) }},
		{"invalid encoding", func(i *Item) { i.Content.Answer = string([]byte{0xff}) }},
		{"null byte", func(i *Item) { i.Reason = "Grund\x00" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bad := item
			tt.change(&bad)
			if err := ValidateCaseExample(bad); !errors.Is(err, ErrInput) || DiagnosticCode(err) != "case_fields_invalid" {
				t.Fatal("invalid structure accepted", err)
			}
		})
	}
	item.Reason = ""
	if err := ValidateCaseExample(item); err != nil {
		t.Fatal("optional reason rejected", err)
	}
}

func TestPostgresCaseV11Lifecycle(t *testing.T) {
	m := isolated(t)
	legacy, e := os.ReadFile("testdata/legacy_views.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(legacy))
	views, e := os.ReadFile("artifacts/002_style_view.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(string(views))
	viewDefs := func() string {
		var s string
		if e := m.DB.Get(&s, `SELECT string_agg(definition,'|' ORDER BY viewname) FROM pg_views WHERE schemaname='public' AND viewname IN ('rag_verified_fact_view','rag_final_style_view','rag_category_learning_view')`); e != nil {
			t.Fatal(e)
		}
		return s
	}
	before := viewDefs()
	for i := 0; i < 2; i++ {
		if _, e := m.DB.Exec(CaseV11Schema); e != nil {
			t.Fatal("migration", i, e)
		}
	}
	if before != viewDefs() {
		t.Fatal("existing retrieval views changed")
	}
	mustFail := func(q string, args ...any) {
		t.Helper()
		if _, e := m.DB.Exec(q, args...); e == nil {
			t.Fatal("unsafe statement accepted", q)
		}
	}
	fid := uuid.NewString()
	payload, _ := json.Marshal(map[string]any{"response_source": "human", "learning_case_v11": stagedCaseV11()})
	m.DB.MustExec(`INSERT INTO ai_feedback(id,feedback_type,human_answer,feedback_data,agent_id,outgoing_message_uuid,mail_event_message_uuid) VALUES($1,'response_feedback','Private source stays unchanged',$2,'101','bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb','aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa')`, fid, string(payload))
	var ids [2]string
	var wg sync.WaitGroup
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if e := m.DB.Get(&ids[i], `SELECT learning_case_stage_v11($1)`, fid); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	id := ids[0]
	if id == "" || id != ids[1] {
		t.Fatal("duplicate or missing case", ids)
	}
	var n int
	count := func(want int) {
		t.Helper()
		if e := m.DB.Get(&n, `SELECT count(*) FROM rag_case_example_view WHERE id=$1`, id); e != nil || n != want {
			t.Fatal("retrieval", n, want, e)
		}
	}
	count(0)
	item, e := m.Get(context.Background(), id)
	if e != nil || item.ReviewReady || item.Decision != "staged" {
		t.Fatal("read-only case", item, e)
	}
	page, e := m.List(context.Background(), Filter{Tab: "history", Type: "case_example", Page: 1})
	if e != nil || len(page.Items) != 1 {
		t.Fatal("history", page, e)
	}
	page, e = m.List(context.Background(), Filter{Tab: "pending", Page: 1})
	if e != nil || len(page.Items) != 0 {
		t.Fatal("case in fact queue", page, e)
	}
	if _, e = m.Decide(context.Background(), id, "reviewer", Decision{Action: "learn"}); !errors.Is(e, ErrConflict) {
		t.Fatal("fact decision accepted", e)
	}
	if _, e = m.Mutate(context.Background(), id, "reviewer", Mutation{Action: "approve", Reason: "Explicit quality decision"}); !errors.Is(e, ErrConflict) {
		t.Fatal("legacy mutation accepted", e)
	}
	mustFail(`INSERT INTO learning_reviews(item_id,original) VALUES($1,'{}')`, id)
	mustFail(`INSERT INTO rag_items SELECT (jsonb_populate_record(NULL::rag_items,to_jsonb(r)||jsonb_build_object('id',gen_random_uuid()))).* FROM rag_items r WHERE id=$1`, id)
	mustFail(`UPDATE rag_items SET source_type='employee_fact' WHERE id=$1`, id)
	mustFail(`UPDATE rag_items SET customer_question='A different historical question' WHERE id=$1`, id)
	mustFail(`DELETE FROM rag_items WHERE id=$1`, id)
	mustFail(`UPDATE rag_items SET active=true,approved=true,quality_status='approved' WHERE id=$1`, id)
	vector := make([]float64, 1536)
	vector[0] = 1
	v := vectorLiteral(vector)
	mustFail(`SELECT learning_case_publish_v11($1,0,$2::vector)`, id, v)
	mustFail(`SELECT learning_case_review_v11($1,0,'approved','reviewer',false,true,true,'[]')`, id)
	m.DB.MustExec(`SELECT learning_case_review_v11($1,0,'approved','reviewer',true,true,true,'["reklamation"]')`, id)
	count(0)
	mustFail(`SELECT learning_case_publish_v11($1,0,$2::vector)`, id, v)
	mustFail(`SELECT learning_case_publish_v11($1,1,'[1,2]'::vector)`, id)
	mustFail(`SELECT learning_case_publish_v11($1,1,$2::vector)`, id, vectorLiteral(make([]float64, 1536)))
	published := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, e := m.DB.Exec(`SELECT learning_case_publish_v11($1,1,$2::vector)`, id, v); published <- e }()
	}
	success := 0
	for i := 0; i < 2; i++ {
		if <-published == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("concurrent publication", success)
	}
	count(1)
	mustFail(`SELECT learning_case_publish_v11($1,1,$2::vector)`, id, v)
	var meta string
	m.DB.Get(&meta, `SELECT metadata::text FROM rag_case_example_view WHERE id=$1`, id)
	var metadata map[string]any
	if json.Unmarshal([]byte(meta), &metadata) != nil || metadata["factual_authority"] != false || metadata["source_type"] != "case_example" {
		t.Fatal("metadata", meta)
	}
	for _, view := range []string{"rag_verified_fact_view", "rag_final_style_view", "rag_category_learning_view"} {
		m.DB.Get(&n, `SELECT count(*) FROM `+view+` WHERE id=$1`, id)
		if n != 0 {
			t.Fatal("case leaked into", view)
		}
	}
	m.DB.Get(&n, `SELECT count(*) FROM learning_reviews WHERE item_id=$1`, id)
	if n != 0 {
		t.Fatal("fact review created")
	}
	// Ingest retry preserves the published record, source and audit.
	var retry string
	m.DB.Get(&retry, `SELECT learning_case_stage_v11($1)`, fid)
	if retry != id {
		t.Fatal("retry id")
	}
	count(1)
	m.DB.Get(&n, `SELECT count(*) FROM learning_case_audit_v11 WHERE item_id=$1`, id)
	if n != 3 {
		t.Fatal("audit", n)
	}
	mustFail(`UPDATE learning_case_audit_v11 SET actor='rewritten' WHERE item_id=$1`, id)
	mustFail(`DELETE FROM learning_case_audit_v11 WHERE item_id=$1`, id)
	// Source blocking is immediately effective, even before a publisher/reviewer runs.
	m.DB.MustExec(`UPDATE ai_feedback SET feedback_data=jsonb_set(feedback_data,'{learning_case_v11}','{"version":11,"status":"blocked","reason_code":"privacy"}') WHERE id=$1`, fid)
	count(0)
	mustFail(`UPDATE ai_feedback SET feedback_data=$2 WHERE id=$1`, fid, string(payload))
	m.DB.MustExec(`SELECT learning_case_review_v11($1,2,'blocked','reviewer')`, id)
	count(0)
	mustFail(`SELECT learning_case_review_v11($1,3,'approved','reviewer',true,true,true,'[]')`, id)
	var source string
	m.DB.Get(&source, `SELECT human_answer FROM ai_feedback WHERE id=$1`, fid)
	if source != "Private source stays unchanged" {
		t.Fatal("feedback altered")
	}
	// The existing automatic publisher never embeds a case.
	m.Embed = func(context.Context, string) ([]float64, error) {
		t.Fatal("case reached existing publisher")
		return nil, ErrInput
	}
	if e := m.PublishOne(context.Background()); e != nil {
		t.Fatal(e)
	}
	// Ordinary pipeline role cannot forge approval or use trusted functions by default.
	m.DB.MustExec(`DO $$ BEGIN IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='case_v11_untrusted') THEN CREATE ROLE case_v11_untrusted NOLOGIN; END IF; END $$;
 GRANT USAGE ON SCHEMA public TO case_v11_untrusted;
 GRANT SELECT,INSERT,UPDATE,DELETE ON rag_items TO case_v11_untrusted;`)
	conn, e := m.DB.Conn(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	if _, e = conn.ExecContext(context.Background(), `SET ROLE case_v11_untrusted`); e != nil {
		t.Fatal(e)
	}
	defer conn.ExecContext(context.Background(), `RESET ROLE`)
	for _, q := range []string{`SELECT learning_case_stage_v11('` + fid + `')`, `SELECT learning_case_review_v11('` + id + `',3,'approved','forged',true,true,true,'[]')`, `UPDATE rag_items SET learning_data=learning_data||'{"privacy_checked":true}' WHERE id='` + id + `'`} {
		if _, e = conn.ExecContext(context.Background(), q); e == nil {
			t.Fatal("untrusted role bypass", q)
		}
	}
	var allowed bool
	if e := m.DB.Get(&allowed, `SELECT has_function_privilege('case_v11_untrusted','public.learning_case_review_v11(uuid,bigint,text,text,boolean,boolean,boolean,jsonb)','EXECUTE')`); e != nil || allowed {
		t.Fatal("review function publicly executable", allowed, e)
	}
	freshFeedback := uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,feedback_type,feedback_data,human_answer,agent_id,outgoing_message_uuid,mail_event_message_uuid) VALUES($1,'response_feedback',$2,'Actual human solution','101','bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb','aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa')`, freshFeedback, string(payload))
	m.DB.MustExec(`GRANT EXECUTE ON FUNCTION learning_case_stage_v11(uuid) TO case_v11_untrusted`)
	var stagedID string
	if e := conn.QueryRowContext(context.Background(), `SELECT learning_case_stage_v11($1)`, freshFeedback).Scan(&stagedID); e != nil || stagedID == "" {
		t.Fatal("explicit ingestion grant", e)
	}
	// Migration is repeatable after real lifecycle data, too.
	if _, e = m.DB.Exec(CaseV11Schema); e != nil {
		t.Fatal("repeat after data", e)
	}
}

func TestPostgresCaseV11Envelope(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	for _, status := range []string{"blocked", "insufficient"} {
		f := uuid.NewString()
		c := map[string]any{"version": 11, "status": status, "reason_code": "privacy"}
		p, _ := json.Marshal(map[string]any{"learning_case_v11": c})
		m.DB.MustExec(`INSERT INTO ai_feedback(id,feedback_data) VALUES($1,$2)`, f, string(p))
		if _, e := m.DB.Exec(`SELECT learning_case_stage_v11($1)`, f); e == nil {
			t.Fatal("blocked case materialized")
		}
		c["human_answer"] = "Private uncleaned customer data"
		p, _ = json.Marshal(map[string]any{"learning_case_v11": c})
		if _, e := m.DB.Exec(`UPDATE ai_feedback SET feedback_data=$2 WHERE id=$1`, f, string(p)); e == nil {
			t.Fatal("unsafe blocked envelope")
		}
	}
	for _, key := range []string{"version", "status", "customer_question", "human_answer", "feedback_reason", "feedback_type", "original_categories", "corrected_categories", "has_human_resolution", "has_explicit_feedback", "requires_quality_review"} {
		c := stagedCaseV11()
		delete(c, key)
		p, _ := json.Marshal(map[string]any{"learning_case_v11": c})
		if _, e := m.DB.Exec(`INSERT INTO ai_feedback(id,feedback_data) VALUES($1,$2)`, uuid.NewString(), string(p)); e == nil {
			t.Fatal("missing field", key)
		}
	}
	for _, key := range []string{"feedback_type", "has_human_resolution", "customer_question", "original_categories"} {
		c := stagedCaseV11()
		c[key] = nil
		p, _ := json.Marshal(map[string]any{"learning_case_v11": c})
		if _, e := m.DB.Exec(`INSERT INTO ai_feedback(id,feedback_data) VALUES($1,$2)`, uuid.NewString(), string(p)); e == nil {
			t.Fatal("null field accepted", key)
		}
	}
	// Unrelated legacy feedback remains legal and is never backfilled.
	m.DB.MustExec(`INSERT INTO ai_feedback(id,feedback_data) VALUES($1,'{"legacy":"untouched"}')`, uuid.NewString())
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_items WHERE source_type='case_example'`)
	if n != 0 {
		t.Fatal("automatic backfill")
	}
}

func TestCaseV11CannotUseFactValidator(t *testing.T) {
	c := validContent()
	c.Type = "case_example"
	if Validate(c, allChecks(), true) == nil || Prepared(Item{Content: c}) {
		t.Fatal("case accepted as fact")
	}
	if strings.Contains(CaseV11Schema, "CREATE OR REPLACE VIEW public.rag_verified_fact_view") {
		t.Fatal("fact view overwritten")
	}
}

func caseFeedbackV11(t *testing.T, m *Manager, c map[string]any) string {
	t.Helper()
	fid := uuid.NewString()
	payload, _ := json.Marshal(map[string]any{"response_source": "human", "learning_case_v11": c})
	_, err := m.DB.Exec(`INSERT INTO ai_feedback(id,feedback_type,human_answer,feedback_data,agent_id,outgoing_message_uuid,mail_event_message_uuid) VALUES($1,$2,'Tatsächlich gesendete Mitarbeiterlösung',$3,'101',$4,$5)`, fid, c["feedback_type"], string(payload), uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	return fid
}

func TestPostgresCaseV11HumanReview(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	// Upgrade an already installed old nonempty-reason CHECK, without data updates.
	m.DB.MustExec(`ALTER TABLE rag_items DROP CONSTRAINT rag_case_contract_v11; ALTER TABLE rag_items ADD CONSTRAINT rag_case_contract_v11 CHECK (source_type<>'case_example' OR length(btrim(feedback_reason))>0)`)
	m.DB.MustExec(CaseV11Schema)
	checks := CaseChecks{Privacy: true, Quality: true, Generalization: true}
	for _, withReason := range []bool{true, false} {
		c := stagedCaseV11()
		if !withReason {
			c["feedback_reason"] = ""
			c["has_explicit_feedback"] = false
			c["feedback_type"] = "manual_resolution"
		}
		fid := caseFeedbackV11(t, m, c)
		var id string
		if err := m.DB.Get(&id, `SELECT learning_case_stage_v11($1)`, fid); err != nil {
			t.Fatal(err)
		}
		page, err := m.List(context.Background(), Filter{Tab: "cases", Page: 1})
		if err != nil || len(page.Items) == 0 {
			t.Fatal("case list", err)
		}
		// Every confirmation is independently required, and starts false.
		for _, missing := range []CaseChecks{{}, {Privacy: true, Quality: true}, {Privacy: true, Generalization: true}, {Quality: true, Generalization: true}} {
			if _, err := m.ReviewCase(context.Background(), id, "42", CaseDecision{Action: "approve", Checks: missing}); !errors.Is(err, ErrInput) {
				t.Fatal("missing explicit check", missing, err)
			}
		}
		authorized := *m
		authorized.View = func(context.Context, *Item) error { return ErrForbidden }
		if _, err := authorized.ReviewCase(context.Background(), id, "42", CaseDecision{Action: "approve", Checks: checks}); !errors.Is(err, ErrForbidden) {
			t.Fatal("source authorization", err)
		}
		revisions := make(chan error, 2)
		for i := 0; i < 2; i++ {
			go func() {
				_, err := m.ReviewCase(context.Background(), id, "42", CaseDecision{Action: "approve", Checks: checks})
				revisions <- err
			}()
		}
		success, conflict := 0, 0
		for i := 0; i < 2; i++ {
			err := <-revisions
			if err == nil {
				success++
			} else if errors.Is(err, ErrConflict) {
				conflict++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflict != 1 {
			t.Fatal("parallel reviews", success, conflict)
		}
		var n int
		m.DB.Get(&n, `SELECT count(*) FROM learning_case_audit_v11 WHERE item_id=$1`, id)
		if n != 2 {
			t.Fatal("duplicate audit", n)
		}
		var actor string
		m.DB.Get(&actor, `SELECT learning_data->>'reviewer' FROM rag_items WHERE id=$1`, id)
		if actor != "42" {
			t.Fatal("actor", actor)
		}
		var active bool
		m.DB.Get(&active, `SELECT active OR approved OR embedding IS NOT NULL FROM rag_items WHERE id=$1`, id)
		if active {
			t.Fatal("review published automatically")
		}
		m.DB.Get(&n, `SELECT count(*) FROM learning_reviews WHERE item_id=$1`, id)
		if n != 0 {
			t.Fatal("case entered fact review")
		}
	}
}

func TestPostgresCaseV11SourceAndPrivacy(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	for _, field := range []string{"human_answer", "has_human_resolution"} {
		c := stagedCaseV11()
		if field == "human_answer" {
			c[field] = ""
		} else {
			c[field] = false
			c["feedback_type"] = "category_correction"
		}
		raw, _ := json.Marshal(map[string]any{"learning_case_v11": c})
		if _, err := m.DB.Exec(`INSERT INTO ai_feedback(id,feedback_data) VALUES($1,$2)`, uuid.NewString(), string(raw)); err == nil {
			t.Fatal("missing employee resolution accepted", field)
		}
	}
	for _, field := range []string{"human_answer", "agent_id", "outgoing_message_uuid", "mail_event_message_uuid", "response_source"} {
		c := stagedCaseV11()
		raw, _ := json.Marshal(map[string]any{"response_source": "human", "learning_case_v11_draft": c})
		fid := uuid.NewString()
		m.DB.MustExec(`INSERT INTO ai_feedback(id,feedback_type,human_answer,agent_id,outgoing_message_uuid,mail_event_message_uuid,feedback_data) VALUES($1,'response_feedback','Human source','101',$2,$3,$4)`, fid, uuid.NewString(), uuid.NewString(), string(raw))
		if field == "response_source" {
			m.DB.MustExec(`UPDATE ai_feedback SET feedback_data=jsonb_set(feedback_data,'{response_source}','"ai_adapted"') WHERE id=$1`, fid)
		} else {
			m.DB.MustExec(`UPDATE ai_feedback SET `+field+`='' WHERE id=$1`, fid)
		}
		canon, _ := json.Marshal(c)
		m.DB.MustExec(`UPDATE ai_feedback SET feedback_data=jsonb_set(feedback_data,'{learning_case_v11}',$2) WHERE id=$1`, fid, string(canon))
		if _, err := m.DB.Exec(`SELECT learning_case_stage_v11($1)`, fid); err == nil {
			t.Fatal("unproven human source", field)
		}
	}
	// Drafts can evolve; only the subsequent canonical value is staged and frozen.
	c := stagedCaseV11()
	fid := uuid.NewString()
	draft, _ := json.Marshal(map[string]any{"response_source": "human", "learning_case_v11_draft": map[string]string{"customer_question": "Vorläufiger Entwurf"}})
	m.DB.MustExec(`INSERT INTO ai_feedback(id,feedback_type,human_answer,agent_id,outgoing_message_uuid,mail_event_message_uuid,feedback_data) VALUES($1,'response_feedback','Human source','101',$2,$3,$4)`, fid, uuid.NewString(), uuid.NewString(), string(draft))
	if _, err := m.DB.Exec(`SELECT learning_case_stage_v11($1)`, fid); err == nil {
		t.Fatal("draft staged")
	}
	canon, _ := json.Marshal(c)
	m.DB.MustExec(`UPDATE ai_feedback SET feedback_data=jsonb_set(feedback_data,'{learning_case_v11}',$2) WHERE id=$1`, fid, string(canon))
	if _, err := m.DB.Exec(`UPDATE ai_feedback SET feedback_data=jsonb_set(feedback_data,'{learning_case_v11,human_answer}','"Rewritten canonical solution"') WHERE id=$1`, fid); err == nil {
		t.Fatal("canonical changed before materialization")
	}
	var id string
	m.DB.Get(&id, `SELECT learning_case_stage_v11($1)`, fid)
	// Privacy is an explicit human decision; placeholders do not trigger a second check.
	prepared := stagedCaseV11()
	prepared["feedback_reason"] = "Bestellnummer: [Bestellnummer]"
	preparedFeedback := caseFeedbackV11(t, m, prepared)
	var preparedID string
	m.DB.Get(&preparedID, `SELECT learning_case_stage_v11($1)`, preparedFeedback)
	checks := CaseChecks{Privacy: true, Generalization: true, Quality: true}
	if _, err := m.ReviewCase(context.Background(), preparedID, "42", CaseDecision{Action: "approve", Checks: CaseChecks{Generalization: true, Quality: true}}); !errors.Is(err, ErrInput) {
		t.Fatal("missing human privacy confirmation accepted", err)
	}
	if _, err := m.ReviewCase(context.Background(), preparedID, "42", CaseDecision{Action: "approve", Checks: checks}); err != nil {
		t.Fatal("human privacy approval", err)
	}
	if _, err := m.ReviewCase(context.Background(), preparedID, "42", CaseDecision{Action: "block", Revision: 1}); err != nil {
		t.Fatal("privacy block", err)
	}
	if _, err := m.ReviewCase(context.Background(), id, "42", CaseDecision{Action: "reject"}); err != nil {
		t.Fatal("rejection", err)
	}
	if _, err := m.ReviewCase(context.Background(), id, "42", CaseDecision{Action: "approve", Revision: 1, Checks: checks}); !errors.Is(err, ErrConflict) {
		t.Fatal("rejected case approved", err)
	}
}
