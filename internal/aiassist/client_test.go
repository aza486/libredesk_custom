package aiassist

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestStyleValidation(t *testing.T) {
	r := StyleRequest{Style: "mine", OriginalText: "Hallo", CurrentText: "Hallo", Instructions: strings.Repeat("🙂", 1000)}
	if !r.Validate() {
		t.Fatal("valid unicode request rejected")
	}
	for _, bad := range []StyleRequest{{Style: "unknown", OriginalText: "x", CurrentText: "x"}, {Style: "mine", OriginalText: " ", CurrentText: "x"}, {Style: "mine", OriginalText: "x", CurrentText: strings.Repeat("a", 8001)}, {Style: "mine", OriginalText: "x", CurrentText: "x", Instructions: strings.Repeat("🙂", 1001)}} {
		if bad.Validate() {
			t.Fatal("invalid request accepted")
		}
	}
	result := StyleResponse{Success: true, RequestID: "id", AdaptedText: "Antwort", ProfileSource: "agent"}
	if !result.Valid("id") || result.Valid("other") {
		t.Fatal("request id validation")
	}
	result.ProfileSource = "other"
	if result.Valid("id") {
		t.Fatal("invalid profile")
	}
}
func TestWebhookContractAndErrors(t *testing.T) {
	for _, status := range []int{200, 400, 422, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client := &Client{endpoint: "https://n8n.example/webhook/style", token: "secret", http: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("X-Libredesk-Style-Token") != "secret" || r.Header.Get("Idempotency-Key") != "id" {
					t.Fatal("missing protected headers")
				}
				raw, _ := io.ReadAll(r.Body)
				if strings.Contains(string(raw), "secret") {
					t.Fatal("credential in payload")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{"success":true,"request_id":"id","adapted_text":"Hallo","profile_source":"preset"}`))}, nil
			})}}
			var result StyleResponse
			got, err := client.Call(context.Background(), "X-Libredesk-Style-Token", "id", StyleRequest{}, &result)
			if status == 200 {
				if err != nil || !result.Valid("id") {
					t.Fatal(got, err)
				}
			} else if err == nil || got == 200 {
				t.Fatal("accepted rejection")
			}
		})
	}
	for _, body := range []string{`{}`, `[]`, strings.Repeat("x", 131073), `{"success":true}garbage`} {
		client := &Client{endpoint: "https://example.org", http: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}}
		var result StyleResponse
		_, err := client.Call(context.Background(), "token", "id", nil, &result)
		if err == nil && result.Valid("id") {
			t.Fatal("invalid output accepted")
		}
	}
	client := &Client{endpoint: "https://example.org", http: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { return nil, errors.New("secret upstream failure") })}}
	var result any
	if status, err := client.Call(context.Background(), "token", "id", nil, &result); status != 502 || strings.Contains(err.Error(), "secret") {
		t.Fatal("unsafe network error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()
	if status, _ := client.Call(ctx, "token", "id", nil, &result); status != 504 {
		t.Fatal("timeout status")
	}
}
func TestWebhookConfiguration(t *testing.T) {
	if c, e := NewWithConfig(Config{}); c != nil || e != nil {
		t.Fatal("disabled config")
	}
	for _, cfg := range []Config{{Endpoint: "http://localhost:5678/webhook", Token: "secret"}, {Endpoint: "https://example.com", Token: "bad\nheader"}, {Endpoint: "http://n8n:5678/webhook", Token: "secret", InternalHTTPEndpoint: "http://n8n:5678/webhook", InternalHTTPIP: "127.0.0.1"}} {
		if _, err := NewWithConfig(cfg); err == nil {
			t.Fatal("unsafe config")
		}
	}
	if _, err := NewWithConfig(Config{Endpoint: "http://n8n:5678/webhook", Token: "secret", InternalHTTPEndpoint: "http://n8n:5678/webhook", InternalHTTPIP: "172.20.0.5"}); err != nil {
		t.Fatal(err)
	}
}

func TestUnauthenticatedInternalServiceIsExplicitAndPinned(t *testing.T) {
	base := Config{Endpoint: "http://anonymizer:8001/analyze", InternalHTTPEndpoint: "http://anonymizer:8001/analyze", InternalHTTPIP: "172.18.0.7"}
	if _, e := NewWithConfig(base); e == nil {
		t.Fatal("other clients lost token requirement")
	}
	base.AllowUnauthenticatedInternalHTTP = true
	client, e := NewWithConfig(base)
	if e != nil || client == nil {
		t.Fatal(e)
	}
	transport := client.http.Transport.(*http.Transport)
	if transport.Proxy != nil {
		t.Fatal("proxy bypass")
	}
	if e = client.http.CheckRedirect(nil, nil); !errors.Is(e, http.ErrUseLastResponse) {
		t.Fatal("redirect bypass")
	}
	// The only actual HTTP exchange in this test is a synthetic RoundTripper.
	client.http.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		if _, ok := r.Header["Authorization"]; ok {
			t.Fatal("empty/fabricated Authorization header")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"clean_text":"Safe text"}`))}, nil
	})
	var result struct {
		Clean string `json:"clean_text"`
	}
	if status, e := client.Call(context.Background(), "Authorization", "", map[string]string{"text": "Synthetic"}, &result); e != nil || status != 200 || result.Clean != "Safe text" {
		t.Fatal(status, e)
	}
	for _, mutate := range []func(*Config){
		func(c *Config) {
			c.Endpoint = "https://example.org/analyze"
			c.InternalHTTPEndpoint = ""
			c.InternalHTTPIP = ""
		},
		func(c *Config) { c.InternalHTTPIP = "" }, func(c *Config) { c.InternalHTTPIP = "127.0.0.1" },
		func(c *Config) { c.InternalHTTPIP = "169.254.169.254" }, func(c *Config) { c.InternalHTTPIP = "8.8.8.8" },
		func(c *Config) { c.InternalHTTPIP = "::ffff:172.18.0.7" },
		func(c *Config) { c.Endpoint = "http://172.18.0.8:8001/analyze"; c.InternalHTTPEndpoint = c.Endpoint },
		func(c *Config) { c.Endpoint += "?next=http://example.org"; c.InternalHTTPEndpoint = c.Endpoint },
		func(c *Config) {
			c.Endpoint = "http://anonymizer:8001/a/../analyze"
			c.InternalHTTPEndpoint = c.Endpoint
		},
		func(c *Config) { c.Token = "bad\nheader" }, func(c *Config) { c.Token = " " },
	} {
		cfg := base
		mutate(&cfg)
		if _, e := NewWithConfig(cfg); e == nil {
			t.Fatal("unsafe override accepted")
		}
	}
}
func TestPinnedInternalDialerDoesNotResolveOrRetarget(t *testing.T) {
	called := false
	dial := pinDestination("anonymizer:8001", "172.18.0.7:8001", func(_ context.Context, network, address string) (net.Conn, error) {
		called = true
		if network != "tcp" || address != "172.18.0.7:8001" {
			t.Fatal("pin bypass")
		}
		return nil, nil
	})
	if _, e := dial(context.Background(), "tcp", "anonymizer:8001"); e != nil || !called {
		t.Fatal(e)
	}
	for _, authority := range []string{"example.org:8001", "172.18.0.8:8001", "anonymizer:80"} {
		called = false
		if _, e := dial(context.Background(), "tcp", authority); e == nil || called {
			t.Fatal("authority bypass")
		}
	}
}
func TestWebhookErrorsAreTypedAndDoNotExposeUpstreamBodies(t *testing.T) {
	client := &Client{endpoint: "https://example.org", http: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("SECRET_RESPONSE_BODY"))}, nil
	})}}
	status, e := client.Call(context.Background(), "Authorization", "", nil, new(any))
	var upstream *HTTPStatusError
	if status != 502 || !errors.As(e, &upstream) || upstream.Status != 401 || strings.Contains(e.Error(), "SECRET") {
		t.Fatal("unsafe upstream error")
	}
}
