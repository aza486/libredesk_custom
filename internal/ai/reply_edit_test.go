package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/crypto"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

// Exercise real provider serialization, workspace instructions and tool registry
// using isolated PostgreSQL and a synthetic provider. This tests the contract,
// not the language quality of a live model.
func TestReplyGenerationAndEditKeepContext(t *testing.T) {
	db := testutil.NewPersonalDB(t, "reply_edit")
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_providers(id SERIAL PRIMARY KEY, created_at TIMESTAMPTZ DEFAULT now(), updated_at TIMESTAMPTZ DEFAULT now(),name TEXT,provider TEXT,type TEXT,config JSONB,is_default BOOL)`)
	if err != nil {
		t.Fatal(err)
	}
	var captured models.ChatCompletionPayload
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"Reviewed reply"}}]}`)
	}))
	defer s.Close()
	key, err := crypto.Encrypt("synthetic-key", testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := json.Marshal(models.ProviderConfig{APIKey: key, BaseURL: s.URL, Model: "synthetic-model", Instructions: "Workspace voice"})
	if _, err := db.Exec(`UPDATE ai_providers SET config=$1 WHERE type='completion'`, string(cfg)); err != nil {
		t.Fatal(err)
	}
	m := newTestManager(t)
	m.q.GetProviderByType, err = db.Preparex(`SELECT id,created_at,updated_at,name,provider,type,config,is_default FROM ai_providers WHERE type=$1`)
	if err != nil {
		t.Fatal(err)
	}
	m.providerHTTPClient = s.Client()
	thread := "Customer: Order 42?\nAgent: Delivery Friday confirmed."
	out, err := m.GenerateReply(context.Background(), thread, "", ToolContext{}, nil)
	if err != nil || out != "Reviewed reply" {
		t.Fatalf("%q %v", out, err)
	}
	if captured.Messages[0].Content != replyDraftSystemPrompt+"\n\nWorkspace admin instructions (follow these, they take precedence on tone and format):\nWorkspace voice" {
		t.Fatal("default prompt changed")
	}
	if !strings.Contains(captured.Messages[1].Content, thread) {
		t.Fatal("legacy context missing")
	}
	for _, instruction := range []string{"Correct spelling only", "Translate into English"} {
		out, err = m.EditReply(context.Background(), thread, instruction, "Order 42 arrives Friday.", ToolContext{}, nil)
		if err != nil || out != "Reviewed reply" {
			t.Fatalf("%q %v", out, err)
		}
		all, _ := json.Marshal(captured.Messages)
		for _, want := range []string{"Order 42?", "Delivery Friday confirmed.", "Order 42 arrives Friday.", instruction, "Workspace voice", "Preserve meaning"} {
			if !strings.Contains(string(all), want) {
				t.Errorf("missing %s", want)
			}
		}
		if len(captured.Tools) != 1 || captured.Tools[0].Function.Name != "search_articles" {
			t.Fatal("knowledge tool missing")
		}
	}
}
