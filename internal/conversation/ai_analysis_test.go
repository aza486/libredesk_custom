package conversation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func validAnalysis() AIAnalysis {
	return AIAnalysis{Kind: "n8n_ai_analysis", Version: 1, OriginalMessageUUID: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", SuggestedMessage: "Hallo\nGerne helfen wir. 2 < 3."}
}
func TestAIAnalysisValidation(t *testing.T) {
	a := validAnalysis()
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*AIAnalysis){func(a *AIAnalysis) { a.Kind = "other" }, func(a *AIAnalysis) { a.Version = 2 }, func(a *AIAnalysis) { a.OriginalMessageUUID = "bad" }, func(a *AIAnalysis) { a.OriginalMessageUUID = "urn:uuid:aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa" }, func(a *AIAnalysis) { a.SuggestedMessage = " " }, func(a *AIAnalysis) { a.SuggestedMessage = "<b>HTML</b>" }, func(a *AIAnalysis) { a.SuggestedMessage = "<img src=x onerror=alert(1)>" }, func(a *AIAnalysis) { a.SuggestedMessage = strings.Repeat("x", 65537) }, func(a *AIAnalysis) { a.SuggestedMessage = strings.Repeat("🙂", 32769) }} {
		a := validAnalysis()
		change(&a)
		if a.Validate() == nil {
			t.Fatalf("invalid analysis accepted: kind=%s version=%d", a.Kind, a.Version)
		}
	}
	for _, text := range []string{strings.Repeat("x", 65536), strings.Repeat("🙂", 32768)} {
		a.SuggestedMessage = text
		if err := a.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	original := models.Message{UUID: a.OriginalMessageUUID, ConversationUUID: "conv", Type: models.MessageIncoming, SenderType: models.SenderTypeContact}
	if err := ValidateAIAnalysisOriginal(a, original, "conv"); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*models.Message){func(m *models.Message) { m.ConversationUUID = "other" }, func(m *models.Message) { m.Private = true }, func(m *models.Message) { m.Type = models.MessageOutgoing }, func(m *models.Message) { m.SenderType = models.SenderTypeAgent }, func(m *models.Message) { m.UUID = "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb" }} {
		m := original
		change(&m)
		if ValidateAIAnalysisOriginal(a, m, "conv") == nil {
			t.Fatal("foreign or invalid original accepted")
		}
	}
}

// Exercise the existing insert statement and the exact Manager.GetMessage path
// used by the public message GET; the full hook stack is not duplicated here.
func TestAIAnalysisPersistenceAndMessageRead(t *testing.T) {
	db := testutil.NewPersonalDB(t, "ai_analysis_metadata")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var convUUID string
	if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`).Scan(&convUUID); err != nil {
		t.Fatal(err)
	}
	a := validAnalysis()
	if _, err := db.Exec(`INSERT INTO conversation_messages(uuid,conversation_id,type,status,sender_id,sender_type,content) SELECT $1,id,'incoming','received',105,'contact','Customer' FROM conversations WHERE uuid=$2`, a.OriginalMessageUUID, convUUID); err != nil {
		t.Fatal(err)
	}
	original, err := m.GetMessage(a.OriginalMessageUUID)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAIAnalysisOriginal(a, original, convUUID); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"ai_analysis": a})
	message := models.Message{}
	html := `<b>KI-Analyse</b><p>{{do_not_render}}</p>`
	if err := m.q.InsertMessage.Get(&message, models.MessageOutgoing, models.MessageStatusSent, 0, convUUID, html, "KI-Analyse", 101, models.SenderTypeAgent, true, models.ContentTypeHTML, null.String{}, string(meta)); err != nil {
		t.Fatal(err)
	}
	fetched, err := m.GetMessage(message.UUID)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Analysis AIAnalysis `json:"ai_analysis"`
	}
	if json.Unmarshal(fetched.Meta, &stored) != nil || stored.Analysis != a || fetched.Content != html || !fetched.Private || fetched.Type != models.MessageOutgoing {
		t.Fatalf("incorrect persisted note: %+v", fetched)
	}
	missing := validAnalysis()
	missing.OriginalMessageUUID = "dddddddd-dddd-4ddd-dddd-dddddddddddd"
	if _, err := m.SendAIAnalysisNote(101, convUUID, "<p>Note</p>", missing); err == nil {
		t.Fatal("nonexistent original accepted")
	} else if e, ok := err.(envelope.Error); !ok || e.ErrorType != envelope.NotFoundError {
		t.Fatalf("missing original: %v", err)
	}
	if _, err := m.GetMessage("dddddddd-dddd-4ddd-dddd-dddddddddddd"); err == nil {
		t.Fatal("nonexistent original was returned")
	}
}
