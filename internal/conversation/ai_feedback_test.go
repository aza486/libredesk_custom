package conversation

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/aifeedback"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestAIFeedbackSnapshotAndDeliveryConfirmation(t *testing.T) {
	db := testutil.NewPersonalDB(t, "ai_feedback_delivery")
	m := &Manager{db: db}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var conversationID int
	if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING id`).Scan(&conversationID); err != nil {
		t.Fatal(err)
	}
	for _, tagged := range []bool{true, false} {
		meta := `{}`
		if tagged {
			meta = `{"ai_feedback_original_message_uuid":"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"}`
		}
		var messageUUID string
		if err := db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta) VALUES($1,'outgoing','pending',101,'agent','raw',$2::jsonb) RETURNING uuid`, conversationID, meta).Scan(&messageUUID); err != nil {
			t.Fatal(err)
		}
		if tagged {
			snapshot, _ := json.Marshal(map[string]any{aifeedback.BodyKey: `<html>Agent's actual answer</html>`})
			if _, err := m.q.UpdateMessageFeedbackSnapshot.Exec(string(snapshot), messageUUID); err != nil {
				t.Fatal(err)
			}
		}
		for _, status := range []string{"pending", "failed", "sent"} {
			if _, err := m.q.UpdateMessageStatus.Exec(status, messageUUID); err != nil {
				t.Fatal(err)
			}
			var raw []byte
			if err := db.Get(&raw, `SELECT meta FROM conversation_messages WHERE uuid=$1`, messageUUID); err != nil {
				t.Fatal(err)
			}
			var stored map[string]any
			if err := json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			if (stored[aifeedback.SentAtKey] != nil) != (tagged && status == "sent") {
				t.Fatalf("unexpected delivery confirmation: %s", raw)
			}
			if tagged && (stored[aifeedback.OriginalKey] != "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa" || stored[aifeedback.BodyKey] != `<html>Agent's actual answer</html>`) {
				t.Fatalf("snapshot changed: %s", raw)
			}
		}
	}
}
