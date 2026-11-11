package main

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestServiceEmailManagementPermission(t *testing.T) {
	for _, tc := range []struct {
		user umodels.User
		want bool
	}{
		{umodels.User{Roles: []string{"Admin"}}, true},
		{umodels.User{Roles: []string{"Kundensupport"}}, false},
		{umodels.User{Permissions: []string{"conversations:read"}}, false},
		{umodels.User{Permissions: []string{"service_emails:manage"}}, true},
		{umodels.User{Permissions: []string{"general_settings:manage"}}, true},
	} {
		if got := canManageServiceEmails(tc.user); got != tc.want {
			t.Fatalf("%+v: %v", tc.user, got)
		}
	}
}

func TestServiceEmailManagementHTTP(t *testing.T) {
	db := testutil.NewPersonalDB(t, "service_email_http")
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
	app := &App{user: users, conversation: conv, i18n: lang, lo: &lo}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Service operator',ARRAY['service_emails:manage']),('Service reader',ARRAY['conversations:read']);
 INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Service operator';
 INSERT INTO user_roles(user_id,role_id) SELECT 102,id FROM roles WHERE name='Service reader';
 INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin';`)
	keys := map[int]string{}
	for _, id := range []int{102, 103, 104} {
		key, secret, err := users.GenerateAPIKey(id)
		if err != nil {
			t.Fatal(err)
		}
		keys[id] = "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":"+secret))
	}
	call := func(handler fastglue.FastRequestHandler, id int, method, body string, want int) *fasthttp.RequestCtx {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(method)
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.Header.Set("Authorization", keys[id])
		ctx.Request.SetBodyString(body)
		ctx.Request.URI().SetQueryString("address=contact%40example.com")
		if err := serviceEmailManagement(handler)(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != want {
			t.Fatalf("%s user %d: %d %s", method, id, ctx.Response.StatusCode(), ctx.Response.Body())
		}
		return ctx
	}
	call(handleAddServiceEmailAddress, 102, "POST", `{"address":"contact@example.com"}`, 403)
	call(handleRemoveServiceEmailAddress, 102, "DELETE", "", 403)
	call(handleGetServiceEmailAddresses, 102, "GET", "", 403)
	call(handleAddServiceEmailAddress, 103, "POST", `{"address":" CONTACT@EXAMPLE.COM "}`, 200)
	ctx := call(handleGetServiceEmailAddresses, 104, "GET", "", 200)
	var response struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &response); err != nil || len(response.Data) != 1 || response.Data[0] != "contact@example.com" {
		t.Fatalf("admin and sidebar diverged: %v %v", response, err)
	}
	call(handleAddServiceEmailAddress, 104, "POST", `{"address":"contact@example.com"}`, 400)
	call(handleAddServiceEmailAddress, 103, "POST", `{"address":"Name <new@example.com>"}`, 400)
	call(handleAddServiceEmailAddress, 103, "POST", `{"address":""}`, 400)
	call(handleRemoveServiceEmailAddress, 103, "DELETE", "", 200)
	addresses, err := conv.GetServiceEmailAddresses()
	if err != nil || len(addresses) != 0 {
		t.Fatalf("remove: %v %v", addresses, err)
	}
	db.MustExec(`UPDATE roles SET permissions='{}' WHERE name='Service operator'`)
	call(handleAddServiceEmailAddress, 103, "POST", `{"address":"new@example.com"}`, 403)
}
