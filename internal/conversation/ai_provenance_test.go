package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

type provenanceProbe struct {
	calls   int
	fail    bool
	payload map[string]string
}

func (p *provenanceProbe) Call(_ context.Context, header, id string, payload, result any) (int, error) {
	p.calls++
	p.payload = payload.(map[string]string)
	if p.fail {
		return 502, errors.New("offline")
	}
	ack, _ := json.Marshal(map[string]any{"committed": true, "outgoing_message_uuid": id})
	return 200, json.Unmarshal(ack, result)
}
func TestAIProvenanceOnlyAfterSentAndIndependentRetry(t *testing.T) {
	db := testutil.NewPersonalDB(t, "ai_provenance")
	m := &Manager{db: db}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var convID int
	var convUUID, source, outgoing string
	if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING id,uuid`).Scan(&convID, &convUUID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','sent',105,'contact','source') RETURNING uuid`, convID).Scan(&source); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"ai_response": map[string]string{"source_message_uuid": source, "response_source": "ai_adapted"}, "ai_response_sent_body": "<p>Rendered actual answer</p>"})
	if err := db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta) VALUES($1,'outgoing','pending',101,'agent','raw',$2::jsonb) RETURNING uuid`, convID, string(meta)).Scan(&outgoing); err != nil {
		t.Fatal(err)
	}
	probe := &provenanceProbe{fail: true}
	ctx := context.Background()
	for _, status := range []string{"pending", "failed"} {
		if _, err := m.q.UpdateMessageStatus.Exec(status, outgoing); err != nil {
			t.Fatal(err)
		}
		m.deliverAIProvenance(ctx, probe)
	}
	if probe.calls != 0 {
		t.Fatal("reported unsent response")
	}
	if _, err := m.q.UpdateMessageStatus.Exec("sent", outgoing); err != nil {
		t.Fatal(err)
	}
	m.deliverAIProvenance(ctx, probe)
	if probe.calls != 1 {
		t.Fatal("missing first attempt")
	}
	probe.fail = false
	m.deliverAIProvenance(ctx, probe)
	m.deliverAIProvenance(ctx, probe)
	if probe.calls != 2 {
		t.Fatal("commit was not remembered")
	}
	if probe.payload["source_message_uuid"] != source || probe.payload["outgoing_message_uuid"] != outgoing || probe.payload["response_source"] != "ai_adapted" || probe.payload["sent_message"] != "<p>Rendered actual answer</p>" || probe.payload["sent_at"] == "" {
		t.Fatal(probe.payload)
	}
	var status string
	var raw []byte
	if err := db.QueryRow(`SELECT status,meta FROM conversation_messages WHERE uuid=$1`, outgoing).Scan(&status, &raw); err != nil {
		t.Fatal(err)
	}
	var stored map[string]any
	json.Unmarshal(raw, &stored)
	if status != "sent" || stored["ai_feedback_original_message_uuid"] != nil {
		t.Fatal("provenance modified email or learning feedback")
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND type='outgoing'`, convID)
	if count != 1 {
		t.Fatal("duplicate customer reply")
	}
	if used, err := m.AIResponseUsed(convUUID, source); err != nil || !used {
		t.Fatal("source reuse not detected", err)
	}
}
