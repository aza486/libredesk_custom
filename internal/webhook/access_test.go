package webhook

import (
	"encoding/json"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/zerodha/logf"
)

func TestInboxAccessOnEveryWebhook(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_webhook")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, deliveryQueue: make(chan DeliveryTask, 20)}
	for _, inboxID := range []int{101, 102} {
		var uuid string
		attrs := `{}`
		wantMode := `"public"`
		wantOwner := "null"
		if inboxID == 101 {
			attrs = `{"access_mode":"personal","owner_user_id":101}`
			wantMode = `"personal"`
			wantOwner = "101"
		}
		err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,$1,101,$2) RETURNING uuid`, inboxID, attrs).Scan(&uuid)
		if err != nil {
			t.Fatal(err)
		}
		if inboxID == 101 {
			db.MustExec(`UPDATE inboxes SET owner_user_id=102 WHERE id=101`)
		}
		for _, event := range []models.WebhookEvent{models.EventConversationCreated, models.EventConversationAssigned, models.EventConversationUnassigned, models.EventConversationStatusChanged, models.EventConversationTagsChanged, models.EventMessageCreated, models.EventMessageUpdated} {
			var payload any = map[string]any{"conversation_uuid": uuid, "unchanged": 123}
			if event == models.EventConversationCreated {
				payload = cmodels.Conversation{UUID: uuid, InboxID: inboxID}
			}
			if event == models.EventMessageCreated || event == models.EventMessageUpdated {
				payload = cmodels.Message{ConversationUUID: uuid, UUID: "message-uuid", Content: "unchanged"}
			}
			m.TriggerEvent(event, payload)
			select {
			case task := <-m.deliveryQueue:
				b, _ := json.Marshal(task.Payload)
				var fields map[string]json.RawMessage
				json.Unmarshal(b, &fields)
				if string(fields["inbox_access_mode"]) != wantMode || string(fields["owner_user_id"]) != wantOwner {
					t.Fatalf("%s: %s", event, b)
				}
				before, _ := json.Marshal(payload)
				var original map[string]json.RawMessage
				json.Unmarshal(before, &original)
				for k, v := range original {
					if k == "inbox_access_mode" || k == "owner_user_id" {
						continue
					}
					if string(fields[k]) != string(v) {
						t.Fatalf("changed field %s", k)
					}
				}
			default:
				t.Fatalf("%s did not fire", event)
			}
		}
		m.TriggerWebhook(1, models.EventConversationCreated, map[string]any{"conversation": cmodels.Conversation{UUID: uuid}})
		select {
		case task := <-m.deliveryQueue:
			if task.WebhookID != 1 {
				t.Fatal("target changed")
			}
		default:
			t.Fatal("targeted webhook missing")
		}
	}
}
