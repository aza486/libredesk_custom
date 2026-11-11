package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestWorkPhoneSelfProfileHTTP(t *testing.T) {
	db := testutil.NewPersonalDB(t, "work_phone_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, _ := authz.NewEnforcer(&lo, lang)
	app := &App{user: users, i18n: lang, lo: &lo, authz: enforcer}
	key, secret, err := users.GenerateAPIKey(102)
	if err != nil {
		t.Fatal(err)
	}
	request := func(handler fastglue.FastRequestHandler, values map[string]string, method string) *fasthttp.RequestCtx {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("id", "101") // Even a forged foreign target must not affect self updates.
		ctx.Request.Header.SetMethod(method)
		ctx.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(key+":"+secret)))
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for name, value := range values {
			if err := writer.WriteField(name, value); err != nil {
				t.Fatal(err)
			}
		}
		writer.Close()
		ctx.Request.Header.SetContentType(writer.FormDataContentType())
		ctx.Request.SetBody(body.Bytes())
		if err := handler(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(ctx.Request.RemoveMultipartFormFiles)
		return ctx
	}
	for _, value := range []string{" 0351 40424-23 ", "+49 (351) 40424-23", ""} {
		ctx := request(auth(handleUpdateCurrentAgent), map[string]string{"work_phone_number": value, "id": "101", "user_id": "101", "roles": "Admin"}, "PUT")
		if ctx.Response.StatusCode() != 200 {
			t.Fatalf("save: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
		}
		ctx = request(auth(handleGetCurrentAgent), nil, "GET")
		var profile struct {
			Data struct {
				ID    int     `json:"id"`
				Phone *string `json:"work_phone_number"`
			} `json:"data"`
		}
		if err := json.Unmarshal(ctx.Response.Body(), &profile); err != nil {
			t.Fatal(err)
		}
		if profile.Data.ID != 102 {
			t.Fatal("wrong profile")
		}
		if value == "" {
			if profile.Data.Phone != nil {
				t.Fatal("empty not null")
			}
		} else if profile.Data.Phone == nil || *profile.Data.Phone != strings.TrimSpace(value) {
			t.Fatal("phone not restored")
		}
		other, err := users.GetAgent(101, "")
		if err != nil || other.WorkPhoneNumber.Valid {
			t.Fatal("foreign user modified", err)
		}
	}
	ctx := request(auth(handleUpdateCurrentAgent), map[string]string{"work_phone_number": "<b>123</b>"}, "PUT")
	if ctx.Response.StatusCode() != 400 {
		t.Fatal("HTML accepted", string(ctx.Response.Body()))
	}
	if err := users.UpdateWorkPhone(102, "030 12345678"); err != nil {
		t.Fatal(err)
	}
	ctx = request(auth(handleUpdateCurrentAgent), map[string]string{"unrelated": "value"}, "PUT")
	if ctx.Response.StatusCode() != 200 || !strings.Contains(string(ctx.Response.Body()), "030 12345678") {
		t.Fatal("omitted field erased phone")
	}
	ctx = request(perm(handleUpdateAgent, "users:manage"), nil, "PUT")
	if ctx.Response.StatusCode() != 403 {
		t.Fatal("foreign profile access allowed")
	}
}
