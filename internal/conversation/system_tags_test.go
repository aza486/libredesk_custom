package conversation

import (
	"slices"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/tag"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
)

type bootstrapWebhooks struct {
	threadingWebhooks
	events []wmodels.WebhookEvent
}

func (w *bootstrapWebhooks) TriggerEvent(event wmodels.WebhookEvent, _ any) {
	w.events = append(w.events, event)
}

func TestSystemTagBootstrapClassifications(t *testing.T) {
	m := emailThreadFixture(t, "system_tag_classifications")
	// Start with no fixture tags; all five rows come from the real bootstrap.
	m.db.MustExec(`DELETE FROM tags`)
	if err := tag.EnsureSystemTags(t.Context(), m.db); err != nil {
		t.Fatal(err)
	}
	hooks := &bootstrapWebhooks{}
	m.webhookStore = hooks
	assertTags := func(uuid string, want ...string) {
		t.Helper()
		got, err := m.getConversationTags(uuid)
		slices.Sort(got)
		slices.Sort(want)
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("tags: got %v want %v, %v", got, want, err)
		}
	}
	in := threadMail("bootstrap-customer", "")
	in.Contact.ID = 105
	_, customer, created, err := m.findOrCreateConversation(in)
	if err != nil || !created {
		t.Fatalf("customer creation: %v %v", created, err)
	}
	assertTags(customer, "🦽Kundenticket")
	if err := m.AddSystemTags(customer, []string{"😎Mensch erforderlich"}); err != nil {
		t.Fatal(err)
	}
	assertTags(customer, "🦽Kundenticket", "😎Mensch erforderlich")
	if err := m.AddServiceEmailAddress("contact@example.com"); err != nil {
		t.Fatal(err)
	}
	in.SourceID.String = "bootstrap-service"
	_, service, created, err := m.findOrCreateConversation(in)
	if err != nil || !created {
		t.Fatalf("service creation: %v %v", created, err)
	}
	assertTags(service, "🧷Service-Mail")
	if slices.Contains(hooks.events, wmodels.EventConversationTagsChanged) {
		t.Fatal("system classification emitted a previously suppressed tag webhook")
	}
	beforeHooks := len(hooks.events)
	if err := tag.EnsureSystemTags(t.Context(), m.db); err != nil {
		t.Fatal(err)
	}
	if len(hooks.events) != beforeHooks {
		t.Fatal("bootstrap emitted webhooks")
	}
	actor := umodels.User{ID: 101, FirstName: "Owner"}
	if err := m.SetConversationTags(service, amodels.ActionAddTags, []string{"🗑Spam"}, actor); err != nil {
		t.Fatal(err)
	}
	assertTags(service, "🗑Spam", "🧷Service-Mail")
	var status, snapshot string
	if err := m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE c.uuid=$1`, service); err != nil || status != "Closed" {
		t.Fatalf("spam did not close service ticket: %q %v", status, err)
	}
	if err := m.db.Get(&snapshot, `SELECT custom_attributes->'_spam_previous_state' FROM conversations WHERE uuid=$1`, service); err != nil || snapshot == "" {
		t.Fatalf("missing spam restoration state: %q %v", snapshot, err)
	}
	// A startup check must preserve existing spam state and every association.
	if err := tag.EnsureSystemTags(t.Context(), m.db); err != nil {
		t.Fatal(err)
	}
	var after string
	if err := m.db.Get(&after, `SELECT custom_attributes->'_spam_previous_state' FROM conversations WHERE uuid=$1`, service); err != nil || after != snapshot {
		t.Fatalf("bootstrap changed spam state: %q -> %q %v", snapshot, after, err)
	}
	if err := m.SetConversationTags(service, amodels.ActionRemoveTags, []string{"🗑Spam"}, actor); err != nil {
		t.Fatal(err)
	}
	assertTags(service, "🧷Service-Mail")
	if err := m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE c.uuid=$1`, service); err != nil || status != "Open" {
		t.Fatalf("spam state not restored: %q %v", status, err)
	}
}
