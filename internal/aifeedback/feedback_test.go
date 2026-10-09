package aifeedback

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

func messages() (cmodels.Message, cmodels.Message) {
	original := cmodels.Message{UUID: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", ConversationUUID: "conversation", Type: cmodels.MessageIncoming, SenderType: cmodels.SenderTypeContact}
	meta, _ := json.Marshal(map[string]any{OriginalKey: original.UUID, BodyKey: "<html>Actually sent</html>", SentAtKey: "2026-10-08T12:00:00Z"})
	reply := cmodels.Message{UUID: "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb", ConversationUUID: "conversation", Type: cmodels.MessageOutgoing, SenderType: cmodels.SenderTypeAgent, SenderID: 42, Status: cmodels.MessageStatusSent, Meta: meta, Content: "unrendered", UpdatedAt: time.Now()}
	return reply, original
}

func TestBuild(t *testing.T) {
	reply, original := messages()
	first, err := Build(reply, original, "conversation", 42, "  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(reply, original, "conversation", 42, "reason")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.FeedbackReason != nil || first.FeedbackType != nil || first.QualityStatus != "pending" || first.ResponseSource != "human" || first.AgentID != "42" || first.HumanAnswer != "<html>Actually sent</html>" || first.MailEventMessageUUID != original.UUID || first.SentAt.Format(time.RFC3339) != "2026-10-08T12:00:00Z" {
		t.Fatalf("unexpected submission: %+v", first)
	}
	if *second.FeedbackReason != "reason" {
		t.Fatal("reason missing")
	}
	reply.UUID = "cccccccc-cccc-4ccc-cccc-cccccccccccc"
	third, _ := Build(reply, original, "conversation", 42, "")
	if first.ID == third.ID {
		t.Fatal("different sends share identity")
	}
}

func TestRejectInvalidMessages(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*cmodels.Message, *cmodels.Message)
		want   error
	}{
		{"pending", func(r, o *cmodels.Message) { r.Status = cmodels.MessageStatusPending }, ErrNotSent},
		{"failed", func(r, o *cmodels.Message) { r.Status = cmodels.MessageStatusFailed }, ErrNotSent},
		{"other agent", func(r, o *cmodels.Message) { r.SenderID = 43 }, ErrInvalid},
		{"private reply", func(r, o *cmodels.Message) { r.Private = true }, ErrInvalid},
		{"contact reply", func(r, o *cmodels.Message) { r.SenderType = cmodels.SenderTypeContact }, ErrInvalid},
		{"incoming reply", func(r, o *cmodels.Message) { r.Type = cmodels.MessageIncoming }, ErrInvalid},
		{"foreign reply", func(r, o *cmodels.Message) { r.ConversationUUID = "other" }, ErrInvalid},
		{"foreign original", func(r, o *cmodels.Message) { o.ConversationUUID = "other" }, ErrInvalid},
		{"outgoing original", func(r, o *cmodels.Message) { o.Type = cmodels.MessageOutgoing }, ErrInvalid},
		{"private original", func(r, o *cmodels.Message) { o.Private = true }, ErrInvalid},
		{"agent original", func(r, o *cmodels.Message) { o.SenderType = cmodels.SenderTypeAgent }, ErrInvalid},
		{"wrong customer message in same conversation", func(r, o *cmodels.Message) { o.UUID = "cccccccc-cccc-4ccc-cccc-cccccccccccc" }, ErrInvalid},
		{"missing metadata", func(r, o *cmodels.Message) { r.Meta = nil }, ErrInvalid},
		{"missing snapshot", func(r, o *cmodels.Message) {
			r.Meta = json.RawMessage(`{"ai_feedback_original_message_uuid":"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"}`)
		}, ErrNotSent},
		{"invalid UUID", func(r, o *cmodels.Message) { r.UUID = "invalid" }, ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, o := messages()
			tt.mutate(&r, &o)
			_, err := Build(r, o, "conversation", 42, "")
			if !errors.Is(err, tt.want) {
				t.Fatalf("got %v want %v", err, tt.want)
			}
		})
	}
	r, o := messages()
	if _, err := Build(r, o, "conversation", 42, strings.Repeat("x", 8001)); !errors.Is(err, ErrInvalid) {
		t.Fatal("oversized reason accepted")
	}
}

func TestClientContract(t *testing.T) {
	r, o := messages()
	submission, _ := Build(r, o, "conversation", 42, "")
	for _, tt := range []struct {
		name    string
		status  int
		body    string
		wantErr bool
	}{
		{"commit", 201, `{"id":"` + submission.ID + `","committed":true}`, false},
		{"duplicate", 200, `{"id":"` + submission.ID + `","committed":true}`, false},
		{"missing mail event", 404, `{}`, true},
		{"wrong conversation", 409, `{}`, true},
		{"database failure", 500, `{}`, true},
		{"unconfirmed queued execution", 202, `{}`, true},
		{"unconfirmed commit", 200, `{"id":"` + submission.ID + `","committed":false}`, true},
		{"wrong receipt", 200, `{"id":"wrong","committed":true}`, true},
		{"invalid JSON", 200, `not JSON`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls++
				if req.Method != "POST" || req.Header.Get("Authorization") != "Bearer secret" || req.Header.Get("Idempotency-Key") != submission.ID {
					t.Error("invalid request")
				}
				var got Submission
				if json.NewDecoder(req.Body).Decode(&got) != nil || got.HumanAnswer != submission.HumanAnswer || got.MailEventMessageUUID != o.UUID {
					t.Error("invalid payload")
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			client, err := New(server.URL, "secret")
			if err != nil {
				t.Fatal(err)
			}
			client.http.Transport = server.Client().Transport
			err = client.Submit(context.Background(), submission)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error %v", err)
			}
			if calls != 1 {
				t.Fatal("unexpected automatic retry")
			}
		})
	}
}

func TestClientConfigurationAndTransportFailure(t *testing.T) {
	if c, err := New("", ""); c != nil || err != nil {
		t.Fatal("disabled configuration failed")
	}
	for _, endpoint := range []string{"http://example.com", "https://user:password@example.com", "https://example.com/#fragment", "invalid"} {
		if _, err := New(endpoint, "secret"); err == nil {
			t.Fatalf("accepted %q", endpoint)
		}
	}
	if _, err := New("https://example.com", ""); err == nil {
		t.Fatal("missing token accepted")
	}
	client, _ := New("https://example.com", "secret")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Submit(ctx, Submission{}); err == nil {
		t.Fatal("canceled request succeeded")
	}
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	targetCalls := 0
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++ }))
	defer target.Close()
	source := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	client, _ := New(source.URL, "secret")
	client.http.Transport = source.Client().Transport
	if err := client.Submit(context.Background(), Submission{}); err == nil {
		t.Fatal("redirect accepted")
	}
	if targetCalls != 0 {
		t.Fatal("credentials forwarded")
	}
}
