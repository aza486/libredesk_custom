package learning

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Schema is an explicit pg-n8n installation artifact, never applied on application startup.
//
//go:embed artifacts/001_review.sql
var Schema string

type Manager struct {
	DB            *sqlx.DB
	Embed         Embedder
	Anonymize     Anonymizer
	ReportFailure func(string, error)
	// View is request-scoped. It must authorize the source conversation before any display output.
	View func(context.Context, *Item) error
}

const projection = `SELECT jsonb_build_object('id',r.id,'record_type','candidate','learning_status',f.learning_status,'review_exists',v.item_id IS NOT NULL,'ai_suggestion',r.ai_answer,'sent_answer',f.human_answer,'content',jsonb_build_object(
 'source_type',r.source_type,'category',r.category,'topic',r.topic,'retrieval_text',r.retrieval_text,
 'customer_question',r.customer_question,'human_answer',r.human_answer,
 'rule',COALESCE(r.learning_data->>'fact_statement',r.learning_data->>'guidance',''),
 'evidence',r.learning_data->>'evidence_quote','source',r.learning_data->>'evidence_source',
 'limits',r.learning_data->>'limitations','original_categories',r.learning_data->'original_categories',
 'corrected_categories',r.learning_data->'corrected_categories'),
 'quality_status',r.quality_status,'approved',r.approved,'active',r.active,
 'publish_state',COALESCE(v.publication,r.learning_data->>'publish_state','unpublished'),
 'decision',COALESCE(v.decision,'pending'),'revision',COALESCE(v.revision,0),
 'feedback_reason',COALESCE(r.feedback_reason,f.feedback_reason),'rationale',r.learning_data->>'rationale',
 'provenance',jsonb_build_object('feedback_id',r.feedback_id,'source_id',r.source_id,'derived_from',r.learning_data->>'derived_from',
 'original_message_uuid',f.mail_event_message_uuid,'outgoing_message_uuid',f.outgoing_message_uuid,'proposal_version',r.learning_data->'proposal_version','review_required',r.learning_data->'review_required','verification_status',r.learning_data->>'verification_status','factual_use_approved',r.learning_data->'factual_use_approved','publication_attempts',v.attempts),
 'created_at',r.created_at,'block_reason',COALESCE(v.last_error,r.learning_data->>'publish_block_reason',r.learning_data->>'review_reason'))
 FROM public.rag_items r LEFT JOIN learning_reviews v ON v.item_id=r.id LEFT JOIN public.ai_feedback f ON f.id=r.feedback_id `

func (m *Manager) safe(ctx context.Context, s string) (string, error) {
	if strings.TrimSpace(s) == "" {
		return "", nil
	}
	if m.Anonymize == nil {
		return "", &DependencyError{Service: "anonymizer", Kind: "not_configured"}
	}
	clean, err := m.Anonymize(ctx, s)
	if err != nil {
		var dependency *DependencyError
		if errors.As(err, &dependency) {
			return "", dependency
		}
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return "", &DependencyError{Service: "anonymizer", Kind: "timeout"}
		}
		return "", &DependencyError{Service: "anonymizer", Kind: "request_failed"}
	}
	if strings.TrimSpace(clean) == "" || !text(clean, 10000) {
		return "", &DependencyError{Service: "anonymizer", Kind: "invalid_response"}
	}
	if sensitive.MatchString(clean) {
		return "", &DependencyError{Service: "anonymizer", Kind: "privacy_rejected"}
	}
	return clean, nil
}

func normalizeDisplay(item *Item) {

	if len(item.Provenance) > 0 {
		var p map[string]any
		if json.Unmarshal(item.Provenance, &p) == nil {
			for _, key := range []string{"feedback_id", "source_id", "derived_from"} {
				value, _ := p[key].(string)
				if _, err := uuid.Parse(value); err != nil {
					delete(p, key)
				}
			}
			if value, _ := p["verification_status"].(string); !contains([]string{"unverified", "employee_attested", "human_verified", "not_applicable"}, value) {
				delete(p, "verification_status")
			}
			item.Provenance, _ = json.Marshal(p)
		}
	}

	// Categories are labels, not a channel for arbitrary JSON or customer data.
	if !displayCategoryCode.MatchString(item.Content.Category) {
		item.Content.Category = ""
	}
	item.Content.OriginalCategories = filterCategories(item.Content.OriginalCategories)
	item.Content.CorrectedCategories = filterCategories(item.Content.CorrectedCategories)
}

