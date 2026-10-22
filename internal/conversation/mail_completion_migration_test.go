package conversation

import (
	"github.com/abhinavxd/libredesk/internal/migrations"
	"testing"
)

func TestMailCompletionMigrationBackfillsVersionsWithoutChoosingSources(t *testing.T) {
	m, _, cid := completionFixture(t, "completion_migration")
	a := input(t, m, cid)
	b := input(t, m, cid)
	m.db.MustExec(`DROP TRIGGER libredesk_mail_input_clock ON conversation_messages; DROP TABLE conversation_mail_versions; UPDATE conversation_messages SET meta=meta-'mail_input_version'`)
	for i := 0; i < 2; i++ {
		if err := migrations.V2_14_0(m.db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	var av, bv string
	m.db.Get(&av, `SELECT meta->>'mail_input_version' FROM conversation_messages WHERE uuid=$1`, a)
	m.db.Get(&bv, `SELECT meta->>'mail_input_version' FROM conversation_messages WHERE uuid=$1`, b)
	if av != "1" || bv != "2" {
		t.Fatal(av, bv)
	}
	var version int
	m.db.Get(&version, `SELECT version FROM conversation_mail_versions WHERE conversation_id=$1`, cid)
	if version != 2 {
		t.Fatal(version)
	}
	input(t, m, cid)
	m.db.Get(&version, `SELECT version FROM conversation_mail_versions WHERE conversation_id=$1`, cid)
	if version != 3 {
		t.Fatal(version)
	}
}
