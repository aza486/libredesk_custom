package learning

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPostgresCasePublisherViewSchema(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	legacy, err := os.ReadFile("testdata/legacy_views.sql")
	if err != nil {
		t.Fatal(err)
	}
	m.DB.MustExec(string(legacy))
	views, err := os.ReadFile("artifacts/002_style_view.sql")
	if err != nil {
		t.Fatal(err)
	}
	m.DB.MustExec(string(views))
	definitions := func() string {
		var s string
		if err := m.DB.Get(&s, `SELECT string_agg(viewname||definition,'|' ORDER BY viewname) FROM pg_views WHERE schemaname='public' AND viewname IN ('rag_case_example_view','rag_verified_fact_view','rag_final_style_view','rag_category_learning_view')`); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := definitions()
	m.DB.MustExec(CasePublisherV11Schema)
	// Rerunning must actually remove an accidentally assigned PUBLIC grant, too.
	m.DB.MustExec(`GRANT SELECT ON public.learning_case_publisher_candidates_v11 TO PUBLIC`)
	m.DB.MustExec(CasePublisherV11Schema)
	if before != definitions() {
		t.Fatal("existing retrieval view changed")
	}
	var columns []string
	if err := m.DB.Select(&columns, `SELECT column_name||':'||data_type FROM information_schema.columns WHERE table_schema='public' AND table_name='learning_case_publisher_candidates_v11' ORDER BY ordinal_position`); err != nil {
		t.Fatal(err)
	}
	want := []string{"id:uuid", "revision:bigint", "topic:text", "customer_question:text", "human_answer:text", "feedback_reason:text", "retrieval_text:text", "updated_at:timestamp without time zone"}
	if !reflect.DeepEqual(columns, want) {
		t.Fatal("unexpected publication projection", columns)
	}
	var protected, publicGrant bool
	if err := m.DB.Get(&protected, `SELECT COALESCE(reloptions @> ARRAY['security_barrier=true'],false) FROM pg_class WHERE oid='public.learning_case_publisher_candidates_v11'::regclass`); err != nil || !protected {
		t.Fatal("missing security barrier", err)
	}
	if err := m.DB.Get(&publicGrant, `SELECT EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE c.oid='public.learning_case_publisher_candidates_v11'::regclass AND a.grantee=0)`); err != nil || publicGrant {
		t.Fatal("view is publicly accessible", err)
	}
	id, _ := approvedPublisherCase(t, m)
	var revision int64
	if err := m.DB.Get(&revision, `SELECT revision FROM public.learning_case_publisher_candidates_v11 WHERE id=$1`, id); err != nil || revision != 1 {
		t.Fatal("approved candidate", revision, err)
	}
	// Repeated installation with real review data preserves candidate and revision.
	m.DB.MustExec(CasePublisherV11Schema)
	if err := m.DB.Get(&revision, `SELECT revision FROM public.learning_case_publisher_candidates_v11 WHERE id=$1`, id); err != nil || revision != 1 {
		t.Fatal("migration changed candidate", revision, err)
	}
}

func TestPostgresCasePublisherViewRejectsInvalidCandidates(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	m.DB.MustExec(CasePublisherV11Schema)
	id, fid := approvedPublisherCase(t, m)
	// Simulate historical corruption under the test owner, without weakening SQL005
	// in the artifact or installed schema: all bypasses are local to a rolled-back tx.
	tx := m.DB.MustBegin()
	defer tx.Rollback()
	tx.MustExec(`ALTER TABLE public.rag_items DROP CONSTRAINT rag_case_contract_v11`)
	visible := func(t *testing.T, want int) {
		t.Helper()
		var n int
		if err := tx.Get(&n, `SELECT count(*) FROM public.learning_case_publisher_candidates_v11 WHERE id=$1`, id); err != nil || n != want {
			t.Fatal("candidate visibility", n, want, err)
		}
	}
	visible(t, 1)
	mutations := map[string]any{
		"case_version": 10, "pipeline_version": 10, "case_status": "staged", "publish_state": "published",
		"privacy_checked": false, "quality_checked": false, "generalization_checked": false,
		"requires_quality_review": false, "factual_authority": true, "factual_use_approved": true,
		"target_agent": "company_knowledge", "example_role": "style_only", "origin_digest": "outdated_source_digest",
	}
	for key, value := range mutations {
		t.Run(key, func(t *testing.T) {
			tx.MustExec(`SAVEPOINT candidate_mutation`)
			payload, _ := json.Marshal(map[string]any{key: value})
			tx.MustExec(`UPDATE public.rag_items SET learning_data=learning_data||$2::jsonb WHERE id=$1`, id, string(payload))
			visible(t, 0)
			tx.MustExec(`ROLLBACK TO SAVEPOINT candidate_mutation`)
		})
	}
	// Missing flags/NULLs must not turn a filter into implicit approval.
	for _, key := range []string{"case_version", "pipeline_version", "case_status", "publish_state", "privacy_checked", "quality_checked", "generalization_checked", "requires_quality_review", "factual_authority", "factual_use_approved", "target_agent", "example_role", "origin_digest"} {
		tx.MustExec(`SAVEPOINT missing_flag`)
		tx.MustExec(`UPDATE public.rag_items SET learning_data=learning_data-$2::text WHERE id=$1`, id, key)
		visible(t, 0)
		tx.MustExec(`ROLLBACK TO SAVEPOINT missing_flag`)
	}
	for _, quality := range []string{"pending", "rejected", "blocked"} {
		tx.MustExec(`SAVEPOINT unreviewed_quality`)
		tx.MustExec(`UPDATE public.rag_items SET quality_status=$2 WHERE id=$1`, id, quality)
		visible(t, 0)
		tx.MustExec(`ROLLBACK TO SAVEPOINT unreviewed_quality`)
	}
	tx.MustExec(`SAVEPOINT prior_approval`)
	tx.MustExec(`UPDATE public.rag_items SET approved=true WHERE id=$1`, id)
	visible(t, 0)
	tx.MustExec(`ROLLBACK TO SAVEPOINT prior_approval`)
	tx.MustExec(`SAVEPOINT prior_embedding`)
	tx.MustExec(`UPDATE public.rag_items SET embedding=$2::vector WHERE id=$1`, id, vectorLiteral(caseVector()))
	visible(t, 0)
	tx.MustExec(`ROLLBACK TO SAVEPOINT prior_embedding`)
	tx.MustExec(`SAVEPOINT source_blocked`)
	tx.MustExec(`UPDATE public.ai_feedback SET feedback_data=jsonb_set(feedback_data,'{learning_case_v11}','{"version":11,"status":"blocked","reason_code":"privacy"}') WHERE id=$1`, fid)
	visible(t, 0)
	tx.MustExec(`ROLLBACK TO SAVEPOINT source_blocked`)
	// The view exposes the current revision; a stale expected revision cannot pass
	// the publisher's second query, even when an owner changed only the revision.
	tx.MustExec(`SAVEPOINT stale_revision`)
	tx.MustExec(`UPDATE public.rag_items SET learning_data=jsonb_set(learning_data,'{case_revision}','2') WHERE id=$1`, id)
	var current bool
	if err := tx.Get(&current, `SELECT EXISTS(SELECT 1 FROM public.learning_case_publisher_candidates_v11 r WHERE r.id=$1 AND r.revision=1)`, id); err != nil || current {
		t.Fatal("stale revision accepted", err)
	}
	tx.MustExec(`ROLLBACK TO SAVEPOINT stale_revision`)
	visible(t, 1)
}

func TestPostgresCasePublisherViewMissingFailsClosed(t *testing.T) {
	m := isolated(t)
	m.DB.MustExec(CaseV11Schema)
	id, _ := approvedPublisherCase(t, m)
	p := NewCasePublisher(m.DB, func(context.Context, string) ([]float64, error) {
		t.Fatal("missing SQL006 reached embedder")
		return nil, nil
	}, true)
	requireCaseFailure(t, p.PublishOne(context.Background()), "case_database_failed")
	var active bool
	if err := m.DB.Get(&active, `SELECT active FROM public.rag_items WHERE id=$1`, id); err != nil || active {
		t.Fatal("fallback publication", err)
	}
}

func TestCasePublisherReadsOnlyProtectedView(t *testing.T) {
	source, err := os.ReadFile("case_publisher.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, base := range []string{"rag_items", "ai_feedback", "caseCandidateWhere", "r.learning_data"} {
		if strings.Contains(string(source), base) {
			t.Fatal("publisher retained a base-table read", base)
		}
	}
	if strings.Count(string(source), "FROM public.learning_case_publisher_candidates_v11 r") != 2 {
		t.Fatal("both read paths must use the protected view")
	}
}
