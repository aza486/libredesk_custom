package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"strings"
	"testing"
)

func TestManualReasonDecode(t *testing.T) {
	for _, body := range []string{`{"feedback_reason":"  plain <b>text</b>  "}`, `{"feedback_reason":null}`, `{"feedback_reason":""}`} {
		if _, e := decodeManualReason([]byte(body)); e != nil {
			t.Fatal(e)
		}
	}
	for _, body := range []string{`null`, `{} {}`, `{"agent_id":101}`, `{"kind":"human"}`, `{"original_message_uuid":"fake"}`, `{"feedback_reason":true}`, `{"feedback_reason":"` + strings.Repeat("x", 2501) + `"}`, string([]byte{0xff})} {
		if _, e := decodeManualReason([]byte(body)); e == nil {
			t.Fatal("forged/invalid input accepted")
		}
	}
}
func TestManualReasonSessionBoundary(t *testing.T) {
	for _, method := range []string{"", authMethodAPIKey} {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("auth_method", method)
		ctx.SetUserValue("user", amodels.User{ID: 101})
		if e := handleManualReviewReason(&fastglue.Request{RequestCtx: ctx, Context: &App{}}); e != nil {
			t.Fatal(e)
		}
		if ctx.Response.StatusCode() != 403 {
			t.Fatal("non-session annotation allowed")
		}
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue("auth_method", authMethodSession)
	if e := handleManualReviewReason(&fastglue.Request{RequestCtx: ctx, Context: &App{}}); e != nil || ctx.Response.StatusCode() != 403 {
		t.Fatal("anonymous accepted")
	}
}
