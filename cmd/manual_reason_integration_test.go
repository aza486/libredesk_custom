package main

import (
	"context"
	"encoding/json"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestManualReasonHandlerPostgresAuthorization(t *testing.T) {
	db := testutil.NewPersonalDB(t, "manual_reason_http")
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
	app := &App{ctx: context.Background(), user: users, conversation: conv, authz: enforcer}
	db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Admin'`)
	var c, original, note, ordinary, foreign string
	var cid int
	if e = db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid,id`).Scan(&c, &cid); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`).Scan(&foreign); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','received',105,'contact','Input') RETURNING uuid`, cid).Scan(&original); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content,meta) VALUES($1,'outgoing','sent',101,'agent',true,'Review',jsonb_build_object('ai_analysis',jsonb_build_object('kind','n8n_manual_review','version',1,'original_message_uuid',$2::text))) RETURNING uuid`, cid, original).Scan(&note); e != nil {
		t.Fatal(e)
	}
	if e = db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content) VALUES($1,'outgoing','sent',101,'agent',true,'ordinary') RETURNING uuid`, cid).Scan(&ordinary); e != nil {
		t.Fatal(e)
	}
	call := func(method, cuuid, n string, agent int, body string, want int) map[string]any {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(method)
		ctx.Request.SetBodyString(body)
		ctx.SetUserValue("auth_method", authMethodSession)
		ctx.SetUserValue("user", amodels.User{ID: agent})
		ctx.SetUserValue("cuuid", cuuid)
		ctx.SetUserValue("uuid", n)
		if e := handleManualReviewReason(&fastglue.Request{RequestCtx: ctx, Context: app}); e != nil {
			t.Fatal(e)
		}
		if ctx.Response.StatusCode() != want {
			t.Fatalf("%s: status %d wanted %d", method, ctx.Response.StatusCode(), want)
		}
		var data struct {
			Data map[string]any `json:"data"`
		}
		_ = json.Unmarshal(ctx.Response.Body(), &data)
		return data.Data
	}
	got := call("PUT", c, note, 101, `{"feedback_reason":"  apology  "}`, 200)
	if got["agent_id"] != float64(101) || got["feedback_reason"] != "apology" || got["original_message_uuid"] != original {
		t.Fatal("wrong server binding")
	}
	call("GET", c, note, 102, "", 403)
	call("GET", foreign, note, 101, "", 404)
	call("PUT", c, ordinary, 101, `{"feedback_reason":"x"}`, 404)
	call("PUT", c, note, 101, `{"feedback_reason":"forged","agent_id":102}`, 400)
	call("DELETE", c, note, 101, "", 200)
	if got = call("GET", c, note, 101, "", 200); got["feedback_reason"] != nil {
		t.Fatal("delete did not clear annotation")
	}
	db.MustExec(`INSERT INTO manual_resolution_feedback(conversation_id,original_message_uuid,outgoing_message_uuid,note_uuid,agent_id,human_answer,sent_at) VALUES($1,$2,gen_random_uuid(),$3,101,'sent HTML',NOW())`, cid, original, note)
	call("PUT", c, note, 101, `{"feedback_reason":"late"}`, 409)
	call("DELETE", c, note, 101, "", 409)
	db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"deleted_at":"now"}' WHERE uuid=$1`, note)
	call("GET", c, note, 101, "", 404)
	db.MustExec(`UPDATE users SET enabled=false WHERE id=103`)
	call("GET", c, note, 103, "", 403)
}
