package aifeedback

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ssrf"
)

func internalConfig() Config {
	return Config{Endpoint: "http://n8n:5678/webhook/ai-feedback", Token: "test-token", InternalHTTPEndpoint: "http://n8n:5678/webhook/ai-feedback", InternalHTTPIP: "172.18.0.5"}
}

func TestEndpointConfiguration(t *testing.T) {
	cases := []struct {
		name    string
		change  func(*Config)
		allowed bool
	}{
		{"explicit internal HTTP", func(c *Config) {}, true},
		{"HTTPS backwards compatible", func(c *Config) { *c = Config{Endpoint: "https://example.com/webhook?key=value", Token: "test"} }, true},
		{"internal HTTPS", func(c *Config) { *c = Config{Endpoint: "https://n8n:5678/webhook/ai-feedback", Token: "test"} }, true},
		{"HTTP requires exact grant", func(c *Config) { c.InternalHTTPEndpoint = "" }, false},
		{"HTTP requires pinned IP", func(c *Config) { c.InternalHTTPIP = "" }, false},
		{"missing token", func(c *Config) { c.Token = "" }, false},
		{"blank token", func(c *Config) { c.Token = "  " }, false},
		{"token newline", func(c *Config) { c.Token = "test\n" }, false},
		{"different scheme", func(c *Config) { c.Endpoint = "https://n8n:5678/webhook/ai-feedback" }, false},
		{"different host", func(c *Config) { c.Endpoint = "http://redis:5678/webhook/ai-feedback" }, false},
		{"host suffix", func(c *Config) { c.Endpoint = "http://n8n.attacker:5678/webhook/ai-feedback" }, false},
		{"different port", func(c *Config) { c.Endpoint = "http://n8n:6379/webhook/ai-feedback" }, false},
		{"different path", func(c *Config) { c.Endpoint = "http://n8n:5678/rest/settings" }, false},
		{"no port", func(c *Config) { c.Endpoint = "http://n8n/webhook/ai-feedback"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"zero port", func(c *Config) { c.Endpoint = "http://n8n:0/webhook/ai-feedback"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"out of range port", func(c *Config) {
			c.Endpoint = "http://n8n:65536/webhook/ai-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"noncanonical port", func(c *Config) {
			c.Endpoint = "http://n8n:05678/webhook/ai-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"path traversal", func(c *Config) { c.Endpoint = "http://n8n:5678/webhook/../rest"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"encoded path", func(c *Config) {
			c.Endpoint = "http://n8n:5678/webhook/%61i-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"double encoded path", func(c *Config) {
			c.Endpoint = "http://n8n:5678/webhook/%252e%252e/rest"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"query", func(c *Config) { c.Endpoint += "?next=http://redis:6379"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"empty query", func(c *Config) { c.Endpoint += "?"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"fragment", func(c *Config) { c.Endpoint += "#next"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"empty fragment", func(c *Config) { c.Endpoint += "#"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"userinfo", func(c *Config) {
			c.Endpoint = "http://user:password@n8n:5678/webhook/ai-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"backslash", func(c *Config) { c.Endpoint = "http://n8n:5678/webhook\\rest"; c.InternalHTTPEndpoint = c.Endpoint }, false},
		{"trailing DNS dot", func(c *Config) {
			c.Endpoint = "http://n8n.:5678/webhook/ai-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"CIDR grant", func(c *Config) { c.InternalHTTPIP = "172.18.0.0/16" }, false},
		{"loopback", func(c *Config) { c.InternalHTTPIP = "127.0.0.1" }, false},
		{"unspecified", func(c *Config) { c.InternalHTTPIP = "0.0.0.0" }, false},
		{"metadata", func(c *Config) { c.InternalHTTPIP = "169.254.169.254" }, false},
		{"public IP", func(c *Config) { c.InternalHTTPIP = "8.8.8.8" }, false},
		{"mapped IP", func(c *Config) { c.InternalHTTPIP = "::ffff:172.18.0.5" }, false},
		{"scoped IPv6", func(c *Config) { c.InternalHTTPIP = "fd00::5%eth0" }, false},
		{"private IPv6", func(c *Config) { c.InternalHTTPIP = "fd00::5" }, true},
		{"matching IP host", func(c *Config) {
			c.Endpoint = "http://172.18.0.5:5678/webhook/ai-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, true},
		{"different IP host", func(c *Config) {
			c.Endpoint = "http://172.18.0.6:5678/webhook/ai-feedback"
			c.InternalHTTPEndpoint = c.Endpoint
		}, false},
		{"HTTPS userinfo", func(c *Config) { *c = Config{Endpoint: "https://user:pass@example.com", Token: "test"} }, false},
		{"HTTPS fragment", func(c *Config) { *c = Config{Endpoint: "https://example.com/#fragment", Token: "test"} }, false},
		{"HTTPS invalid port", func(c *Config) { *c = Config{Endpoint: "https://example.com:65536/", Token: "test"} }, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			cfg := internalConfig()
			tt.change(&cfg)
			_, err := NewWithConfig(cfg)
			if (err == nil) != tt.allowed {
				t.Fatalf("unexpected result: %v", err)
			}
		})
	}
}

func TestPinnedDestination(t *testing.T) {
	calls := 0
	connect := func(ctx context.Context, network, address string) (net.Conn, error) {
		calls++
		if network != "tcp" || address != "172.18.0.5:5678" {
			t.Fatalf("unexpected physical target %s %s", network, address)
		}
		return nil, errors.New("test connection")
	}
	dial := pinDestination("n8n:5678", "172.18.0.5:5678", connect)
	for _, address := range []string{"redis:5678", "n8n:6379", "127.0.0.1:5678", "169.254.169.254:80", "172.18.0.6:5678"} {
		if _, err := dial(context.Background(), "tcp", address); err == nil {
			t.Fatal("foreign target accepted")
		}
	}
	if _, err := dial(context.Background(), "udp", "n8n:5678"); err == nil {
		t.Fatal("UDP accepted")
	}
	if calls != 0 {
		t.Fatal("rejected target reached dialer")
	}
	_, _ = dial(context.Background(), "tcp", "n8n:5678")
	if calls != 1 {
		t.Fatal("approved logical target did not dial pinned IP")
	}
	// The production dialer accepts a literal IP even when the logical hostname
	// cannot resolve: cancellation yields the pinned TCP target, not a DNS error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := pinnedDialer("unresolvable.invalid:5678", netip.MustParseAddr("172.18.0.5"), "5678")(ctx, "tcp", "unresolvable.invalid:5678")
	var op *net.OpError
	if !errors.As(err, &op) || op.Addr == nil || op.Addr.String() != "172.18.0.5:5678" {
		t.Fatalf("did not dial literal pinned address: %v", err)
	}
}

func TestInternalHTTPTransportAndRedirect(t *testing.T) {
	targetCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++ }))
	defer target.Close()
	reply, original := messages()
	submission, _ := Build(reply, original, "conversation", 42, "")
	for _, redirect := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Host != "n8n:5678" || r.URL.Path != "/webhook/ai-feedback" || r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("Idempotency-Key") != submission.ID {
				t.Error("invalid internal HTTP request")
			}
			if redirect {
				http.Redirect(w, r, target.URL, 307)
				return
			}
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"id":"` + submission.ID + `","committed":true}`))
		}))
		cfg := internalConfig()
		client, err := NewWithConfig(cfg)
		if err != nil {
			t.Fatal(err)
		}
		transport := client.http.Transport.(*http.Transport)
		if transport.Proxy != nil {
			t.Fatal("environment proxy enabled")
		}
		physical := server.Listener.Addr().String()
		// Substitute only the final socket connector; retain logical target enforcement.
		transport.DialContext = pinDestination("n8n:5678", "172.18.0.5:5678", func(ctx context.Context, network, address string) (net.Conn, error) {
			if address != "172.18.0.5:5678" {
				t.Fatal("wrong pin")
			}
			return (&net.Dialer{}).DialContext(ctx, network, physical)
		})
		err = client.Submit(context.Background(), submission)
		if (err != nil) != redirect {
			t.Fatalf("unexpected submit result: %v", err)
		}
		transport.CloseIdleConnections()
		server.Close()
	}
	if targetCalls != 0 {
		t.Fatal("redirect followed")
	}
}

func TestHTTPSUsesSharedGuard(t *testing.T) {
	cfg := Config{Endpoint: "https://127.0.0.1:1/feedback", Token: "test", DialControl: ssrf.NewControl(true, nil, nil)}
	client, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	transport := client.http.Transport.(*http.Transport)
	if transport.Proxy != nil {
		t.Fatal("HTTPS proxy bypass enabled")
	}
	_, err = transport.DialContext(context.Background(), "tcp", "127.0.0.1:1")
	if err == nil || !strings.Contains(err.Error(), "ssrf:") {
		t.Fatalf("shared guard did not reject loopback: %v", err)
	}
}
