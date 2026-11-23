package conversation

import (
	"slices"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
)

func TestStoredInternalMessageContract(t *testing.T) {
	m := emailThreadFixture(t, "stored_internal_contract")
	hooks := &bootstrapWebhooks{}
	m.webhookStore = hooks
	id, uuid, err := m.CreateConversation(0, 102, "", time.Now(), "Internal", false, nil, map[string]any{"internal": true, "private": true, "creator_id": 101, "visible_users": []int{101}}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	msg, err := m.StoreInternalMessage(nil, 101, uuid, "<p>Complete original content</p>")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Type != models.MessageInternal || msg.Status != models.MessageStatusStored || msg.Private || msg.ConversationID != id {
		t.Fatalf("invalid history: %+v", msg)
	}
	if slices.Contains(hooks.events, wmodels.EventMessageCreated) || slices.Contains(hooks.events, wmodels.EventMessageUpdated) {
		t.Fatal("stored content emitted email-message webhook", hooks.events)
	}
	var pending []models.Message
	if err := m.q.GetOutgoingPendingMessages.Select(&pending, "{}"); err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatal("internal message entered mail scanner")
	}
	// Ordinary and customer tickets still require an actual contact.
	if _, err := m.db.Exec(`INSERT INTO conversations(inbox_id,status_id) VALUES(102,101)`); err == nil {
		t.Fatal("public ticket accepted no contact")
	}
}
