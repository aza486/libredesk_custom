package main

import (
	"encoding/json"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"testing"
)

func TestStyleDecodeAndNoteBinding(t *testing.T) {
	for _, body := range []string{`{"agent_id":"spoof"}`, `{} {}`, `{"meta":{}}`} {
		if _, err := decodeStyle([]byte(body)); err == nil {
			t.Fatal("accepted unexpected field")
		}
	}
	if _, err := decodeStyle([]byte(`{"style":"mine","instructions":"","original_text":"Hallo","current_text":"Hallo","note_uuid":"id"}`)); err != nil {
		t.Fatal(err)
	}
	note := models.Message{Private: true, Type: "outgoing", ConversationUUID: "conversation", Meta: json.RawMessage(`{"ai_analysis":{"kind":"n8n_ai_analysis","version":1,"original_message_uuid":"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa","suggested_message":"Hallo"}}`)}
	if _, ok := validatedNote(note, "conversation"); !ok {
		t.Fatal("valid note")
	}
	if _, ok := validatedNote(note, "other"); ok {
		t.Fatal("foreign note")
	}
	note.Private = false
	if _, ok := validatedNote(note, "conversation"); ok {
		t.Fatal("public note")
	}
	note.Private = true
	note.Meta = json.RawMessage(`{"deleted_at":"now"}`)
	if _, ok := validatedNote(note, "conversation"); ok {
		t.Fatal("deleted note")
	}
}
func TestStyleRequiresSession(t *testing.T) {
	for _, method := range []string{"", authMethodAPIKey} {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("auth_method", method)
		if err := handleAIStyleAdjust(&fastglue.Request{RequestCtx: ctx, Context: &App{}}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != 403 {
			t.Fatal("non-session accepted")
		}
	}
}
func TestResponseProvenanceBoundary(t *testing.T) {
	for _, source := range []string{"ai", "ai_adapted", "human", "other"} {
		p := responseProvenance{Source: source, SourceUUID: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", NoteUUID: "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb"}
		feedback := ""
		if source == "human" {
			feedback = p.SourceUUID
		}
		if got := validProvenance(p, false, "agent", "email", authMethodSession, feedback); got != (source != "other") {
			t.Fatal(source)
		}
		if validProvenance(p, true, "agent", "email", authMethodSession, feedback) || validProvenance(p, false, "contact", "email", authMethodSession, feedback) || validProvenance(p, false, "agent", "email", authMethodAPIKey, feedback) {
			t.Fatal("unauthorized provenance")
		}
		if source == "ai" && validProvenance(p, false, "agent", "email", authMethodSession, "source") {
			t.Fatal("AI learning feedback accepted")
		}
	}
}
