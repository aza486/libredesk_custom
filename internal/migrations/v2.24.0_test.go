package migrations_test

import (
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"testing"
)

func TestContactlessInternalMigration(t *testing.T) {
	db := testutil.NewPersonalDB(t, "contactless_migration")
	// Reconstruct the old NOT NULL boundary with an existing ticket/message.
	db.MustExec(`ALTER TABLE conversations DROP CONSTRAINT conversations_contact_required;
 ALTER TABLE conversations ALTER COLUMN contact_id SET NOT NULL;
 INSERT INTO conversations(id,contact_id,inbox_id,status_id,subject) VALUES(901,105,102,101,'Untouched');
 INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,content) VALUES(901,101,'agent','outgoing','sent','Existing mail')`)
	var before, after string
	snapshot := `SELECT json_build_object('ticket',row_to_json(c),'messages',(SELECT json_agg(row_to_json(m)) FROM conversation_messages m WHERE conversation_id=c.id))::text FROM conversations c WHERE id=901`
	db.Get(&before, snapshot)
	for range 2 {
		if err := migrations.V2_24_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	db.Get(&after, snapshot)
	if before != after {
		t.Fatal("migration changed existing business data")
	}
	for _, attrs := range []string{`{}`, `{"internal":true}`, `{"private":true}`, `{"internal":"true","private":true}`} {
		if _, err := db.Exec(`INSERT INTO conversations(inbox_id,status_id,custom_attributes) VALUES(102,101,$1)`, attrs); err == nil {
			t.Fatal("contactless unprotected conversation accepted", attrs)
		}
	}
	if _, err := db.Exec(`INSERT INTO conversations(inbox_id,status_id,custom_attributes) VALUES(102,101,'{"internal":true,"private":true}')`); err != nil {
		t.Fatal(err)
	}
}

func TestContactlessInternalMigrationRollback(t *testing.T) {
	db := testutil.NewPersonalDB(t, "contactless_rollback")
	db.MustExec(`ALTER TABLE conversations DROP CONSTRAINT conversations_contact_required;
 INSERT INTO conversations(inbox_id,status_id) VALUES(102,101)`)
	if err := migrations.V2_24_0(db, nil, nil); err == nil {
		t.Fatal("inconsistent preexisting row must stop migration")
	}
	var exists bool
	db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='conversations'::regclass AND conname='conversations_contact_required')`)
	if exists {
		t.Fatal("failed migration left partial constraint")
	}
}
