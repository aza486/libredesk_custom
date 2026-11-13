package conversation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestConversationListTimestamps(t *testing.T) {
	db := testutil.NewPersonalDB(t, "list_timestamps")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: folderUsers{}, settingsStore: personalSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	var id int
	var uuid string
	if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,created_at) VALUES(105,102,101,$1) RETURNING id,uuid`, created).Scan(&id, &uuid); err != nil {
		t.Fatal(err)
	}
	check := func(customer, activity time.Time) {
		t.Helper()
		item, err := m.GetConversationListItem(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if item.LastCustomerReplyAt.Valid != !customer.IsZero() || (!customer.IsZero() && !item.LastCustomerReplyAt.Time.Equal(customer)) {
			t.Fatalf("customer = %v, want %v", item.LastCustomerReplyAt, customer)
		}
		if !activity.IsZero() && !item.LastMessageAt.Time.Equal(activity) {
			t.Fatalf("activity = %v, want %v", item.LastMessageAt, activity)
		}
		rows, err := m.GetAllConversationsList(104, "", "", "[]", 1, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].LastCustomerReplyAt.Valid != !customer.IsZero() || (!customer.IsZero() && !rows[0].LastCustomerReplyAt.Time.Equal(customer)) {
			t.Fatalf("list customer time differs: %+v", rows)
		}
		detail, err := m.GetConversation(0, uuid, "")
		if err != nil {
			t.Fatal(err)
		}
		if detail.LastCustomerReplyAt.Valid != !customer.IsZero() || (!customer.IsZero() && !detail.LastCustomerReplyAt.Time.Equal(customer)) {
			t.Fatalf("detail customer time = %v", detail.LastCustomerReplyAt)
		}
		payload, err := json.Marshal(convToBroadcast(&item))
		if err != nil {
			t.Fatal(err)
		}
		var event map[string]any
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatal(err)
		}
		if (!customer.IsZero() && event["last_customer_reply_at"] != customer.Format(time.RFC3339)) || (customer.IsZero() && event["last_customer_reply_at"] != nil) {
			t.Fatalf("WS customer time = %v", event["last_customer_reply_at"])
		}
	}
	check(time.Time{}, time.Time{})
	insert := func(kind, sender string, private bool, hour int) time.Time {
		t.Helper()
		at := created.Add(time.Duration(hour) * time.Hour)
		senderID := 101
		if sender == "contact" {
			senderID = 105
		}
		if _, err := db.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content,created_at) VALUES($1,$2,'sent',$3,$4,$5,'test',$6)`, id, kind, senderID, sender, private, at); err != nil {
			t.Fatal(err)
		}
		if err := m.UpdateConversationLastMessage(id, uuid, "test", sender, kind, private, at, senderID); err != nil {
			t.Fatal(err)
		}
		return at
	}
	customer := insert("incoming", "contact", false, 1)
	check(customer, customer)
	for _, tc := range []struct {
		name, kind, sender string
		private            bool
	}{
		{"employee reply", "outgoing", "agent", false},
		{"AI outgoing reply", "outgoing", "agent", false},
		{"internal note", "outgoing", "agent", true},
		{"status activity", "activity", "agent", true},
		{"private customer excluded", "incoming", "contact", true},
		{"agent incoming excluded", "incoming", "agent", false},
	} {
		t.Run(tc.name, func(t *testing.T) { activity := insert(tc.kind, tc.sender, tc.private, 4); check(customer, activity) })
	}
	// Later insertion with an older date must not become the latest customer reply.
	if _, err := db.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,created_at) VALUES($1,'incoming','sent',105,'contact','old',$2)`, id, created); err != nil {
		t.Fatal(err)
	}
	check(customer, created.Add(4*time.Hour))
	latest := insert("incoming", "contact", false, 5)
	check(latest, latest)
	// Read markers / technical row writes must not advance business activity.
	db.MustExec(`UPDATE conversations SET updated_at=NOW() WHERE id=$1`, id)
	check(latest, latest)
}
