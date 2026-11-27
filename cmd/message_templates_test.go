package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/messagetemplate"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestMessageTemplateHTTPAuthorization(t *testing.T) {
	db := testutil.NewPersonalDB(t, "message_template_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	conv, err := conversation.New(nil, lang, nil, nil, nil, nil, users, nil, customerReturnMedia{}, nil, nil, nil, nil, nil, nil, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, _ := authz.NewEnforcer(&lo, lang)
	app := &App{user: users, conversation: conv, authz: enforcer, i18n: lang, lo: &lo, messageTemplates: messagetemplate.New(db)}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Writer',ARRAY['messages:write','conversations:read','conversations:read_all']),('Kundensupport',ARRAY['conversations:read','messages:write']);
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN(101,102,103) AND r.name='Writer';
 INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport';`)
	if err := migrations.V2_25_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	var public, private string
	db.Get(&public, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject) VALUES(105,102,101,'Support') RETURNING uuid`)
	db.Get(&private, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,101,101,'Private','{"access_mode":"personal"}') RETURNING uuid`)
	request := func(actor int, method, uuid, context, body string, handler fastglue.FastRequestHandler, status int) []byte {
		t.Helper()
		key, secret, err := users.GenerateAPIKey(actor)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(method)
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(key+":"+secret)))
		ctx.SetUserValue("uuid", uuid)
		ctx.Request.URI().QueryArgs().Set("conversation_uuid", context)
		ctx.Request.URI().QueryArgs().Set("revision", "1")
		ctx.Request.SetBodyString(body)
		if err := perm(handler, "messages:write")(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != status {
			t.Fatalf("%s actor=%d expected %d got %d: %s", method, actor, status, ctx.Response.StatusCode(), ctx.Response.Body())
		}
		return append([]byte(nil), ctx.Response.Body()...)
	}
	for _, field := range []string{`"owner_id":102`, `"owner_id":101`, `"created_by":102`, `"updated_by":102`, `"inbox_id":102`} {
		request(101, "POST", "", "", `{"name":"Forged","content":"text","scope":"personal",`+field+`}`, handleCreateMessageTemplate, 400)
	}
	request(101, "POST", "", "", `{"name":"Forged","content":"text","scope":"invalid"}`, handleCreateMessageTemplate, 400)
	request(101, "POST", "", public, `{"name":"Forged","content":"text","scope":"customer_support"}`, handleCreateMessageTemplate, 403)
	var result struct {
		Data messagetemplate.Template `json:"data"`
	}
	json.Unmarshal(request(101, "POST", "", private, `{"name":"Private","content":"<p>Safe</p>","scope":"personal"}`, handleCreateMessageTemplate, 200), &result)
	request(101, "GET", "invalid-uuid", "", "", handleGetMessageTemplate, 404)
	request(101, "GET", "", "invalid-uuid", "", handleListMessageTemplates, 400)
	uuid := result.Data.UUID
	if uuid == "" {
		t.Fatal("missing UUID")
	}
	for _, actor := range []int{102, 103} {
		request(actor, "GET", uuid, "", "", handleGetMessageTemplate, 404)
		request(actor, "PUT", uuid, "", `{"name":"Stolen","content":"text","scope":"personal","revision":1}`, handleUpdateMessageTemplate, 404)
		request(actor, "DELETE", uuid, "", "", handleDeleteMessageTemplate, 404)
	}
	request(101, "GET", uuid, private, "", handleGetMessageTemplate, 200)
	// Even support cannot substitute a private conversation as shared context.
	request(103, "GET", "", private, "", handleListMessageTemplates, 403)
	json.Unmarshal(request(103, "POST", "", public, `{"name":"Shared","content":"<p>Shared</p>","scope":"customer_support"}`, handleCreateMessageTemplate, 200), &result)
	global := result.Data.UUID
	request(103, "PUT", global, public, `{"name":"Changed","content":"text","scope":"customer_support","revision":1}`, handleUpdateMessageTemplate, 200)
	request(103, "PUT", global, public, `{"name":"Old","content":"text","scope":"customer_support","revision":1}`, handleUpdateMessageTemplate, 409)
	request(103, "DELETE", global, public, "", handleDeleteMessageTemplate, 409)
	db.MustExec(`UPDATE roles SET permissions=array_remove(permissions,'message_templates:customer_support') WHERE name='Kundensupport'`)
	request(103, "GET", global, public, "", handleGetMessageTemplate, 404)
	request(103, "PUT", global, public, `{"name":"Old","content":"text","scope":"customer_support","revision":2}`, handleUpdateMessageTemplate, 404)
	db.MustExec(`UPDATE roles SET permissions=array_remove(permissions,'messages:write') WHERE name IN('Writer','Kundensupport')`)
	request(101, "GET", uuid, "", "", handleGetMessageTemplate, 403)
	// Table constraints remain a backstop even outside the API.
	for _, values := range []string{`'personal',NULL`, `'customer_support',101`, `'invalid',NULL`} {
		if _, err := db.Exec(fmt.Sprintf(`INSERT INTO message_templates(name,content,scope,owner_id) VALUES('Invalid','Text',%s)`, values)); err == nil {
			t.Fatal("invalid scope/owner accepted", values)
		}
	}
}
