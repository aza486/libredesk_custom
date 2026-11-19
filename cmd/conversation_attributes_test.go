package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	texttemplate "text/template"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	customattribute "github.com/abhinavxd/libredesk/internal/custom_attribute"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/media"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/team"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/abhinavxd/libredesk/internal/webhook"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

// Real PostgreSQL managers and API-key middleware; no inbox receivers, outgoing
// workers or webhook delivery workers are started by this fixture.
func systemAttributesHTTPFixture(t *testing.T, name string) (*App, *sqlx.DB) {
	t.Helper()
	db := testutil.NewPersonalDB(t, name)
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
	teams, err := team.New(team.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	attrs, err := customattribute.New(customattribute.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	files, err := media.New(media.Opts{DB: db, Lo: &lo, I18n: lang, Store: accessTestBlobStore{}, RootURL: func() string { return "https://desk.example" }})
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New(&lo, db, nil, nil, texttemplate.FuncMap{"RootURL": func() string { return "https://desk.example" }}, lang)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := automation.New(automation.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	hooks, err := webhook.New(webhook.Opts{DB: db, Lo: &lo, I18n: lang, QueueSize: 256})
	if err != nil {
		t.Fatal(err)
	}
	notifications, err := notifier.NewUserNotificationManager(notifier.UserNotificationOpts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := notifier.NewDispatcher(notifier.DispatcherOpts{InApp: notifications, Lo: &lo})
	conv, err := conversation.New(ws.NewHub(&lo, nil), lang, customerReturnSLA{}, nil, nil, inboxes, users, teams, files, customerReturnSettings{}, nil, engine, tmpl, hooks, dispatcher, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, _ := authz.NewEnforcer(&lo, lang)
	app := &App{ctx: context.Background(), user: users, inbox: inboxes, team: teams, media: files, tmpl: tmpl, customAttribute: attrs, webhook: hooks, conversation: conv, authz: enforcer, i18n: lang, lo: &lo}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES
 ('Security creator',ARRAY['conversations:write','conversations:read','conversations:read_all','conversations:update_user_assignee','conversations:update_team_assignee','messages:read']),
 ('Security observer',ARRAY['conversations:read','conversations:read_all','conversations:read_unassigned','messages:read']),
 ('Kundensupport',ARRAY['conversations:read','messages:read']);
 INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Security creator';
 INSERT INTO user_roles(user_id,role_id) SELECT u,id FROM roles CROSS JOIN unnest(ARRAY[102,103]) u WHERE name='Security observer';
 INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport';
 INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin';
 INSERT INTO tags(id,name) VALUES(8,'🏢Intern');
 INSERT INTO teams(id,name,conversation_assignment_type,timezone) VALUES(201,'Internal team','Manual','UTC');
 INSERT INTO templates(name,type,is_default,body) VALUES('Test outgoing','email_outgoing',true,'{{ template "content" . }}');
 INSERT INTO team_members(team_id,user_id) VALUES(201,102);`)
	return app, db
}

func systemAttributesRequest(t *testing.T, app *App, actor int, method, uuid, body string, handler fastglue.FastRequestHandler, want int) []byte {
	t.Helper()
	key, secret, err := app.user.GenerateAPIKey(actor)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(method)
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(key+":"+secret)))
	ctx.Request.SetBodyString(body)
	ctx.SetUserValue("uuid", uuid)
	if err := handler(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
		t.Fatal(err)
	}
	if ctx.Response.StatusCode() != want {
		t.Fatalf("%s as %d: status %d want %d: %s", method, actor, ctx.Response.StatusCode(), want, ctx.Response.Body())
	}
	return append([]byte(nil), ctx.Response.Body()...)
}

func systemAttributesCreateBody(attrs string, inboxID, agentID, teamID int) string {
	return fmt.Sprintf(`{"inbox_id":%d,"agent_id":%d,"team_id":%d,"contact_email":"contact@example.com","first_name":"Contact","subject":"Attribute security","content":"<p>Initial message</p>","initiator":"agent","custom_attributes":%s}`, inboxID, agentID, teamID, attrs)
}

func TestConversationSystemAttributesCreateHTTP(t *testing.T) {
	app, db := systemAttributesHTTPFixture(t, "system_attributes_create_http")
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM custom_attribute_definitions`); err != nil || count != 0 {
		t.Fatal("fixture must have no custom attribute definitions", err)
	}
	for _, tc := range []struct {
		name, attrs        string
		inbox, agent, team int
	}{
		{"internal private", `{"private":true,"internal":true}`, 102, 0, 0},
		{"private assigned", `{"private":true}`, 102, 102, 0},
		{"private team", `{"private":true,"internal":true}`, 102, 0, 201},
		{"public", `{}`, 102, 0, 0},
		{"personal", `{}`, 101, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(tc.attrs, tc.inbox, tc.agent, tc.team), perm(handleCreateConversation, "conversations:write"), 200)
			var response struct {
				Data cmodels.Conversation `json:"data"`
			}
			if err := json.Unmarshal(body, &response); err != nil || response.Data.UUID == "" {
				t.Fatalf("invalid create response: %s %v", body, err)
			}
			conv, err := app.conversation.GetConversation(0, response.Data.UUID, "")
			if err != nil {
				t.Fatal(err)
			}
			var attrs map[string]any
			if err := json.Unmarshal(conv.CustomAttributes, &attrs); err != nil {
				t.Fatal(err)
			}
			if tc.name == "public" {
				if attrs["private"] != nil || attrs["internal"] != nil || attrs["creator_id"] != nil || attrs["customer_visibility"] != true || !reflect.DeepEqual(attrs["visible_users"], []any{float64(103)}) {
					t.Fatalf("public initialization changed: %s", conv.CustomAttributes)
				}
				return
			}
			if tc.name == "personal" {
				if attrs["access_mode"] != "personal" || attrs["owner_user_id"] != float64(101) || attrs["customer_visibility"] != false || attrs["creator_id"] != nil {
					t.Fatalf("personal initialization changed: %s", conv.CustomAttributes)
				}
				return
			}
			if attrs["private"] != true || attrs["creator_id"] != float64(101) || attrs["customer_visibility"] != nil || (tc.name != "private assigned" && attrs["internal"] != true) {
				t.Fatalf("private initialization lost: %s", conv.CustomAttributes)
			}
			visible := attrs["visible_users"].([]any)
			if !slices.Contains(visible, any(float64(101))) || (tc.agent > 0 || tc.team > 0) && !slices.Contains(visible, any(float64(102))) {
				t.Fatalf("missing private visibility: %v", visible)
			}
			for _, id := range []int{101, 102, 103, 104} {
				want := 403
				if id == 101 || id == 104 || id == 102 && (tc.agent > 0 || tc.team > 0) {
					want = 200
				}
				systemAttributesRequest(t, app, id, "GET", conv.UUID, "", perm(handleGetConversation, "conversations:read"), want)
			}
		})
	}
}

func TestConversationSystemAttributesCreateRejectsHTTP(t *testing.T) {
	app, db := systemAttributesHTTPFixture(t, "system_attributes_create_rejects")
	for _, attrs := range []string{
		`{"private":"true"}`, `{"private":1}`, `{"private":null}`, `{"private":[]}`, `{"internal":"true"}`, `{"internal":null}`,
		`{"internal":true}`, `{"private":false,"internal":true}`,
	} {
		t.Run(attrs, func(t *testing.T) {
			systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(attrs, 102, 0, 0), perm(handleCreateConversation, "conversations:write"), 400)
		})
	}
	// Defining a reserved key must never make ACL injection an allowed input.
	db.MustExec(`INSERT INTO custom_attribute_definitions(applies_to,name,description,key,data_type) VALUES('conversation','Creator','Forged definition','creator_id','text')`)
	for _, key := range []string{"creator_id", "visible_users", "visibility_managers", "customer_visibility", "access_mode", "owner_user_id", "_spam_previous_state"} {
		systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(fmt.Sprintf(`{"%s":null}`, key), 102, 0, 0), perm(handleCreateConversation, "conversations:write"), 403)
	}
	systemAttributesRequest(t, app, 102, "POST", "", systemAttributesCreateBody(`{"private":true}`, 102, 0, 0), perm(handleCreateConversation, "conversations:write"), 403)
	for _, assignment := range [][2]int{{-1, 0}, {0, -1}, {999999, 0}} {
		want := 400
		if assignment[0] == 999999 {
			want = 404
		}
		systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(`{"private":true}`, 102, assignment[0], assignment[1]), perm(handleCreateConversation, "conversations:write"), want)
	}
	contactBody := strings.Replace(systemAttributesCreateBody(`{"private":true}`, 102, 0, 0), `"initiator":"agent"`, `"initiator":"contact"`, 1)
	systemAttributesRequest(t, app, 101, "POST", "", contactBody, perm(handleCreateConversation, "conversations:write"), 400)
	db.MustExec(`UPDATE roles SET permissions=array_remove(array_remove(permissions,'conversations:update_user_assignee'),'conversations:update_team_assignee') WHERE name='Security creator'`)
	for _, assignment := range [][2]int{{102, 0}, {0, 201}} {
		systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(`{"private":true}`, 102, assignment[0], assignment[1]), perm(handleCreateConversation, "conversations:write"), 403)
	}
	systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(`{"private":true}`, 101, 0, 0), perm(handleCreateConversation, "conversations:write"), 400)
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM conversations`); err != nil || count != 0 {
		t.Fatal("rejected create persisted a ticket", count, err)
	}
}

func TestConversationSystemAttributesUpdateHTTP(t *testing.T) {
	app, db := systemAttributesHTTPFixture(t, "system_attributes_update_http")
	for _, tc := range []struct {
		name, attrs string
		inbox       int
		customer    bool
	}{
		{"private", `{"private":true,"internal":true,"creator_id":101,"visible_users":[101],"visibility_managers":[101],"customer_visibility":false,"_spam_previous_state":{"status_id":101}}`, 102, false},
		{"customer", `{"customer_visibility":true,"visibility_managers":[101],"creator_id":101,"visible_users":[101]}`, 102, true},
		{"personal", `{"access_mode":"personal","owner_user_id":101,"customer_visibility":false,"visibility_managers":[101],"visible_users":[101]}`, 101, false},
		{"ordinary", `{}`, 102, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var uuid string
			if err := db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,$1,101,'Attribute test',$2::jsonb || '{"ordinary":"old"}') RETURNING uuid`, tc.inbox, tc.attrs); err != nil {
				t.Fatal(err)
			}
			if tc.customer {
				db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, uuid)
			}
			var expected map[string]any
			json.Unmarshal([]byte(tc.attrs), &expected)
			for _, body := range []string{`{}`, `{"ordinary":"new"}`} {
				systemAttributesRequest(t, app, 101, "PUT", uuid, body, auth(handleUpdateConversationCustomAttributes), 200)
				conv, err := app.conversation.GetConversation(0, uuid, "")
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				json.Unmarshal(conv.CustomAttributes, &got)
				if body != `{}` {
					expected["ordinary"] = "new"
				}
				if !reflect.DeepEqual(got, expected) {
					t.Fatalf("system fields changed: got %s expected %v", conv.CustomAttributes, expected)
				}
			}
			for _, key := range []string{"private", "internal", "creator_id", "visible_users", "visibility_managers", "customer_visibility", "access_mode", "owner_user_id", "_spam_previous_state"} {
				for _, value := range []string{`null`, `false`, `[]`} {
					systemAttributesRequest(t, app, 101, "PUT", uuid, fmt.Sprintf(`{"%s":%s}`, key, value), auth(handleUpdateConversationCustomAttributes), 403)
				}
			}
			if tc.name == "private" || tc.name == "personal" {
				systemAttributesRequest(t, app, 103, "GET", uuid, "", perm(handleGetConversation, "conversations:read"), 403)
				systemAttributesRequest(t, app, 102, "PUT", uuid, `{}`, auth(handleUpdateConversationCustomAttributes), 403)
			}
			// Existing dedicated visibility API remains functional and authoritative.
			if tc.name != "ordinary" {
				systemAttributesRequest(t, app, 101, "PUT", uuid, `{"user_id":102}`, auth(handleAddVisibleUser), 200)
				systemAttributesRequest(t, app, 102, "GET", uuid, "", perm(handleGetConversation, "conversations:read"), 200)
				systemAttributesRequest(t, app, 101, "PUT", uuid, `{"user_id":102}`, auth(handleRemoveVisibleUser), 200)
				systemAttributesRequest(t, app, 102, "GET", uuid, "", perm(handleGetConversation, "conversations:read"), 403)
			}
		})
	}
}
