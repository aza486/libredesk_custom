package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/knadh/koanf/v2"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestFormRecipientConfig(t *testing.T) {
	previous := ko
	ko = koanf.New(".")
	defer func() { ko = previous }()
	config := initFormRecipients()
	if config.DB != nil || len(config.TechnicalSenders) != 1 || config.TechnicalSenders[0] != "bestellung@pixelfotoexpress.de" {
		t.Fatal("unsafe defaults")
	}
	ko.Set("form_recipients.technical_senders", []string{"forms@example.com"})
	ko.Set("form_recipients.database_dsn", "postgres://readonly:synthetic@127.0.0.1:1/forms?connect_timeout=1&sslmode=disable")
	config = initFormRecipients()
	if config.DB == nil || config.TechnicalSenders[0] != "forms@example.com" || config.DB.Stats().MaxOpenConnections != 2 {
		t.Fatal("configuration ignored")
	}
	config.DB.Close()
}

func TestFormRecipientDetailAuthorization(t *testing.T) {
	db := testutil.NewPersonalDB(t, "form_recipient_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := conversation.New(nil, lang, nil, nil, nil, nil, users, nil, nil, nil, nil, nil, nil, nil, nil, conversation.Opts{DB: db, Lo: &lo, FormRecipients: &conversation.FormRecipients{DB: db, TechnicalSenders: []string{"bestellung@pixelfotoexpress.de"}}})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := authz.NewEnforcer(&lo, lang)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{ctx: context.Background(), conversation: manager, user: users, authz: enforcer, lo: &lo, i18n: lang}
	db.MustExec(`CREATE TABLE mail_events(conversation_uuid text,message_uuid text,is_form boolean,direction text,sender_email text);
 CREATE TABLE cases(conversation_uuid text,customer_email text);
 UPDATE users SET email='bestellung@pixelfotoexpress.de' WHERE id=105;
 INSERT INTO roles(name,permissions) VALUES('Form employee',ARRAY['conversations:read','conversations:read_assigned','conversations:read_all','messages:write']);
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN(101,102) AND r.name='Form employee'`)
	var uuid, original string
	if err = db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,101,101,'Formular','{"access_mode":"personal","owner_user_id":101}') RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	if err = db.Get(&original, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) SELECT id,'incoming','received',105,'contact','Form' FROM conversations WHERE uuid=$1 RETURNING uuid`, uuid); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO mail_events VALUES($1,$2,true,'inbound','private-customer@example.com')`, uuid, original)
	request := func(id int) *fasthttp.RequestCtx {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("user", amodels.User{ID: id})
		ctx.SetUserValue("uuid", uuid)
		if err := handleGetConversation(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
	ctx := request(101)
	var result struct{ Data cmodels.Conversation }
	if ctx.Response.StatusCode() != 200 || json.Unmarshal(ctx.Response.Body(), &result) != nil || result.Data.ReplyRecipient == nil || result.Data.ReplyRecipient.Email != "private-customer@example.com" {
		t.Fatalf("authorized detail: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	ctx = request(102)
	if ctx.Response.StatusCode() != 403 || strings.Contains(string(ctx.Response.Body()), "private-customer@example.com") {
		t.Fatalf("personal inbox leaked: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	// With explicit visibility revoked, even the inbox owner cannot read recipients.
	db.MustExec(`UPDATE conversations SET custom_attributes='{"private":true,"creator_id":102,"visible_users":[102]}' WHERE uuid=$1`, uuid)
	ctx = request(101)
	if ctx.Response.StatusCode() != 403 || strings.Contains(string(ctx.Response.Body()), "private-customer@example.com") {
		t.Fatalf("visibility leaked: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

// Confirmation comes from the authenticated employee send, never request metadata
// or an integration key; no sender worker or real mail transport is started.
func TestFormRecipientEmployeeConfirmationHTTP(t *testing.T) {
	f := newNotificationHTTP(t, "form_confirmation_http")
	var uuid string
	if err := f.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject) VALUES(105,102,101,'Formular') RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	send := func(method string) *fasthttp.RequestCtx {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("user", amodels.User{ID: 101})
		ctx.SetUserValue("cuuid", uuid)
		ctx.SetUserValue("auth_method", method)
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBodyString(`{"sender_type":"agent","message":"Manual answer","to":["chosen@example.com","second@example.com"],"cc":["cc@example.com"],"bcc":["hidden@example.com"],"recipients_confirmed":true}`)
		if err := handleSendMessage(&fastglue.Request{RequestCtx: ctx, Context: f.app}); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
	ctx := send(authMethodSession)
	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("employee send: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var result struct {
		Data struct {
			Meta struct {
				Confirmation conversation.RecipientConfirmation `json:"recipient_confirmation"`
				To, CC, BCC  []string
			}
		}
	}
	if err := json.Unmarshal(ctx.Response.Body(), &result); err != nil {
		t.Fatal(err)
	}
	c := result.Data.Meta.Confirmation
	if c.ActorID != 101 || len(c.To) != 2 || c.To[0] != "chosen@example.com" || len(c.CC) != 1 || len(c.BCC) != 1 {
		t.Fatalf("confirmation not bound to employee recipients: %+v", c)
	}
	ctx = send(authMethodAPIKey)
	if ctx.Response.StatusCode() != 403 {
		t.Fatalf("integration key forged employee confirmation: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var count int
	f.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=(SELECT id FROM conversations WHERE uuid=$1) AND type='outgoing'`, uuid)
	if count != 1 {
		t.Fatal("rejected confirmation queued a message")
	}
}
