package conversation

import (
	"encoding/json"
	"testing"
	"time"

	azmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
)

type spamNoteWebhooks struct {
	threadingWebhooks
	created int
}

func (h *spamNoteWebhooks) TriggerEvent(event wmodels.WebhookEvent, _ any) {
	if event == wmodels.EventMessageCreated {
		h.created++
	}
}

func spamNoteActor() umodels.User {
	return umodels.User{ID: 101, Enabled: true, Type: umodels.UserTypeAgent, Permissions: []string{azmodels.PermMessagesWriteSpamDetectionNote}}
}

func TestSpamDetectionNoteStoredWithoutDelivery(t *testing.T) {
	m := emailThreadFixture(t, "spam_note_delivery")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	hooks := &spamNoteWebhooks{}
	m.webhookStore = hooks
	transport := &manualSendInbox{}
	m.inboxStore = manualSendInboxes{transport: transport}
	var uuid string
	if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,12 FROM conversations WHERE uuid=$1`, uuid)
	actor := spamNoteActor()
	for _, invalid := range []umodels.User{
		{ID: 101, Enabled: true, Type: "agent"},
		{ID: 101, Enabled: false, Type: "agent", Permissions: actor.Permissions},
		{ID: 101, Enabled: true, Type: "contact", Permissions: actor.Permissions},
	} {
		if _, err := m.SendSpamDetectionNote(invalid, uuid, "Spam detected"); err == nil {
			t.Fatal("invalid actor accepted")
		}
	}
	message, err := m.SendSpamDetectionNote(actor, uuid, "Spam detected {{ template }}")
	if err != nil {
		t.Fatal(err)
	}
	if !message.Private || message.Status != models.MessageStatusSent || message.Content != "Spam detected {{ template }}" {
		t.Fatal("note escaped stored-only path or rendered a template")
	}
	if hooks.created != 0 {
		t.Fatal("security note triggered an external creation webhook")
	}
	m.sendOutgoingMessage(message)
	// Even a future explicit retry cannot put a private note into the send queue.
	if err := m.MarkMessageAsPending(message.UUID); err != nil {
		t.Fatal(err)
	}
	message.Status = models.MessageStatusPending
	m.sendOutgoingMessage(message)
	var queued []models.Message
	if err := m.q.GetOutgoingPendingMessages.Select(&queued, "{}"); err != nil {
		t.Fatal(err)
	}
	if len(transport.sent) != 0 || len(queued) != 0 {
		t.Fatal("security note reached delivery or pending queue")
	}
	// Security metadata cannot excuse ordinary automatic AI publication.
	forged := models.Message{ConversationUUID: uuid, SenderID: 101, SenderType: "agent", Private: true,
		Type: models.MessageOutgoing, Content: "forged", Meta: json.RawMessage(`{"automatic_email_ai":true,"security_note":{"purpose":"spam_detection"}}`)}
	if err := m.InsertMessage(&forged); err == nil {
		t.Fatal("metadata bypassed automatic AI gate")
	}
}

func TestSpamDetectionNoteSerializesWithTagRemoval(t *testing.T) {
	m := emailThreadFixture(t, "spam_note_lock")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	var uuid string
	if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,12 FROM conversations WHERE uuid=$1`, uuid)
	tx, err := m.lockSpamConversation(uuid)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM conversation_tags WHERE conversation_id=(SELECT id FROM conversations WHERE uuid=$1) AND tag_id=12`, uuid); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, err := m.SendSpamDetectionNote(spamNoteActor(), uuid, "Spam detected"); done <- err }()
	// Prove the insert waits for the tag transition, rather than relying on timing.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var blocked bool
		if err := m.db.Get(&blocked, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND cardinality(pg_blocking_pids(pid))>0)`); err != nil {
			t.Fatal(err)
		}
		if blocked {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("note did not acquire the Spam transition lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("note used stale Spam state")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("note remained blocked after commit")
	}
	var count int
	if err := m.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=(SELECT id FROM conversations WHERE uuid=$1)`, uuid); err != nil || count != 0 {
		t.Fatalf("rejected note persisted: %d %v", count, err)
	}
}
