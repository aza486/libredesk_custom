package conversation

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestConversationListExactUnreadCount(t *testing.T) {
	db := testutil.NewPersonalDB(t, "exact_unread_count")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: folderUsers{}, settingsStore: personalSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var id int
	if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,subject)
        VALUES(105,102,101,'Unread count') RETURNING id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	// Also exercise the last-seen cutoff and continuity-email exclusion.
	db.MustExec(`INSERT INTO conversation_last_seen(conversation_id,user_id,last_seen_at) VALUES($1,104,'2026-01-01')`, id)
	db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,created_at)
        SELECT $1,'incoming','sent',105,'contact','Unread','2026-01-02'::timestamptz FROM generate_series(1,125)`, id)
	db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,created_at,meta)
        VALUES($1,'incoming','sent',105,'contact','Seen','2025-12-31',NULL),
              ($1,'incoming','sent',105,'contact','Continuity','2026-01-02','{"continuity_email":true}')`, id)
	rows, err := m.GetAllConversationsList(104, "", "", "[]", 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].UnreadMessageCount != 125 {
		t.Fatalf("expected one conversation with 125 unread messages, got %+v", rows)
	}
	rows, err = m.GetAllConversationsList(103, "", "", "[]", 1, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].UnreadMessageCount != 126 {
		t.Fatalf("expected 126 unread messages for agent without last-seen, got %+v", rows)
	}
}
