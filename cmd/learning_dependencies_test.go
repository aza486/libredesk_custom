package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	"github.com/abhinavxd/libredesk/internal/learning"
	"github.com/lib/pq"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

func TestLearningDependencyDiagnosticsNeverExposeData(t *testing.T) {
	const secret = "CUSTOMER_TOKEN_PASSWORD_SENTINEL"
	for _, tt := range []struct {
		err  error
		want string
	}{
		{&learning.DependencyError{Service: "anonymizer", Kind: "not_configured"}, "anonymizer_not_configured"},
		{learningDependencyFailure("anonymizer", 504, aiassist.ErrTimeout), "anonymizer_timeout"},
		{learningDependencyFailure("anonymizer", 502, aiassist.ErrInvalidResponse), "anonymizer_invalid_response"},
		{learningDependencyFailure("anonymizer", 502, &aiassist.HTTPStatusError{Status: 401}), "anonymizer_http_status"},
		{learningDependencyFailure("anonymizer", 502, errors.New(secret)), "anonymizer_request_failed"},
		{&pq.Error{Code: "42501", Message: secret, Detail: secret, Hint: secret}, "42501"},
		{&pq.Error{Code: "42P01", Message: secret}, "42P01"},
		{context.DeadlineExceeded, "deadline_exceeded"},
		{aiassist.ErrInvalidConfiguration, "configuration_invalid"},
		{errors.New(secret), "internal_failure"},
		{&learning.DependencyError{Service: secret, Kind: secret}, "dependency_unavailable"},
	} {
		t.Run(tt.want, func(t *testing.T) {
			var output bytes.Buffer
			lo := logf.New(logf.Opts{Writer: &output})
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.SetMethod("POST")
			ctx.Request.SetBodyString(secret)
			r := &fastglue.Request{RequestCtx: ctx, Context: &App{lo: &lo}}
			if e := learningResult(r, nil, tt.err); e != nil {
				t.Fatal(e)
			}
			if ctx.Response.StatusCode() != 503 || !strings.Contains(output.String(), tt.want) {
				t.Fatal("missing safe diagnosis", ctx.Response.StatusCode(), output.String())
			}
			if strings.Contains(output.String(), secret) || strings.Contains(string(ctx.Response.Body()), secret) {
				t.Fatal("sensitive diagnostic exposed")
			}
		})
	}
}

func TestLearningLocalPrivacyDiagnosticKeepsHTTP400(t *testing.T) {
	const secret = "PRIVATE_CUSTOMER_SENTINEL@example.org"
	c := learning.Content{Type: "employee_fact", Category: "firmenfragen", Topic: "Versand", Retrieval: "Nachricht an " + secret}
	err := learning.Validate(c, learning.Checks{}, false)
	if !errors.Is(err, learning.ErrInput) {
		t.Fatal("not an input error", err)
	}
	var output bytes.Buffer
	lo := logf.New(logf.Opts{Writer: &output})
	ctx := &fasthttp.RequestCtx{}
	if e := learningResult(&fastglue.Request{RequestCtx: ctx, Context: &App{lo: &lo}}, nil, err); e != nil {
		t.Fatal(e)
	}
	if ctx.Response.StatusCode() != 400 || !strings.Contains(output.String(), "privacy_email") || strings.Contains(output.String(), secret) || strings.Contains(string(ctx.Response.Body()), secret) {
		t.Fatal("invalid privacy mapping/logging")
	}
}
