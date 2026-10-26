package aifeedback

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestManualResolutionClientUsesExistingCredentialAndReceipt(t *testing.T) {
	const id = "dddddddd-dddd-4ddd-dddd-dddddddddddd"
	calls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer same-credential" || r.Header.Get("Idempotency-Key") != id {
			t.Error("feedback credential or idempotency key changed")
		}
		var body map[string]any
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("bad JSON")
		}
		if body["feedback_type"] != "manual_resolution" || body["feedback_reason"] != nil || body["response_source"] != "human" || len(body) != 10 {
			t.Error("manual feedback contract changed")
		}
		json.NewEncoder(w).Encode(map[string]any{"id": id, "committed": calls > 1})
	}))
	defer server.Close()
	client := &Client{endpoint: server.URL, token: "same-credential", http: server.Client()}
	submission := ManualResolutionSubmission{ID: id, MailEventMessageUUID: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", ConversationUUID: "cccccccc-cccc-4ccc-cccc-cccccccccccc", OutgoingMessageUUID: "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb", AgentID: "101", HumanAnswer: "<html>Actual answer</html>", FeedbackType: "manual_resolution", ResponseSource: "human", SentAt: time.Now().UTC()}
	if client.SubmitManualResolution(context.Background(), submission) == nil {
		t.Fatal("uncommitted receipt accepted")
	}
	if err := client.SubmitManualResolution(context.Background(), submission); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatal("unexpected calls")
	}
}
