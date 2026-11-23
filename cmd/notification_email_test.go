package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	inboxemail "github.com/abhinavxd/libredesk/internal/inbox/channel/email"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/setting"
	smodels "github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func newNotificationHTTP(t *testing.T, name string) *tagSecurityHTTP {
	t.Helper()
	app, db := systemAttributesHTTPFixture(t, name)
	var err error
	app.setting, err = setting.New(setting.Opts{DB: db, Lo: app.lo, EncryptionKey: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO inboxes(id,name,channel,"from") VALUES(3087,'System mail','email','system@example.test'),(4087,'Replacement','email','replacement@example.test')`)
	g := fastglue.NewGlue()
	g.SetContext(app)
	g.GET("/api/v1/settings/notifications/email", perm(handleGetEmailNotificationSettings, "notification_settings:manage"))
	g.PUT("/api/v1/settings/notifications/email", perm(handleUpdateEmailNotificationSettings, "notification_settings:manage"))
	g.POST("/api/v1/conversations", perm(handleCreateConversation, "conversations:write"))
	g.POST("/api/v1/conversations/{cuuid}/messages", perm(handleSendMessage, "messages:write"))
	g.GET("/api/v1/conversations/{uuid}/messages", perm(handleGetMessages, "conversations:read"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	dir, err := os.MkdirTemp("/tmp", "ld-email-http-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "http.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &fasthttp.Server{Handler: g.Handler()}
	go func() { _ = server.Serve(listener) }()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	t.Cleanup(func() { transport.CloseIdleConnections(); _ = server.Shutdown(); _ = listener.Close() })
	f := &tagSecurityHTTP{app: app, db: db, client: &http.Client{Transport: transport, Timeout: 10 * time.Second}, keys: map[int][2]string{}}
	f.keys[0] = [2]string{"invalid-test-key", "invalid-test-secret"}
	for _, id := range []int{101, 102, 103, 104} {
		key, secret, err := app.user.GenerateAPIKey(id)
		if err != nil {
			t.Fatal(err)
		}
		f.keys[id] = [2]string{key, secret}
	}
	return f
}
func notificationChoice(t *testing.T, f *tagSecurityHTTP, id int, contact string) string {
	t.Helper()
	var uuid string
	if id != 0 {
		if err := f.db.Get(&uuid, `SELECT uuid FROM inboxes WHERE id=$1`, id); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := json.Marshal(map[string]any{"notification.email.inbox_uuid": uuid, "notification.email.team_contact_email": contact, "notification.email.enabled": true, "notification.email.email_address": "legacy@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
func TestNotificationInternalInboxHTTP(t *testing.T) {
	f := newNotificationHTTP(t, "notification_shared_http")
	if err := f.app.setting.UpdateEmailNotification(smodels.EmailNotification{Host: "preserved-smtp.example.test", Username: "preserved-user", Password: "synthetic-test-password", Port: 587, EmailAddress: "legacy@example.test"}, func(*smodels.EmailNotification) error { return nil }); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/settings/notifications/email"
	for _, actor := range []int{0, 101, 103} {
		want := 403
		if actor == 0 {
			want = 401
		}
		f.request(t, actor, "PUT", path, notificationChoice(t, f, 3087, "team@example.test"), want)
		f.request(t, actor, "GET", path, "", want)
	}
	f.request(t, 104, "PUT", path, notificationChoice(t, f, 3087, "team@example.test"), 200)
	raw := f.request(t, 104, "GET", path, "", 200)
	if !strings.Contains(string(raw), "preserved-smtp.example.test") || !strings.Contains(string(raw), "preserved-user") || strings.Contains(string(raw), "synthetic-test-password") {
		t.Fatal("routing update lost legacy SMTP settings or password masking")
	}
	if !strings.Contains(string(raw), "System mail") || !strings.Contains(string(raw), "system@example.test") {
		t.Fatal("selection not reloaded", string(raw))
	}
	for _, tc := range []struct {
		agent, team int
		recipient   string
	}{{102, 0, "other@example.com"}, {102, 201, "other@example.com"}, {0, 201, ""}} {
		t.Run(fmt.Sprintf("agent_%d_team_%d", tc.agent, tc.team), func(t *testing.T) {
			response := f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, tc.agent, tc.team), 200)
			var data struct {
				Data cmodels.Conversation `json:"data"`
			}
			if err := json.Unmarshal(response, &data); err != nil {
				t.Fatal(err)
			}
			conv, err := f.app.conversation.GetConversation(0, data.Data.UUID, "")
			if err != nil {
				t.Fatal(err)
			}
			if conv.InboxID != 3087 || conv.Contact.Email.String != tc.recipient || conv.AssignedTeamID.Int != tc.team || conv.AssignedUserID.Int != tc.agent {
				t.Fatalf("routing changed: %+v", conv)
			}
			var attrs map[string]any
			if err := json.Unmarshal(conv.CustomAttributes, &attrs); err != nil {
				t.Fatal(err)
			}
			if attrs["private"] != true || attrs["internal"] != true || attrs["creator_id"] != float64(101) {
				t.Fatal(attrs)
			}
			for actor, want := range map[int]int{101: 200, 103: 403, 104: 200} {
				f.request(t, actor, "GET", "/api/v1/conversations/"+conv.UUID, "", want)
			}
		})
	}
	// A different selection is effective without rebuilding App or restarting workers.
	f.request(t, 104, "PUT", path, notificationChoice(t, f, 4087, "new-team@example.test"), 200)
	req := createConversationRequest{AssignedAgentID: 102, CustomAttributes: map[string]any{"internal": true, "private": true}}
	if err := prepareInternalConversation(&req, f.app, 101); err != nil || req.InboxID != 4087 {
		t.Fatal(req.InboxID, err)
	}
	f.db.MustExec(`UPDATE inboxes SET name='Renamed' WHERE id=4087`)
	req.InboxID = 0
	if err := prepareInternalConversation(&req, f.app, 101); err != nil || req.InboxID != 4087 {
		t.Fatal(err)
	}
	if f.app.restartRequired {
		t.Fatal("configuration still requires restart")
	}
	// Old clients cannot clear the chosen UUID merely by omitting new fields.
	f.request(t, 104, "PUT", path, `{"notification.email.email_address":"old-client@example.test"}`, 200)
	cfg, err := loadNotificationEmail(f.app.setting)
	if err != nil || cfg.InboxUUID == nil || cfg.EmailAddress != "replacement@example.test" {
		t.Fatal(cfg.EmailAddress, err)
	}
}

func TestNotificationInboxInvalidationHTTP(t *testing.T) {
	for i, change := range []string{
		`UPDATE inboxes SET enabled=false WHERE id=3087`,
		`UPDATE inboxes SET channel='livechat' WHERE id=3087`,
		`UPDATE inboxes SET access_mode='personal',owner_user_id=101 WHERE id=3087`,
		`UPDATE inboxes SET deleted_at=NOW() WHERE id=3087; INSERT INTO inboxes(id,name,channel,"from") VALUES(5087,'Recreated','email','system@example.test')`,
		`INSERT INTO inboxes(id,name,channel,"from") VALUES(5087,'Duplicate','email','SYSTEM@example.test')`,
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			f := newNotificationHTTP(t, fmt.Sprintf("notification_invalid_%d", i))
			path := "/api/v1/settings/notifications/email"
			choice := notificationChoice(t, f, 3087, "team@example.test")
			f.request(t, 104, "PUT", path, choice, 200)
			f.db.MustExec(change)
			f.request(t, 104, "PUT", path, choice, 400)
			f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 102, 0), 400)
			// Invalid stored choices stay visible as configured and can be repaired; no fallback.
			f.request(t, 104, "GET", path, "", 200)
		})
	}
}
func TestNotificationLegacyFallbackAndRecipients(t *testing.T) {
	f := newNotificationHTTP(t, "notification_legacy")
	path := "/api/v1/settings/notifications/email"
	req := func() createConversationRequest {
		return createConversationRequest{AssignedAgentID: 102, CustomAttributes: map[string]any{"internal": true, "private": true}}
	}
	r := req()
	if err := prepareInternalConversation(&r, f.app, 101); err != nil || r.InboxID != 102 {
		t.Fatal("legacy UUID stopped working", err)
	}
	f.app.internalConversation.InboxUUID = ""
	f.db.MustExec(`UPDATE settings SET value='"system@example.test"' WHERE key='notification.email.email_address'`)
	r = req()
	if err := prepareInternalConversation(&r, f.app, 101); err != nil || r.InboxID != 3087 {
		t.Fatal("existing address not resolved", err)
	}
	f.request(t, 104, "PUT", path, notificationChoice(t, f, 3087, "SYSTEM@example.test"), 400)
	f.request(t, 104, "PUT", path, notificationChoice(t, f, 3087, "invalid"), 400)
	f.request(t, 104, "PUT", path, notificationChoice(t, f, 3087, ""), 200)
	f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 0, 201), 200)
	f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 102, 201), 200)
	f.db.MustExec(`UPDATE users SET email='system@example.test' WHERE id=102`)
	f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 102, 0), 400)
	f.request(t, 104, "PUT", path, notificationChoice(t, f, 0, ""), 200)
	f.app.internalConversation.InboxUUID = "bad-old-uuid"
	r = req()
	if err := prepareInternalConversation(&r, f.app, 101); err == nil {
		t.Fatal("explicit empty admin choice fell back")
	}
	f.db.MustExec(`DELETE FROM inboxes`)
	f.request(t, 104, "GET", path, "", 200)
}
func TestNotificationConcurrentSettings(t *testing.T) {
	f := newNotificationHTTP(t, "notification_concurrent")
	var one, two string
	f.db.Get(&one, `SELECT uuid FROM inboxes WHERE id=3087`)
	f.db.Get(&two, `SELECT uuid FROM inboxes WHERE id=4087`)
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, contact := one, "one@example.test"
			if i%2 == 1 {
				id, contact = two, "two@example.test"
			}
			errs <- f.app.setting.UpdateEmailNotification(smodels.EmailNotification{InboxUUID: &id, TeamContactEmail: &contact}, func(next *smodels.EmailNotification) error { return nil })
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := loadNotificationEmail(f.app.setting)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.InboxUUID == nil || cfg.TeamContactEmail == nil || (*cfg.InboxUUID == one && *cfg.TeamContactEmail != "one@example.test") || (*cfg.InboxUUID == two && *cfg.TeamContactEmail != "two@example.test") {
		t.Fatal("torn settings snapshot")
	}
}

func TestNotificationTransportLive(t *testing.T) {
	f := newNotificationHTTP(t, "notification_transport")
	port, delivered := notificationSMTP(t)
	for _, id := range []int{3087, 4087} {
		transport, err := inboxemail.New(nil, nil, inboxemail.Opts{ID: id, Name: "Test SMTP", Lo: f.app.lo, Config: imodels.Config{SMTP: []imodels.SMTPConfig{{Host: "127.0.0.1", Port: port, AuthProtocol: "none", TLSType: "none", MaxConns: 1}}}})
		if err != nil {
			t.Fatal(err)
		}
		f.app.inbox.Register(transport)
		t.Cleanup(func() { transport.Close() })
	}
	provider := &notificationEmailProvider{settings: f.app.setting, inboxes: f.app.inbox, lo: f.app.lo}
	message := notifier.Message{RecipientEmails: []string{"employee@example.test"}, Subject: "Live configuration", Content: "test", Headers: map[string][]string{"Reply-To": {"no-reply@example.test"}}}
	path := "/api/v1/settings/notifications/email"
	for _, id := range []int{3087, 4087} {
		f.request(t, 104, "PUT", path, notificationChoice(t, f, id, "team@example.test"), 200)
		if err := provider.Send(message); err != nil {
			t.Fatal(err)
		}
		select {
		case data := <-delivered:
			want := "system@example.test"
			if id == 4087 {
				want = "replacement@example.test"
			}
			parsed, err := mail.ReadMessage(strings.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if mailboxAddress(parsed.Header.Get("From")) != want || parsed.Header.Get("Reply-To") != "no-reply@example.test" || parsed.Header.Get("X-Libredesk-Loop-Prevention") != want {
				t.Fatal("sender/verification/loop header changed", data)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("SMTP delivery missing")
		}
	}
	// Optional assignments obey the live disabled setting, verification/account
	// mail remains deliverable as before, without a process-wide enabled cache.
	f.db.MustExec(`UPDATE settings SET value='false' WHERE key='notification.email.enabled'`)
	message.Optional = true
	if err := provider.Send(message); err != nil {
		t.Fatal(err)
	}
	select {
	case <-delivered:
		t.Fatal("disabled optional mail sent")
	default:
	}
	message.Optional = false
	if err := provider.Send(message); err != nil {
		t.Fatal(err)
	}
	select {
	case <-delivered:
	case <-time.After(5 * time.Second):
		t.Fatal("account mail suppressed")
	}
	// Legacy dedicated SMTP configuration also refreshes without restart.
	f.db.MustExec(`DELETE FROM settings WHERE key='notification.email.inbox_uuid'`)
	cfg := smodels.EmailNotification{EmailAddress: "legacy-notification@example.test", Host: "127.0.0.1", Port: port, AuthProtocol: "none", TLSType: "none", MaxConns: 1}
	if err := f.app.setting.UpdateEmailNotification(cfg, func(*smodels.EmailNotification) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := provider.Send(message); err != nil {
		t.Fatal(err)
	}
	select {
	case data := <-delivered:
		parsed, err := mail.ReadMessage(strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if mailboxAddress(parsed.Header.Get("From")) != "legacy-notification@example.test" {
			t.Fatal("legacy sender changed", data)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("legacy SMTP delivery missing")
	}
}

func notificationSMTP(t *testing.T) (int, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	delivered := make(chan string, 8)
	var relay sync.WaitGroup
	relay.Add(1)
	go func() {
		defer relay.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			relay.Add(1)
			go func() {
				defer relay.Done()
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
				p := textproto.NewConn(conn)
				_ = p.PrintfLine("220 test SMTP")
				for {
					line, err := p.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_ = p.PrintfLine("250 test SMTP")
					case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"), line == "RSET":
						_ = p.PrintfLine("250 OK")
					case line == "DATA":
						_ = p.PrintfLine("354 data")
						data, err := io.ReadAll(p.DotReader())
						if err != nil {
							return
						}
						delivered <- string(data)
						_ = p.PrintfLine("250 accepted")
					case line == "QUIT":
						_ = p.PrintfLine("221 bye")
						return
					default:
						_ = p.PrintfLine("250 OK")
					}
				}
			}()
		}
	}()
	t.Cleanup(func() { listener.Close(); relay.Wait() })
	port, err := strconv.Atoi(strings.Split(listener.Addr().String(), ":")[1])
	if err != nil {
		t.Fatal(err)
	}
	return port, delivered
}
