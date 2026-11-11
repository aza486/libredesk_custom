package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/zerodha/logf"
)

func TestLearningOriginalContextConversationAuthorization(t *testing.T) {
	db := testutil.NewPersonalDB(t, "learning_context")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, e := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if e != nil {
		t.Fatal(e)
	}
	conv, e := conversation.New(nil, lang, nil, nil, nil, nil, users, nil, nil, nil, nil, nil, nil, nil, nil, conversation.Opts{DB: db, Lo: &lo})
	if e != nil {
		t.Fatal(e)
	}
	enforcer, e := authz.NewEnforcer(&lo, lang)
	if e != nil {
		t.Fatal(e)
	}
	app := &App{conversation: conv, authz: enforcer, user: users}
	var cid, other int
	var original, outgoing, foreign string
	db.Get(&cid, `INSERT INTO conversations(contact_id,inbox_id,status_id,assigned_user_id) VALUES(105,101,101,101) RETURNING id`)
	db.Get(&other, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING id`)
	db.Get(&original, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','received',105,'contact','<p>Original Kunde Beispiel</p>') RETURNING uuid`, cid)
	db.Get(&outgoing, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'outgoing','sent',101,'agent','Changed editor content') RETURNING uuid`, cid)
	db.Get(&foreign, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'outgoing','sent',101,'agent','Foreign content') RETURNING uuid`, other)
	db.MustExec(`UPDATE conversation_messages SET text_content=content,content_type='html'`)
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Learning context test',ARRAY['conversations:read','conversations:read_assigned','ai_learning:manage']);
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN (101,102) AND r.name='Learning context test'`)
	owner, e := users.GetAgentCachedOrLoad(101)
	if e != nil {
		t.Fatal(e)
	}
	stranger, e := users.GetAgentCachedOrLoad(102)
	if e != nil {
		t.Fatal(e)
	}
	owner.Permissions = []string{"conversations:read", "conversations:read_assigned", "ai_learning:manage"}
	stranger.Permissions = append([]string{}, owner.Permissions...)
	makeItem := func(reply string) *learning.Item {
		p, _ := json.Marshal(map[string]any{"original_message_uuid": original, "outgoing_message_uuid": reply})
		return &learning.Item{Provenance: p, SentAnswer: "<p>Persistierter Transporttext</p>"}
	}
	item := makeItem(outgoing)
	if e = learningView(app, owner)(context.Background(), item); e != nil {
		t.Fatal(e)
	}
	if item.Context == nil || item.Context.Question != "Original Kunde Beispiel" || item.Context.Answer != "Persistierter Transporttext" || item.SentAnswer != "" {
		t.Fatal("wrong display source", item.Context)
	}
	if e = learningView(app, stranger)(context.Background(), makeItem(outgoing)); !errors.Is(e, learning.ErrForbidden) {
		t.Fatal("cross-ticket access", e)
	}
	if e = learningView(app, owner)(context.Background(), makeItem(foreign)); !errors.Is(e, learning.ErrForbidden) {
		t.Fatal("mixed conversation", e)
	}
	db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"deleted_at":"now"}' WHERE uuid=$1`, original)
	if e = learningView(app, owner)(context.Background(), makeItem(outgoing)); !errors.Is(e, learning.ErrForbidden) {
		t.Fatal("deleted source", e)
	}
}

func TestLearningDecisionRejectsBrowserFlags(t *testing.T) {
	for _, body := range []string{`{"action":"learn"}`, `{"action":"learn","revision":null}`, `{"action":"learn","revision":0,"content":{}}`, `{"action":"learn","revision":0,"checks":{"general":true}}`, `{"action":"learn","revision":0,"agent_id":1}`, `{"action":"learn","reason":"pretend"}`} {
		if decodeLearningDecision([]byte(body), new(learning.Decision)) == nil {
			t.Fatal("browser field accepted", body)
		}
	}
	var d learning.Decision
	if e := decodeLearningDecision([]byte(`{"action":"learn","revision":1}`), &d); e != nil || d.Action != "learn" || d.Revision != 1 {
		t.Fatal("two-decision contract", e, d)
	}
}
