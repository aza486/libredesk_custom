package learning

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestPostgresV10PreparedDraftContract(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	var workflow struct {
		Nodes []struct {
			Name       string `json:"name"`
			Parameters struct {
				Query string `json:"query"`
			} `json:"parameters"`
		} `json:"nodes"`
	}
	raw, e := os.ReadFile("artifacts/AI_Learning_Processor_v10.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &workflow); e != nil {
		t.Fatal(e)
	}
	queries := map[string]string{}
	for _, n := range workflow.Nodes {
		queries[n.Name] = n.Parameters.Query
	}
	m.DB.MustExec(`ALTER TABLE rag_items ALTER COLUMN id SET DEFAULT gen_random_uuid()`)
	fid := uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,human_answer,learning_status) VALUES($1,'Das Büro ist samstags geschlossen.','processing')`, fid)
	c := validContent()
	draft := map[string]any{"feedback_id": fid, "candidates": []any{map[string]any{
		"source_type": "employee_fact", "category": c.Category, "topic": c.Topic, "customer_question": c.Question, "retrieval_text": c.Retrieval,
		"learning_data": map[string]any{"proposal_version": 2, "pipeline_version": 10, "review_required": true, "managed_by_review": true, "fact_statement": c.Rule, "evidence_quote": c.Evidence, "evidence_source": "human_answer", "limitations": c.Limits, "rationale": "Belegte allgemeine Firmeninformation unabhängig prüfen.", "review_reason": "Allgemeine Gültigkeit prüfen.", "target_agent": "company_knowledge", "verification_status": "unverified", "factual_use_approved": false, "publish_state": "unpublished"},
	}}}
	payload, _ := json.Marshal(draft)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := m.DB.Exec(queries["Save Draft RAG Items"], string(payload)); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM rag_items WHERE feedback_id=$1`, fid)
	if n != 1 {
		t.Fatal("duplicate draft", n)
	}
	var id string
	m.DB.Get(&id, `SELECT id FROM rag_items WHERE feedback_id=$1`, fid)
	var status string
	m.DB.Get(&status, `SELECT learning_status FROM ai_feedback WHERE id=$1`, fid)
	if status != "prepared" {
		t.Fatal(status)
	}
	rows, e := m.DB.Queryx(queries["Claim RAG Publication"])
	if e != nil {
		t.Fatal(e)
	}
	if rows.Next() {
		t.Fatal("fact claimed automatically")
	}
	rows.Close()
	if _, e = m.DB.Exec(`UPDATE rag_items SET topic='Unreviewed replacement' WHERE id=$1`, id); e == nil {
		t.Fatal("prepared proposal changed without revision")
	}
	item, e := m.Get(ctx, id)
	if e != nil || !item.ReviewReady || item.Active {
		t.Fatal("proposal", e, item)
	}
	if _, e = m.Decide(ctx, id, "42", Decision{Action: "learn"}); e != nil {
		t.Fatal(e)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	views, _ := os.ReadFile("artifacts/002_style_view.sql")
	m.DB.MustExec(string(views))
	m.DB.Get(&n, `SELECT count(*) FROM rag_verified_fact_view WHERE id=$1`, id)
	if n != 1 {
		t.Fatal("published fact not retrieved", n)
	}
	// Same payload after timeout/restart cannot create a second candidate or reset publication.
	if _, e = m.DB.Exec(queries["Save Draft RAG Items"], string(payload)); e != nil {
		t.Fatal(e)
	}
	m.DB.Get(&n, `SELECT count(*) FROM rag_items WHERE feedback_id=$1 AND active`, fid)
	if n != 1 {
		t.Fatal("retry changed result", n)
	}
	// A genuinely unlearnable feedback records skipped without a review row.
	skipped := uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,learning_status) VALUES($1,'processing')`, skipped)
	p, _ := json.Marshal(map[string]any{"feedback_id": skipped, "candidates": []any{}, "skip_reason": "no_complete_grounded_learning_candidate"})
	if _, e = m.DB.Exec(queries["Save Draft RAG Items"], string(p)); e != nil {
		t.Fatal(e)
	}
	m.DB.Get(&status, `SELECT learning_status FROM ai_feedback WHERE id=$1`, skipped)
	if status != "skipped" {
		t.Fatal(status)
	}
	m.DB.Get(&n, `SELECT count(*) FROM rag_items WHERE feedback_id=$1`, skipped)
	if n != 0 {
		t.Fatal("skip produced review task")
	}

	// A pipeline role needs only the new column INSERT grant, never original/audit reads.
	m.DB.MustExec(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='learning_v10_pipeline') THEN CREATE ROLE learning_v10_pipeline NOLOGIN; END IF; END $$;
 GRANT USAGE ON SCHEMA public TO learning_v10_pipeline;
 GRANT SELECT,INSERT,UPDATE ON rag_items TO learning_v10_pipeline;
 GRANT SELECT,UPDATE ON ai_feedback TO learning_v10_pipeline;
 GRANT SELECT(item_id) ON learning_reviews TO learning_v10_pipeline;`)
	grants, e := os.ReadFile("artifacts/004_v10_permissions.sql")
	if e != nil {
		t.Fatal(e)
	}
	m.DB.MustExec(strings.ReplaceAll(string(grants), `:"publisher_role"`, `"learning_v10_pipeline"`))
	restricted := uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,learning_status) VALUES($1,'processing')`, restricted)
	draft["feedback_id"] = restricted
	p, _ = json.Marshal(draft)
	m.DB.SetMaxOpenConns(1)
	m.DB.MustExec(`SET ROLE learning_v10_pipeline`)
	_, e = m.DB.Exec(queries["Save Draft RAG Items"], string(p))
	m.DB.MustExec(`RESET ROLE`)
	if e != nil {
		t.Fatal("narrow publisher INSERT permission", e)
	}
	var canReadOriginal, canUpdateReview bool
	if e = m.DB.QueryRow(`SELECT has_column_privilege('learning_v10_pipeline','learning_reviews','original','SELECT'),has_table_privilege('learning_v10_pipeline','learning_reviews','UPDATE')`).Scan(&canReadOriginal, &canUpdateReview); e != nil || canReadOriginal || canUpdateReview {
		t.Fatal("excessive pipeline rights", e)
	}

	// Existing native SQL automatically publishes a safe style after one valid staged embedding.
	styleFeedback := uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,learning_status) VALUES($1,'processing')`, styleFeedback)
	style := map[string]any{"feedback_id": styleFeedback, "candidates": []any{map[string]any{"source_type": "response_example", "category": "firmenfragen", "topic": "Freundliche Antwort", "retrieval_text": "Wie kann ich mich freundlich bedanken?", "human_answer": "Vielen Dank für Ihre Nachricht. Gerne helfen wir Ihnen weiter.", "learning_data": map[string]any{"pipeline_version": 10, "target_agent": "final_answer", "example_role": "style_only", "factual_authority": false}}}}
	p, _ = json.Marshal(style)
	if _, e = m.DB.Exec(queries["Save Draft RAG Items"], string(p)); e != nil {
		t.Fatal(e)
	}
	rows, e = m.DB.Queryx(queries["Claim RAG Publication"])
	if e != nil {
		t.Fatal(e)
	}
	var styleID string
	if !rows.Next() {
		t.Fatal("safe style not claimed")
	}
	data := map[string]any{}
	if e = rows.MapScan(data); e != nil {
		t.Fatal(e)
	}
	styleID = data["rag_item_id"].(string)
	rows.Close()
	m.DB.MustExec(`CREATE TABLE learning_embedding_stage(id uuid DEFAULT gen_random_uuid(),embedding vector(1536),metadata jsonb)`)
	vector := make([]float64, 1536)
	vector[0] = 1
	meta, _ := json.Marshal(map[string]string{"rag_item_id": styleID})
	m.DB.MustExec(`INSERT INTO learning_embedding_stage(embedding,metadata) VALUES($1::vector,$2::jsonb)`, vectorLiteral(make([]float64, 1536)), string(meta))
	if _, e = m.DB.Exec(queries["Finalize RAG Publication"], string(meta)); e != nil {
		t.Fatal(e)
	}
	m.DB.Get(&n, `SELECT count(*) FROM rag_items WHERE id=$1 AND active`, styleID)
	if n != 0 {
		t.Fatal("zero vector activated")
	}
	m.DB.MustExec(`DELETE FROM learning_embedding_stage`)
	m.DB.MustExec(`INSERT INTO learning_embedding_stage(embedding,metadata) VALUES($1::vector,$2::jsonb)`, vectorLiteral(vector), string(meta))
	if _, e = m.DB.Exec(queries["Finalize RAG Publication"], string(meta)); e != nil {
		t.Fatal(e)
	}
	if _, e = m.DB.Exec(queries["Finalize RAG Publication"], string(meta)); e != nil {
		t.Fatal("finalize retry", e)
	}
	m.DB.Get(&n, `SELECT count(*) FROM rag_items WHERE id=$1 AND active AND approved AND quality_status='approved'`, styleID)
	if n != 1 {
		t.Fatal("automatic style publication", n)
	}
	m.DB.Get(&n, `SELECT count(*) FROM learning_embedding_stage`)
	if n != 0 {
		t.Fatal("stage not consumed")
	}
}
