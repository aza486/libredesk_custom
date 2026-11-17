package main

import (
	"context"
	"encoding/json"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestLearningCaseDecisionStrictInput(t *testing.T) {
	for _, body := range []string{`[]`, `{"action":"approve"}`, `{"action":"approve","revision":null}`, `{"action":"approve","revision":0,"actor":"999"}`, `{"action":"approve","revision":0,"agent_id":999}`, `{"action":"approve","revision":0,"checks":{"general":true}}`, `{"action":"approve","revision":0,"checks":{"privacy":"true"}}`, `{"action":"approve","revision":0} {}`} {
		if decodeLearningCaseDecision([]byte(body), new(learning.CaseDecision)) == nil {
			t.Fatal("untrusted field accepted", body)
		}
	}
	var d learning.CaseDecision
	if err := decodeLearningCaseDecision([]byte(`{"action":"approve","revision":2,"checks":{"privacy":true,"generalization":true,"quality":true}}`), &d); err != nil || d.Revision != 2 || !d.Checks.Quality {
		t.Fatal(err, d)
	}
	// Existing fact API still refuses all browser-provided quality flags.
	if decodeLearningDecision([]byte(`{"action":"learn","revision":0,"checks":{"privacy":true}}`), new(learning.Decision)) == nil {
		t.Fatal("fact API changed")
	}
}

func TestLearningCaseHandlerAuthorizationAndActor(t *testing.T) {
	db := testutil.NewPersonalDB(t, "learning_case_api")
	db.MustExec(`CREATE EXTENSION IF NOT EXISTS vector;
 CREATE TABLE ai_feedback(id uuid PRIMARY KEY,mail_event_message_uuid text,outgoing_message_uuid text,agent_id text,feedback_reason text,human_answer text,feedback_type text,quality_status text DEFAULT 'pending',created_at timestamp DEFAULT now(),updated_at timestamp DEFAULT now(),feedback_data jsonb DEFAULT '{}',learning_status text);
 CREATE TABLE rag_items(id uuid PRIMARY KEY,source_type text,scope text,agent_id text,category text,subcategory text,topic text,customer_question text,ai_answer text,human_answer text,feedback_reason text,retrieval_text text,embedding vector(1536),quality_status text DEFAULT 'pending',approved boolean DEFAULT false,active boolean DEFAULT false,source_id text,created_at timestamp DEFAULT now(),updated_at timestamp DEFAULT now(),feedback_id uuid REFERENCES ai_feedback(id),learning_data jsonb DEFAULT '{}');`)
	db.MustExec(learning.Schema)
	db.MustExec(learning.CaseV11Schema)
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	conv, err := conversation.New(nil, lang, nil, nil, nil, nil, users, nil, nil, nil, nil, nil, nil, nil, nil, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := authz.NewEnforcer(&lo, lang)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{ctx: context.Background(), user: users, conversation: conv, authz: enforcer, learning: &learning.Manager{DB: db}}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Case Reviewer',ARRAY['conversations:read','conversations:read_assigned','ai_learning:manage']);
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN (101,102) AND r.name='Case Reviewer'`)
	var cid int
	var original, outgoing string
	db.Get(&cid, `INSERT INTO conversations(contact_id,inbox_id,status_id,assigned_user_id) VALUES(105,101,101,101) RETURNING id`)
	db.Get(&original, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','received',105,'contact','Schaden am gelieferten Artikel') RETURNING uuid`, cid)
	db.Get(&outgoing, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'outgoing','sent',101,'agent','Tatsächlich gesendete menschliche Antwort') RETURNING uuid`, cid)
	db.MustExec(`UPDATE conversation_messages SET text_content=content,content_type='html'`)
	fid := uuid.NewString()
	raw, _ := json.Marshal(map[string]any{"response_source": "human", "learning_case_v11": map[string]any{
		"version": 11, "status": "staged", "customer_question": "Wie wird ein beschädigter Artikel reklamiert?", "human_answer": "Im historischen Fall wurde Bestellnummer: [Bestellnummer] geprüft und ein Ersatz angeboten. Daraus folgt kein aktueller Anspruch.", "feedback_reason": "", "feedback_type": "manual_resolution", "original_categories": []string{}, "corrected_categories": []string{}, "has_human_resolution": true, "has_explicit_feedback": false, "requires_quality_review": true,
	}})
	db.MustExec(`INSERT INTO ai_feedback(id,mail_event_message_uuid,outgoing_message_uuid,agent_id,human_answer,feedback_type,feedback_data) VALUES($1,$2,$3,'101','Persistierte tatsächlich gesendete Lösung','manual_resolution',$4)`, fid, original, outgoing, string(raw))
	var id string
	if err := db.Get(&id, `SELECT learning_case_stage_v11($1)`, fid); err != nil {
		t.Fatal(err)
	}
	request := func(agent int, method, body string) int {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetBodyString(body)
		ctx.SetUserValue("auth_method", method)
		ctx.SetUserValue("id", id)
		if agent > 0 {
			ctx.SetUserValue("user", amodels.User{ID: agent})
		}
		if err := handleLearningCaseReview(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		return ctx.Response.StatusCode()
	}
	approve := `{"action":"approve","revision":0,"checks":{"privacy":true,"generalization":true,"quality":true}}`
	for _, tt := range []struct {
		agent  int
		method string
		want   int
	}{{0, authMethodSession, 403}, {105, authMethodSession, 403}, {103, authMethodSession, 403}, {101, "api_key", 403}, {102, authMethodSession, 403}} {
		if got := request(tt.agent, tt.method, approve); got != tt.want {
			t.Fatal("authorization", tt, got)
		}
	}
	if got := request(101, authMethodSession, `{"action":"approve","revision":0,"actor":"999"}`); got != 400 {
		t.Fatal("actor spoof", got)
	}
	if got := request(101, authMethodSession, `{"action":"approve","revision":0,"checks":{"privacy":true,"quality":true}}`); got != 400 {
		t.Fatal("generalization missing", got)
	}
	// Actual outgoing AI provenance contradicts the feedback assertion: fail closed.
	db.MustExec(`UPDATE conversation_messages SET meta=jsonb_build_object('ai_response',jsonb_build_object('response_source','ai_adapted','source_message_uuid',$2::text)) WHERE uuid=$1`, outgoing, original)
	if got := request(101, authMethodSession, approve); got != 403 {
		t.Fatal("AI source", got)
	}
	db.MustExec(`UPDATE conversation_messages SET meta='{}' WHERE uuid=$1`, outgoing)
	if got := request(101, authMethodSession, approve); got != 200 {
		t.Fatal("valid human decision", got)
	}
	var actor string
	db.Get(&actor, `SELECT learning_data->>'reviewer' FROM rag_items WHERE id=$1`, id)
	if actor != "101" {
		t.Fatal("session actor not used", actor)
	}
	if got := request(101, authMethodSession, approve); got != 409 {
		t.Fatal("stale revision", got)
	}
	var active bool
	db.Get(&active, `SELECT active OR approved OR embedding IS NOT NULL FROM rag_items WHERE id=$1`, id)
	if active {
		t.Fatal("automatic publication")
	}
	if got := request(101, authMethodSession, `{"action":"block","revision":1}`); got != 200 {
		t.Fatal("blocking", got)
	}
}
