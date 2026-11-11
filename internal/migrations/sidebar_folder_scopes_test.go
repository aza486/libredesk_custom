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

func TestRedistributedScopeMigration(t *testing.T) {
	db := testutil.NewDB(t, "redistributed_scope_migration")
	if err := migrations.V2_18_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := migrations.V2_19_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters,sort_order,show_count) VALUES('Existing customer','customer',0,'[]',3,false),('Existing team','team',77,'[]',0,true),('Existing personal','personal',78,'[]',0,true),('Existing mine','my_inbox',79,'[]',0,true)`)
	var before, after string
	const snapshot = `SELECT jsonb_agg(to_jsonb(f) ORDER BY id)::text FROM sidebar_folders f`
	if err := db.Get(&before, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('Before','redistributed',0,'[]')`); err == nil {
		t.Fatal("old constraints accepted new scope")
	}
	for range 2 {
		if err := migrations.V2_20_0(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Get(&after, snapshot); err != nil || after != before {
		t.Fatalf("existing folders changed: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('New','redistributed',0,'[]')`); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"('Invalid','redistributed',1,'[]')", "('Invalid','customer',1,'[]')", "('Invalid','my_inbox',0,'[]')", "('Invalid','other',0,'[]')"} {
		if _, err := db.Exec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES` + value); err == nil {
			t.Fatalf("invalid scope accepted: %s", value)
		}
	}
}
