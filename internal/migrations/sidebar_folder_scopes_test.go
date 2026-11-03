package migrations_test

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestMyInboxScopeMigration(t *testing.T) {
	db := testutil.NewDB(t, "my_inbox_scope_migration")
	if err := migrations.V2_18_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters,sort_order,show_count) VALUES('Existing','customer',0,'[]',3,false)`)
	var before, after string
	const snapshot = `SELECT to_jsonb(f)::text FROM sidebar_folders f WHERE name='Existing'`
	if err := db.Get(&before, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('Before','my_inbox',101,'[]')`); err == nil {
		t.Fatal("old schema unexpectedly accepted my_inbox")
	}
	for i := 0; i < 2; i++ {
		if err := migrations.V2_19_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&after, snapshot); err != nil || before != after {
		t.Fatal("scope upgrade modified existing folders")
	}
	if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('New','my_inbox',101,'[]')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('Invalid','my_inbox',0,'[]')`); err == nil {
		t.Fatal("zero employee ID accepted")
	}
	if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('Invalid','unknown',101,'[]')`); err == nil {
		t.Fatal("unknown scope accepted")
	}
}
