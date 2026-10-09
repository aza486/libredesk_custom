package main

import (
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"testing"
)

func TestAIFeedbackRequiresEmployeeSession(t *testing.T) {
	for _, method := range []string{"", authMethodAPIKey, authMethodPublic, authMethodSignedURL} {
		t.Run(method, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			ctx.SetUserValue("auth_method", method)
			req := &fastglue.Request{Context: &App{}, RequestCtx: ctx}
			if err := handleAIFeedback(req); err != nil {
				t.Fatal(err)
			}
			if ctx.Response.StatusCode() != fasthttp.StatusForbidden {
				t.Fatalf("got HTTP %d", ctx.Response.StatusCode())
			}
		})
	}
}