var displayCategoryCode = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

func filterCategories(in []string) []string {
	out := []string{}
	for _, s := range in {
		if displayCategoryCode.MatchString(s) && !contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
func decodeItem(raw []byte) (Item, error) {
	var item Item
	err := json.Unmarshal(raw, &item)
	item.ReviewReady = Prepared(item)
	return item, err
}
func (m *Manager) List(ctx context.Context, f Filter) (Page, error) {
	out := Page{Items: []Item{}, Page: f.Page}
	if f.Status != "" && !contains([]string{"pending", "approved", "rejected", "disabled", "unpublished", "publishing", "published", "failed", "blocked", "skipped", "prepared", "processing", "new"}, f.Status) {
		return out, ErrInput
	}
	if f.Page < 1 || f.Page > 10000 || (!contains(Types, f.Type) && f.Type != "manual" && f.Type != "") {
		return out, ErrInput
	}
	if f.Tab == "skipped" && m.View != nil {
		return out, ErrInput
	}
	if f.Tab == "skipped" {
		rows, err := m.DB.QueryContext(ctx, `SELECT jsonb_build_object('id',f.id,'record_type','feedback','learning_status',f.learning_status,'candidate_count',(SELECT count(*) FROM public.rag_items r WHERE r.feedback_id=f.id),'content',jsonb_build_object('human_answer',f.human_answer,'customer_question',e.clean_text,'original_categories',to_jsonb(e.categories)),'feedback_reason',f.feedback_reason,'block_reason',f.feedback_data->>'learning_skip_reason','quality_status',f.quality_status,'created_at',f.created_at,'decision','skipped','revision',0) FROM public.ai_feedback f LEFT JOIN public.mail_events e ON e.message_uuid=f.mail_event_message_uuid AND (SELECT count(e2.message_uuid) FROM public.mail_events e2 WHERE e2.message_uuid=f.mail_event_message_uuid)=1 WHERE f.learning_status='skipped' AND ($1='' OR f.quality_status=$1 OR $1='skipped') ORDER BY f.created_at DESC,f.id LIMIT 26 OFFSET $2`, f.Status, (f.Page-1)*25)
		if err != nil {
			return out, err
		}
		defer rows.Close()
		for rows.Next() {
			var raw []byte
			if err = rows.Scan(&raw); err != nil {
				return out, err
			}
			item, e := decodeItem(raw)
			if e != nil {
				return out, e
			}
			normalizeDisplay(&item)
			out.Items = append(out.Items, item)
		}
		out.HasMore = len(out.Items) > 25
		if out.HasMore {
			out.Items = out.Items[:25]
		}
		return out, rows.Err()
	}
	if f.Tab != "pending" && f.Tab != "history" {
		return out, ErrInput
	}
	rows, err := m.DB.QueryContext(ctx, projection+` WHERE ($1='' OR r.source_type=$1) AND ($2='' OR r.quality_status=$2 OR v.decision=$2 OR v.publication=$2 OR r.learning_data->>'publish_state'=$2 OR f.learning_status=$2) AND CASE WHEN $3='pending' THEN (COALESCE(v.decision,'pending')='pending' AND NOT r.active AND r.learning_data->>'proposal_version'='2' AND r.learning_data->>'review_required'='true' AND r.source_type='employee_fact') ELSE NOT (COALESCE(v.decision,'pending')='pending' AND NOT r.active AND COALESCE(r.learning_data->>'proposal_version','')='2' AND COALESCE(r.learning_data->>'review_required','')='true' AND r.source_type='employee_fact') END ORDER BY r.created_at DESC,r.id LIMIT 26 OFFSET $4`, f.Type, f.Status, f.Tab, (f.Page-1)*25)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		if count > 25 {
			out.HasMore = true
			break
		}
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			return out, err
		}
		item, e := decodeItem(raw)
		if e != nil {
			return out, e
		}
		if f.Tab == "pending" && !item.ReviewReady {
			continue
		}
		if m.View != nil {
			if e := m.View(ctx, &item); e != nil {
				continue
			}
		}
		normalizeDisplay(&item)
		out.Items = append(out.Items, item)
	}
	return out, rows.Err()
}
func (m *Manager) Get(ctx context.Context, id string) (Item, error) {
	var raw []byte
	if _, err := uuid.Parse(id); err != nil {
		return Item{}, ErrInput
	}
	if err := m.DB.QueryRowContext(ctx, projection+` WHERE r.id=$1::uuid`, id).Scan(&raw); err != nil {
		return Item{}, err
	}
	item, err := decodeItem(raw)
	if err != nil {
		return item, err
	}
	if m.View != nil {
		if e := m.View(ctx, &item); e != nil {
			return Item{}, e
		}
	}
	normalizeDisplay(&item)
	// Audit/original are plain-text display data under the same request authorization.
	var original []byte
	err = m.DB.QueryRowContext(ctx, `SELECT original FROM learning_reviews WHERE item_id=$1::uuid`, id).Scan(&original)
	if err != nil && err != sql.ErrNoRows {
		return item, err
	}
	if len(original) > 0 {
		c, e := loadLockedContent(original)
		if e != nil {
			return item, e
		}
		old := Item{Content: c}
		normalizeDisplay(&old)
		item.Original, _ = json.Marshal(old.Content)
	}
	rows, err := m.DB.QueryContext(ctx, `SELECT jsonb_build_object('revision',revision,'actor',actor,'action',action,'reason',reason,'created_at',created_at,'snapshot',snapshot) FROM learning_review_audit WHERE item_id=$1::uuid ORDER BY revision DESC LIMIT 100`, id)
	if err != nil {
		return item, err
	}
	defer rows.Close()
	item.History = []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err = rows.Scan(&b); err != nil {
			return item, err
		}
		var entry map[string]any
		if err = json.Unmarshal(b, &entry); err != nil {
			return item, err
		}
		if snap, ok := entry["snapshot"]; ok {
			raw, _ := json.Marshal(snap)
			c, e := loadLockedContent(raw)
			if e != nil {
				return item, e
			}
			past := Item{Content: c}
			normalizeDisplay(&past)
			entry["snapshot"] = past.Content
		}

		safe, _ := json.Marshal(entry)
		item.History = append(item.History, safe)
	}
	return item, rows.Err()
}
func validateMutation(req Mutation) error {
	if !contains([]string{"approve", "reject", "deactivate", "retry"}, req.Action) || !text(req.Reason, 1000) || utf8.RuneCountInString(strings.TrimSpace(req.Reason)) < 5 || req.Revision < 0 || sensitive.MatchString(req.Reason) {
		return ErrInput
	}
	return nil
}
func (m *Manager) Mutate(ctx context.Context, id, actor string, req Mutation) (string, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrInput
	}
	if actor == "" {
		return "", ErrInput
	}
	if err := validateMutation(req); err != nil {
		return "", err
	}
	tx, err := m.DB.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT set_config('libredesk.review_writer','true',true)`); err != nil {
		return "", err
	}
	var raw []byte
	if err = tx.QueryRowContext(ctx, `SELECT to_jsonb(r)-'embedding' FROM rag_items r WHERE id=$1::uuid FOR UPDATE`, id).Scan(&raw); err != nil {
		return "", err
	}
	var row struct {
		Type   string `json:"source_type"`
		Active bool   `json:"active"`
	}
	if err = json.Unmarshal(raw, &row); err != nil {
		return "", err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_reviews(item_id,revision,original) VALUES($1::uuid,0,$2::jsonb) ON CONFLICT DO NOTHING`, id, string(raw))
	if err != nil {
		return "", err
	}
	var rev int64
	var decision, publication string
	if err = tx.QueryRowContext(ctx, `SELECT revision,decision,publication FROM learning_reviews WHERE item_id=$1::uuid FOR UPDATE`, id).Scan(&rev, &decision, &publication); err != nil {
		return "", err
	}
	if rev != req.Revision || publication == "publishing" {
		return "", ErrConflict
	}
	if req.decisionV2 {
		c, e := loadLockedContent(raw)
		if e != nil {
			return "", e
		}
		var meta struct {
			Data struct {
				Version   int    `json:"proposal_version"`
				Required  bool   `json:"review_required"`
				Rationale string `json:"rationale"`
			} `json:"learning_data"`
		}
		if json.Unmarshal(raw, &meta) != nil || meta.Data.Version != 2 || !meta.Data.Required || meta.Data.Rationale == "" || c.Type != "employee_fact" || decision != "pending" {
			return "", ErrConflict
		}
	}
	switch req.Action {
	case "approve":
		if row.Active || decision == "rejected" {
			return "", ErrConflict
		}
		// Approval validates the persisted candidate, not an unpersisted browser edit.
		item, e := loadLockedContent(raw)
		if e != nil {
			return "", e
		}
		if req.decisionV2 {
			// Recheck the actual feedback evidence under the candidate lock; model metadata is not proof.
			var source string
			if item.Source != "human_answer" && item.Source != "feedback_reason" {
				return "", ErrInput
			}
			e = tx.QueryRowContext(ctx, `SELECT CASE WHEN $2='human_answer' THEN f.human_answer ELSE f.feedback_reason END FROM public.ai_feedback f JOIN public.rag_items r ON r.feedback_id=f.id WHERE r.id=$1::uuid`, id, item.Source).Scan(&source)
			if e != nil {
				return "", ErrInput
			}
			if item.Source == "human_answer" {
				source = stringutil.HTML2Text(source)
			}
			clean, e := m.safe(ctx, source)
			if e != nil {
				return "", e
			}
			if !strings.Contains(strings.Join(strings.Fields(clean), " "), strings.Join(strings.Fields(item.Evidence), " ")) {
				return "", ErrInput
			}
		}
		if err = Validate(item, req.Checks, true); err != nil {
			return "", err
		}
		if err = m.privacyChecked(ctx, item); err != nil {
			return "", err
		}
		if err = writeContent(ctx, tx, id, item); err != nil {
			return "", err
		}
		decision = "approved"
		publication = "unpublished"
	case "reject":
		if row.Active || publication == "published" {
			return "", ErrConflict
		}
		decision = "rejected"
		publication = "unpublished"
	case "deactivate":
		if !row.Active {
			return "", ErrConflict
		}
		decision = "disabled"
		publication = "unpublished"
	case "retry":
		if decision != "approved" || publication != "failed" {
			return "", ErrConflict
		}
		publication = "unpublished"
	}
	checks, _ := json.Marshal(req.Checks)
	_, err = tx.ExecContext(ctx, `UPDATE learning_reviews SET revision=revision+1,decision=$2,publication=$3,reviewer=$4,note=$5,checks=CASE WHEN $7='retry' THEN checks ELSE $6::jsonb END,retry_at=now(),lease=NULL,last_error=NULL,updated_at=now() WHERE item_id=$1::uuid`, id, decision, publication, actor, req.Reason, string(checks), req.Action)
	if err != nil {
		return "", err
	}
	// A decision invalidates old embeddings. Publication always creates a fresh one.
	_, err = tx.ExecContext(ctx, `UPDATE rag_items SET active=false,approved=false,embedding=NULL,quality_status=CASE WHEN $2='rejected' THEN 'rejected' ELSE 'pending' END,learning_data=COALESCE(learning_data,'{}')||jsonb_build_object('managed_by_review',true,'publish_state',$3::text),updated_at=now() WHERE id=$1::uuid`, id, decision, publication)
	if err != nil {
		return "", err
	}
	if err = audit(ctx, tx, id, actor, req.Action, req.Reason); err != nil {
		return "", err
	}
	return id, tx.Commit()
}
func loadLockedContent(raw []byte) (Content, error) {
	var r struct {
		Content
		Data map[string]json.RawMessage `json:"learning_data"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return Content{}, err
	}
	for k, p := range map[string]*string{"fact_statement": &r.Rule, "evidence_quote": &r.Evidence, "evidence_source": &r.Source, "limitations": &r.Limits} {
		if len(r.Data[k]) > 0 {
			_ = json.Unmarshal(r.Data[k], p)
		}
	}
	if r.Rule == "" {
		_ = json.Unmarshal(r.Data["guidance"], &r.Rule)
	}
	_ = json.Unmarshal(r.Data["original_categories"], &r.OriginalCategories)
	_ = json.Unmarshal(r.Data["corrected_categories"], &r.CorrectedCategories)
	return r.Content, nil
}
func writeContent(ctx context.Context, tx *sqlx.Tx, id string, c Content) error {
	if c.Type == "style_rule" {
		c.Answer = c.Rule
	}
	data := map[string]any{"guidance": c.Rule, "fact_statement": c.Rule, "evidence_quote": c.Evidence, "evidence_source": c.Source, "limitations": c.Limits, "original_categories": c.OriginalCategories, "corrected_categories": c.CorrectedCategories, "managed_by_review": true, "verification_status": "unverified", "factual_use_approved": false}
	switch c.Type {
	case "employee_fact":
		data["target_agent"] = "company_knowledge"
		data["authority"] = "employee_feedback"
	case "classification_example":
		data["target_agent"] = "classifier"
		data["category_correction_origin"] = "human_review"
	case "response_example", "style_rule":
		data["target_agent"] = "final_answer"
		data["example_role"] = "style_only"
		data["factual_authority"] = false
	case "factual_review":
		data["target_agent"] = "knowledge_review"
	}
	b, _ := json.Marshal(data)
	_, err := tx.ExecContext(ctx, `UPDATE rag_items SET category=$2,topic=$3,retrieval_text=$4,customer_question=$5,human_answer=$6,learning_data=COALESCE(learning_data,'{}')||$7::jsonb,embedding=NULL,active=false,approved=false,updated_at=now() WHERE id=$1::uuid`, id, c.Category, c.Topic, c.Retrieval, c.Question, c.Answer, string(b))
	return err
}
func audit(ctx context.Context, tx *sqlx.Tx, id, actor, action, reason string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO learning_review_audit(item_id,revision,actor,action,reason,snapshot) SELECT r.id,v.revision,$2,$3,$4,to_jsonb(r)-'embedding' FROM rag_items r JOIN learning_reviews v ON v.item_id=r.id WHERE r.id=$1::uuid`, id, actor, action, reason)
	return err
}
func (m *Manager) Run(ctx context.Context) {
	t := time.NewTicker(15 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if m.Embed != nil {
				if err := m.PublishOne(ctx); err != nil && m.ReportFailure != nil {
					m.ReportFailure("publish", err)
				}
			}
		}
	}
}
func vectorLiteral(v []float64) string { b, _ := json.Marshal(v); return string(b) }
func errorLabel(err error) string {
	if err == ErrInput {
		return "publication_validation_failed"
	}
	return "embedding_or_database_unavailable"
}

// Publication privacy checks remain mandatory independently of display configuration.
func (m *Manager) privacyChecked(ctx context.Context, c Content) error {
	if m.Anonymize == nil {
		return &DependencyError{Service: "anonymizer", Kind: "not_configured"}
	}
	for _, value := range []string{c.Topic, c.Retrieval, c.Question, c.Answer, c.Rule, c.Evidence, c.Source, c.Limits} {
		clean, err := m.safe(ctx, value)
		if err != nil {
			return err
		}
		if strings.TrimSpace(clean) != strings.TrimSpace(value) {
			return ErrInput
		}
	}
	return nil
}
