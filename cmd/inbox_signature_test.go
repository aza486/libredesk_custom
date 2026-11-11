package main

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestInboxSignatureHTTPPermissions(t *testing.T) {
	db := testutil.NewPersonalDB(t, "inbox_signature_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	inboxes, err := inbox.New(&lo, db, lang, "")
	if err != nil {
		t.Fatal(err)
	}
	templates, err := template.New(&lo, db, nil, nil, nil, lang)
	if err != nil {
		t.Fatal(err)
	}
	enforcer, _ := authz.NewEnforcer(&lo, lang)
	app := &App{user: users, inbox: inboxes, tmpl: templates, authz: enforcer, i18n: lang, lo: &lo}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Inbox administrator',ARRAY['inboxes:manage']);
 INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Inbox administrator';
 INSERT INTO templates(name,type,body) VALUES('Allowed signature','email_outgoing','public'),('Hidden notification','email_notification','private');`)
	request := func(userID int, method string, handler fastglue.FastRequestHandler, body []byte) *fasthttp.RequestCtx {
		t.Helper()
		key, secret, err := users.GenerateAPIKey(userID)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("id", "101")
		ctx.Request.Header.SetMethod(method)
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(key+":"+secret)))
		ctx.Request.SetBody(body)
		if err := handler(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
	// Even the personal owner needs the established inbox-management permission.
	for _, id := range []int{101, 102} {
		ctx := request(id, "PUT", perm(handleUpdateInbox, "inboxes:manage"), []byte(`{"default_signature_id":1}`))
		if ctx.Response.StatusCode() != 403 {
			t.Fatalf("unauthorized write returned %d", ctx.Response.StatusCode())
		}
		ctx = request(id, "GET", perm(handleGetOutgoingTemplates, "inboxes:manage"), nil)
		if ctx.Response.StatusCode() != 403 {
			t.Fatal("unauthorized configuration signature read")
		}
	}
	ctx := request(104, "GET", perm(handleGetOutgoingTemplates, "inboxes:manage"), nil)
	if ctx.Response.StatusCode() != 200 || !strings.Contains(string(ctx.Response.Body()), "Allowed signature") || strings.Contains(string(ctx.Response.Body()), "Hidden notification") {
		t.Fatalf("allowed list: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	ctx = request(104, "GET", perm(handleGetInbox, "inboxes:manage"), nil)
	if ctx.Response.StatusCode() != 200 || !strings.Contains(string(ctx.Response.Body()), `"default_signature_id":null`) {
		t.Fatal("nullable field missing", string(ctx.Response.Body()))
	}
	// A manipulated ID is rejected through the real update handler, not just SQL.
	var notificationID int
	if err := db.Get(&notificationID, `SELECT id FROM templates WHERE name='Hidden notification'`); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{notificationID, 9999999, -1} {
		in, err := inboxes.GetDBRecord(101)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := json.Marshal(map[string]any{
			"name": in.Name, "channel": "email", "from": "desk@example.com", "access_mode": "personal", "owner_user_id": 101,
			"default_signature_id": id,
			"config":               map[string]any{"imap": []map[string]any{{"host": "imap.invalid", "port": 993, "mailbox": "INBOX", "tls_type": "tls"}}, "smtp": []map[string]any{{"host": "smtp.invalid", "port": 587, "auth_protocol": "none"}}},
		})
		ctx = request(104, "PUT", perm(handleUpdateInbox, "inboxes:manage"), body)
		if ctx.Response.StatusCode() != 400 {
			t.Fatalf("invalid ID %s returned %d: %s", strconv.Itoa(id), ctx.Response.StatusCode(), ctx.Response.Body())
		}
	}
	var references int
	if err := db.Get(&references, `SELECT count(*) FROM inboxes WHERE default_signature_id IS NOT NULL`); err != nil || references != 0 {
		t.Fatal("denied write changed inbox", err)
	}
}
