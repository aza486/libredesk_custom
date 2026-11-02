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

func TestLearningAnonymizerConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name             string
		cfg              aiassist.Config
		enabled, invalid bool
	}{
		{name: "disabled"},
		{name: "internal only no token", cfg: aiassist.Config{InternalHTTPEndpoint: "http://anonymizer:8001/analyze", InternalHTTPIP: "172.18.0.7"}, enabled: true},
		{name: "explicit internal with token", cfg: aiassist.Config{Endpoint: "http://anonymizer:8001/analyze", InternalHTTPEndpoint: "http://anonymizer:8001/analyze", InternalHTTPIP: "172.18.0.7", Token: "Bearer synthetic"}, enabled: true},
		{name: "external authenticated HTTPS", cfg: aiassist.Config{Endpoint: "https://anonymizer.example/analyze", Token: "Bearer synthetic"}, enabled: true},
		{name: "HTTPS must authenticate", cfg: aiassist.Config{Endpoint: "https://anonymizer.example/analyze"}, invalid: true},
		{name: "internal missing pin", cfg: aiassist.Config{InternalHTTPEndpoint: "http://anonymizer:8001/analyze"}, invalid: true},
		{name: "pin only", cfg: aiassist.Config{InternalHTTPIP: "172.18.0.7"}, invalid: true},
		{name: "token only", cfg: aiassist.Config{Token: "Bearer synthetic"}, invalid: true},
		{name: "public pin", cfg: aiassist.Config{InternalHTTPEndpoint: "http://anonymizer:8001/analyze", InternalHTTPIP: "8.8.8.8"}, invalid: true},
		{name: "loopback pin", cfg: aiassist.Config{InternalHTTPEndpoint: "http://anonymizer:8001/analyze", InternalHTTPIP: "127.0.0.1"}, invalid: true},
		{name: "ambiguous endpoints", cfg: aiassist.Config{Endpoint: "https://anonymizer.example/analyze", InternalHTTPEndpoint: "http://anonymizer:8001/analyze", InternalHTTPIP: "172.18.0.7", Token: "Bearer synthetic"}, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			anon, err := newLearningAnonymizer(tt.cfg)
			if (err != nil) != tt.invalid || (anon != nil) != tt.enabled {
				t.Fatal("configuration contract", err, anon != nil)
			}
		})
	}
}
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
