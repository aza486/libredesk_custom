// Package aiassist provides isolated text-only and provenance webhook clients.
package aiassist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/abhinavxd/libredesk/internal/ssrf"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Client struct {
	endpoint, token string
	http            *http.Client
}

// Config is deployment-owned. Internal HTTP needs both an exact URL and one pinned IP.
type Config struct {
	Endpoint             string
	Token                string
	InternalHTTPEndpoint string
	InternalHTTPIP       string
	DialControl          ssrf.Control
}

// New constructs an HTTPS webhook client.
func New(endpoint, token string) (*Client, error) {
	return NewWithConfig(Config{Endpoint: endpoint, Token: token})
}

// NewWithConfig returns nil only for a completely disabled configuration.
func NewWithConfig(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" && cfg.Token == "" && cfg.InternalHTTPEndpoint == "" && cfg.InternalHTTPIP == "" {
		return nil, nil
	}
	invalid := errors.New("invalid ai webhook configuration: HTTPS or an exact internal HTTP URL and pinned private IP, plus a token, are required")
	u, err := url.Parse(cfg.Endpoint)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(cfg.Endpoint, "\\\r\n") || strings.TrimSpace(cfg.Token) == "" || strings.ContainsAny(cfg.Token, "\r\n") {
		return nil, invalid
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nil, invalid
		}
	}
	transport := ssrf.NewTransport(cfg.DialControl, 5*time.Second)
	// A proxy would move destination enforcement away from this backend.
	transport.Proxy = nil
	switch u.Scheme {
	case "https":
		if cfg.InternalHTTPEndpoint != "" || cfg.InternalHTTPIP != "" {
			return nil, invalid
		}
	case "http":
		if cfg.Endpoint != cfg.InternalHTTPEndpoint || cfg.Endpoint != u.String() || !validInternalURL(u) {
			return nil, invalid
		}
		ip, err := netip.ParseAddr(cfg.InternalHTTPIP)
		if err != nil || !ip.IsPrivate() || ip.Is4In6() || ip.Zone() != "" {
			return nil, invalid
		}
		if hostIP, err := netip.ParseAddr(u.Hostname()); err == nil && hostIP != ip {
			return nil, invalid
		}
		transport.DialContext = pinnedDialer(u.Host, ip, u.Port())
	default:
		return nil, invalid
	}
	return &Client{endpoint: cfg.Endpoint, token: cfg.Token, http: &http.Client{Transport: transport, Timeout: 45 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

// HTTP targets must have an explicit canonical port and unambiguous absolute path.
// Queries, escaped paths, dot segments and hostname aliases are deliberately rejected.
func validInternalURL(u *url.URL) bool {
	n, err := strconv.Atoi(u.Port())
	if err != nil || n < 1 || n > 65535 || strconv.Itoa(n) != u.Port() {
		return false
	}
	if u.RawQuery != "" || u.ForceQuery || u.RawPath != "" || u.EscapedPath() != u.Path || u.Path == "" || !strings.HasPrefix(u.Path, "/") || path.Clean(u.Path) != u.Path || strings.Contains(u.Path, "//") {
		return false
	}
	host := u.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		return ip.Zone() == ""
	}
	if host != strings.ToLower(host) || len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-') {
				return false
			}
		}
	}
	return true
}

// Never resolve the hostname: enforce the logical authority and connect to one
// configured IP literal. The shared guard gets only a single-address exception.
func pinnedDialer(authority string, ip netip.Addr, port string) func(context.Context, string, string) (net.Conn, error) {
	guard := ssrf.NewControl(true, []string{netip.PrefixFrom(ip, ip.BitLen()).String()}, nil)
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second, Control: guard}
	target := net.JoinHostPort(ip.String(), port)
	return pinDestination(authority, target, dialer.DialContext)
}

func pinDestination(authority, target string, connect func(context.Context, string, string) (net.Conn, error)) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" || address != authority {
			return nil, errors.New("webhook destination not permitted")
		}
		return connect(ctx, "tcp", target)
	}
}

// Call never logs request data or tokens and never follows redirects.
func (c *Client) Call(ctx context.Context, header, id string, payload, result any) (int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 502, errors.New("invalid webhook request")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return 502, errors.New("webhook unavailable")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(header, c.token)
	req.Header.Set("Idempotency-Key", id)
	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return 504, errors.New("webhook timeout")
		}
		var e net.Error
		if errors.As(err, &e) && e.Timeout() {
			return 504, errors.New("webhook timeout")
		}
		return 502, errors.New("webhook unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if res.StatusCode == 400 || res.StatusCode == 422 {
			return res.StatusCode, errors.New("webhook rejected text")
		}
		return 502, errors.New("webhook unavailable")
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 131073))
	if err != nil || len(raw) > 131072 || !utf8.Valid(raw) {
		return 502, errors.New("invalid webhook response")
	}
	if json.Unmarshal(raw, result) != nil {
		return 502, errors.New("invalid webhook response")
	}
	return 200, nil
}

type StyleRequest struct {
	RequestID    string `json:"request_id"`
	AgentID      string `json:"agent_id"`
	Style        string `json:"style"`
	Instructions string `json:"instructions"`
	OriginalText string `json:"original_text"`
	CurrentText  string `json:"current_text"`
}
type StyleResponse struct {
	Success       bool   `json:"success"`
	RequestID     string `json:"request_id"`
	AdaptedText   string `json:"adapted_text"`
	ProfileSource string `json:"profile_source"`
}

func ValidText(s string, max int) bool {
	return utf8.ValidString(s) && strings.TrimSpace(s) != "" && !strings.ContainsRune(s, 0) && utf8.RuneCountInString(s) <= max
}
func (r StyleRequest) Validate() bool {
	switch r.Style {
	case "mine", "friendly", "professional", "empathetic", "short", "neutral":
	default:
		return false
	}
	return ValidText(r.OriginalText, 8000) && ValidText(r.CurrentText, 8000) && utf8.ValidString(r.Instructions) && utf8.RuneCountInString(r.Instructions) <= 1000 && !strings.ContainsRune(r.Instructions, 0)
}
func (r StyleResponse) Valid(id string) bool {
	return r.Success && r.RequestID == id && ValidText(r.AdaptedText, 8000) && (r.ProfileSource == "agent" || r.ProfileSource == "fallback" || r.ProfileSource == "preset")
}
