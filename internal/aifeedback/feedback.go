// Package aifeedback implements the trusted backend-to-n8n feedback contract.
package aifeedback

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/google/uuid"
)

const OriginalKey = "ai_feedback_original_message_uuid"
const BodyKey = "ai_feedback_sent_body"
const SentAtKey = "ai_feedback_sent_at"

var ErrInvalid = errors.New("invalid feedback message association")
var ErrNotSent = errors.New("reply has no confirmed successful delivery")

type Submission struct {
	ID                   string    `json:"id"`
	MailEventMessageUUID string    `json:"mail_event_message_uuid"`
	ConversationUUID     string    `json:"conversation_uuid"`
	OutgoingMessageUUID  string    `json:"outgoing_message_uuid"`
	AgentID              string    `json:"agent_id"`
	HumanAnswer          string    `json:"human_answer"`
	FeedbackReason       *string   `json:"feedback_reason"`
	FeedbackType         *string   `json:"feedback_type"`
	QualityStatus        string    `json:"quality_status"`
	ResponseSource       string    `json:"response_source"`
	SentAt               time.Time `json:"sent_at"`
}

func ValidateOriginal(original cmodels.Message, conversation string) error {
	if original.ConversationUUID != conversation || original.Private || original.Type != cmodels.MessageIncoming || original.SenderType != cmodels.SenderTypeContact {
		return ErrInvalid
	}
	if _, err := uuid.Parse(original.UUID); err != nil {
		return ErrInvalid
	}
	return nil
}

// Build trusts persisted messages and session identity, never a client answer or agent ID.
func Build(reply, original cmodels.Message, conversation string, agentID int, reason string) (Submission, error) {
	if reply.ConversationUUID != conversation || reply.SenderID != agentID || agentID <= 0 || reply.Private || reply.Type != cmodels.MessageOutgoing || reply.SenderType != cmodels.SenderTypeAgent {
		return Submission{}, ErrInvalid
	}
	if reply.Status != cmodels.MessageStatusSent {
		return Submission{}, ErrNotSent
	}
	if err := ValidateOriginal(original, conversation); err != nil {
		return Submission{}, err
	}
	var meta map[string]any
	if json.Unmarshal(reply.Meta, &meta) != nil || meta[OriginalKey] != original.UUID {
		return Submission{}, ErrInvalid
	}
	body, ok := meta[BodyKey].(string)
	if !ok || strings.TrimSpace(body) == "" {
		return Submission{}, ErrNotSent
	}
	outgoing, err := uuid.Parse(reply.UUID)
	if err != nil {
		return Submission{}, ErrInvalid
	}
	sentAtString, ok := meta[SentAtKey].(string)
	sentAt, timeErr := time.Parse(time.RFC3339Nano, sentAtString)
	if !ok || timeErr != nil {
		return Submission{}, ErrNotSent
	}
	reason = strings.TrimSpace(reason)
	if len(reason) > 8000 {
		return Submission{}, ErrInvalid
	}
	var optional *string
	if reason != "" {
		optional = &reason
	}
	return Submission{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("libredesk-ai-feedback:"+outgoing.String())).String(), MailEventMessageUUID: original.UUID, ConversationUUID: conversation, OutgoingMessageUUID: reply.UUID, AgentID: fmt.Sprint(agentID), HumanAnswer: body, FeedbackReason: optional, QualityStatus: "pending", ResponseSource: "human", SentAt: sentAt}, nil
}

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

// New preserves the existing HTTPS-only constructor.
func New(endpoint, token string) (*Client, error) {
	return NewWithConfig(Config{Endpoint: endpoint, Token: token})
}

// NewWithConfig returns nil only for a completely disabled configuration.
func NewWithConfig(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" && cfg.Token == "" && cfg.InternalHTTPEndpoint == "" && cfg.InternalHTTPIP == "" {
		return nil, nil
	}
	invalid := errors.New("invalid ai_feedback configuration: HTTPS or an exact internal HTTP URL and pinned private IP, plus a bearer token, are required")
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
	return &Client{endpoint: cfg.Endpoint, token: cfg.Token, http: &http.Client{Transport: transport, Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
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
			return nil, errors.New("feedback destination not permitted")
		}
		return connect(ctx, "tcp", target)
	}
}

// Submit performs one attempt. Only the receiver can guarantee transactional idempotency.
func (c *Client) Submit(ctx context.Context, submission Submission) error {
	body, err := json.Marshal(submission)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot create feedback request")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", submission.ID)
	res, err := c.http.Do(req)
	if err != nil {
		return errors.New("feedback receiver unavailable; retry feedback only")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		return fmt.Errorf("feedback receiver returned HTTP %d", res.StatusCode)
	}
	var receipt struct {
		ID        string `json:"id"`
		Committed bool   `json:"committed"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&receipt); err != nil || receipt.ID != submission.ID || !receipt.Committed {
		return errors.New("feedback receiver did not confirm commit")
	}
	return nil
}
