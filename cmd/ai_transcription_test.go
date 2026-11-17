package main

import (
	"bytes"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestTranscriptionInputAndCurrentAgent(t *testing.T) {
	db := testutil.NewPersonalDB(t, "transcription_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{user: users, i18n: lang, lo: &lo}
	request := func(typ string, data []byte) *fasthttp.RequestCtx {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("auth_method", authMethodSession)
		ctx.SetUserValue("user", amodels.User{ID: 102})
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.Header.SetContentType(typ)
		ctx.Request.SetBody(data)
		if err := handleAITranscribe(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		return ctx
	}
	if r := request("audio/ogg", []byte("OggSsynthetic audio")); r.Response.StatusCode() != 503 {
		t.Fatalf("unconfigured: %d", r.Response.StatusCode())
	}
	if r := request("audio/ogg", []byte("<html>invalid audio</html>")); r.Response.StatusCode() != 400 {
		t.Fatal("bad audio accepted")
	}
	if r := request("audio/ogg", bytes.Repeat([]byte{'x'}, (8<<20)+1)); r.Response.StatusCode() != 413 {
		t.Fatal("oversize accepted")
	}
	db.MustExec(`UPDATE users SET enabled=false WHERE id=102`)
	if r := request("audio/ogg", []byte("OggSsynthetic audio")); r.Response.StatusCode() != 403 {
		t.Fatal("disabled agent accepted")
	}
}
