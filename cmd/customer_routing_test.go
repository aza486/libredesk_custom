package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/media"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/sla"
	"github.com/abhinavxd/libredesk/internal/team"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx/types"
	"slices"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestCustomerRoutingVisibilityAuthorityPreserved(t *testing.T) {
	attrs := map[string]any{"customer_visibility": true, "creator_id": float64(101), "visibility_managers": []any{float64(103)}, "visible_users": []any{float64(102)}}
	ticket := &cmodels.Conversation{CustomerTicket: true, AssignedTeamID: null.IntFrom(202)}
	for _, tc := range []struct {
		name string
		user umodels.User
		want bool
	}{
		{"support", umodels.User{ID: 105, Roles: []string{"Kundensupport"}}, true},
		{"admin", umodels.User{ID: 104, Roles: []string{"Admin"}}, true},
		{"creator", umodels.User{ID: 101}, true},
		{"manager", umodels.User{ID: 103}, true},
		{"visible employee", umodels.User{ID: 102}, false},
		{"outsider", umodels.User{ID: 106}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := canManageConversationVisibility(attrs, tc.user, ticket); got != tc.want {
				t.Fatalf("management rights changed: %v", got)
			}
		})
	}
}

func TestCustomerRoutingHandlersRejectUnauthorizedAndSpam(t *testing.T) {
	db := testutil.NewPersonalDB(t, "routing_http_acl")
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
	app := &App{ctx: context.Background(), user: users, conversation: conv, authz: enforcer, i18n: lang, lo: &lo}
	db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin';
 INSERT INTO teams(id,name,conversation_assignment_type) VALUES(201,'Kundensupport','Manual'),(202,'External','Manual');
 INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	var customer, ordinary string
	for _, target := range []*string{&customer, &ordinary} {
		if err = db.Get(target, `INSERT INTO conversations(contact_id,inbox_id,status_id,assigned_team_id,custom_attributes) VALUES(105,102,101,202,'{"customer_visibility":true,"visible_users":[103]}') RETURNING uuid`); err != nil {
			t.Fatal(err)
		}
	}
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, customer)
	call := func(handler fastglue.FastRequestHandler, uuid string, agent, want int) {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod("PUT")
		ctx.SetUserValue("uuid", uuid)
		ctx.SetUserValue("auth_method", authMethodSession)
		ctx.SetUserValue("user", amodels.User{ID: agent})
		if err := handler(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != want {
			t.Fatalf("HTTP status %d want %d; %s", ctx.Response.StatusCode(), want, ctx.Response.Body())
		}
	}
	call(handleReturnCustomerToSupport, customer, 102, 403)
	call(handleGetCustomerConversations, customer, 102, 403)
	call(handleGetConversation, customer, 102, 403)
	call(handleReturnCustomerToSupport, ordinary, 104, 400)
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,12 FROM conversations WHERE uuid=$1`, customer)
	call(handleReturnCustomerToSupport, customer, 104, 400)
	var team int
	if err = db.Get(&team, `SELECT assigned_team_id FROM conversations WHERE uuid=$1`, customer); err != nil || team != 202 {
		t.Fatalf("rejected API mutation changed team: %d %v", team, err)
	}
}

type customerReturnSLA struct{ *sla.Manager }

func (customerReturnSLA) EvaluateConversationSLA(int) error { return nil }

type customerReturnMedia struct{ *media.Manager }

func (customerReturnMedia) GetByModel(int, string) ([]mmodels.Media, error) { return nil, nil }

type customerReturnInbox struct{ *inbox.Manager }

func (customerReturnInbox) Get(int) (inbox.Inbox, error) {
	return nil, fmt.Errorf("no widget in return API test")
}

type customerReturnSettings struct{ *setting.Manager }

func (customerReturnSettings) Get(string) (types.JSONText, error) {
	return types.JSONText(`"UTC"`), nil
}

type customerReturnWebhooks struct{}

func (customerReturnWebhooks) TriggerEvent(wmodels.WebhookEvent, any)        {}
func (customerReturnWebhooks) TriggerWebhook(int, wmodels.WebhookEvent, any) {}

func TestCustomerReturnAPIWithoutTeamAndWithoutGenericWrites(t *testing.T) {
	db := testutil.NewPersonalDB(t, "return_no_team_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	teams, err := team.New(team.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	engine, err := automation.New(automation.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	conv, err := conversation.New(ws.NewHub(&lo, nil), lang, customerReturnSLA{}, nil, nil, customerReturnInbox{}, users, teams, customerReturnMedia{}, customerReturnSettings{}, nil, engine, nil, customerReturnWebhooks{}, nil, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := authz.NewEnforcer(&lo, lang)
	if err != nil {
		t.Fatal(err)
	}
	app := &App{ctx: context.Background(), user: users, conversation: conv, authz: enforcer, i18n: lang, lo: &lo}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Return employee',ARRAY['conversations:read','conversations:read_assigned','conversations:read_team_all']),('Kundensupport',ARRAY['conversations:read']);
 INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Return employee';
 INSERT INTO user_roles(user_id,role_id) SELECT 102,id FROM roles WHERE name='Return employee';
 INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport';
 INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin';
 INSERT INTO teams(id,name,conversation_assignment_type) VALUES(202,'External','Manual');
 INSERT INTO team_members(team_id,user_id) VALUES(202,101)`)
	makeTicket := func(individual bool) string {
		t.Helper()
		var uuid string
		if err := db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,assigned_team_id,custom_attributes) SELECT 105,102,id,202,'{"customer_visibility":true,"creator_id":103,"visibility_managers":[103],"visible_users":[101,102,103]}' FROM conversation_statuses WHERE name='Closed' RETURNING uuid`); err != nil {
			t.Fatal(err)
		}
		db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, uuid)
		if individual {
			db.MustExec(`INSERT INTO conversation_assignees(conversation_id,user_id) SELECT id,101 FROM conversations WHERE uuid=$1`, uuid)
			db.MustExec(`UPDATE conversations SET assigned_user_id=101 WHERE uuid=$1`, uuid)
		}
		return uuid
	}
	call := func(handler fastglue.FastRequestHandler, uuid string, id, want int) []byte {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod("PUT")
		ctx.SetUserValue("auth_method", authMethodSession)
		ctx.SetUserValue("user", amodels.User{ID: id})
		ctx.SetUserValue("uuid", uuid)
		if err := handler(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != want {
			t.Fatalf("status %d want %d: %s", ctx.Response.StatusCode(), want, ctx.Response.Body())
		}
		return append([]byte(nil), ctx.Response.Body()...)
	}
	for _, tc := range []struct {
		name       string
		user       int
		individual bool
	}{{"admin", 104, true}, {"support role", 103, true}, {"own assignment without generic writes", 101, true}, {"team responsibility without generic writes", 101, false}} {
		t.Run(tc.name, func(t *testing.T) {
			uuid := makeTicket(tc.individual)
			detail := call(handleGetConversation, uuid, tc.user, 200)
			var payload struct {
				Data struct {
					Allowed bool `json:"can_return_to_support"`
				} `json:"data"`
			}
			if err := json.Unmarshal(detail, &payload); err != nil || !payload.Data.Allowed {
				t.Fatalf("missing return capability: %s %v", detail, err)
			}
			call(handleReturnCustomerToSupport, uuid, tc.user, 200)
			after, err := conv.GetConversation(0, uuid, "")
			if err != nil {
				t.Fatal(err)
			}
			if after.Status.String != "Open" || after.AssignedTeamID.Valid || after.AssignedUserID.Valid || len(after.AssignedUserIDs) != 0 || !after.CustomerTicket {
				t.Fatalf("invalid returned state: %+v", after)
			}
			var attrs map[string]any
			json.Unmarshal(after.CustomAttributes, &attrs)
			if len(attrs["visible_users"].([]any)) != 3 || attrs["creator_id"] != float64(103) {
				t.Fatal("visibility or creator changed")
			}
		})
	}
	// Read access alone, even explicit historical visibility, does not grant return.
	foreign := makeTicket(true)
	call(handleReturnCustomerToSupport, foreign, 102, 403)
	// A cached membership must not replace the current transaction's team check.
	former := makeTicket(false)
	if _, err := users.GetAgentCachedOrLoad(101); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`DELETE FROM team_members WHERE team_id=202 AND user_id=101`)
	call(handleReturnCustomerToSupport, former, 101, 403)
	// An empty later page still reports the authorized total, including when
	// no custom folders exist yet.
	active := makeTicket(false)
	db.MustExec(`UPDATE conversations SET status_id=(SELECT id FROM conversation_statuses WHERE name='Open') WHERE uuid=$1`, active)
	for _, mode := range []string{"", "&folder_mode=unfiled"} {
		request := &fasthttp.RequestCtx{}
		request.Request.SetRequestURI("/api/v1/conversations/customer?routing=redistributed&page=999&page_size=1" + mode)
		request.SetUserValue("user", amodels.User{ID: 103})
		if err := handleGetCustomerConversations(&fastglue.Request{RequestCtx: request, Context: app}); err != nil {
			t.Fatal(err)
		}
		var page struct {
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(request.Response.Body(), &page); err != nil || request.Response.StatusCode() != 200 || page.Data.Total != 1 {
			t.Fatalf("empty later-page total: %s %v", request.Response.Body(), err)
		}
		request.Request.URI().QueryArgs().Set("filters", `[{"model":"conversations","field":"subject","operator":"equals","value":"no match"}]`)
		request.Response.Reset()
		if err := handleGetCustomerConversations(&fastglue.Request{RequestCtx: request, Context: app}); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(request.Response.Body(), &page); err != nil || request.Response.StatusCode() != 200 || page.Data.Total != 0 {
			t.Fatalf("empty filtered later-page total: %s %v", request.Response.Body(), err)
		}

	}
	// The dedicated action grants no permission to invoke ordinary write handlers.
	employee, err := users.GetAgentCachedOrLoad(101)
	if err != nil {
		t.Fatal(err)
	}
	for _, permission := range []string{"conversations:update_team_assignee", "conversations:update_user_assignee", "conversations:update_status"} {
		if slices.Contains(employee.Permissions, permission) {
			t.Fatalf("return introduced broad permission: %s", permission)
		}
	}
}
