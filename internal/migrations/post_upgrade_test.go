package migrations_test

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
)

func TestPostUpgradeMigrationChainPreservesExistingRecords(t *testing.T) {
	db := testutil.NewDB(t, "post_upgrade_migrations")
	for _, migrate := range []func(*sqlx.DB) error{
		func(db *sqlx.DB) error { return migrations.V2_18_0(db, nil, nil) },
		func(db *sqlx.DB) error { return migrations.V2_19_0(db, nil, nil) },
	} {
		if err := migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	// Explicitly model the pre-2.20 schema, with real existing records.
	db.MustExec(`ALTER TABLE users DROP COLUMN IF EXISTS work_phone_number;
 INSERT INTO users(id,type,email,first_name) VALUES(101,'agent','migration@example.test','Existing');
 INSERT INTO inboxes(id,name,channel) VALUES(101,'Existing email','email');
 INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('Existing folder','customer',0,'[]');
 INSERT INTO roles(name,permissions) VALUES('Kundensupport','{}') ON CONFLICT DO NOTHING;`)
	const snapshot = `SELECT jsonb_build_object(
 'users',(SELECT jsonb_agg(to_jsonb(u)-'work_phone_number' ORDER BY id) FROM users u),
 'inboxes',(SELECT jsonb_agg(to_jsonb(i)-'default_signature_id' ORDER BY id) FROM inboxes i),
 'folders',(SELECT jsonb_agg(to_jsonb(f) ORDER BY id) FROM sidebar_folders f))::text`
	var before, after string
	if err := db.Get(&before, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, migrate := range []struct {
		name string
		fn   func(*sqlx.DB) error
	}{
		{"2.20", func(db *sqlx.DB) error { return migrations.V2_20_0(db, nil, nil) }},
		{"2.21", func(db *sqlx.DB) error { return migrations.V2_21_0(db, nil, nil) }},
		{"2.22", func(db *sqlx.DB) error { return migrations.V2_22_0(db, nil, nil) }},
		{"2.23", func(db *sqlx.DB) error { return migrations.V2_23_0(db, nil, nil) }},
	} {
		for range 2 {
			if err := migrate.fn(db); err != nil {
				t.Fatalf("migration %s: %v", migrate.name, err)
			}
		}
	}
	if err := db.Get(&after, snapshot); err != nil || before != after {
		t.Fatalf("migration chain changed existing records: %v", err)
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM users WHERE id=101 AND work_phone_number IS NULL`); err != nil || count != 1 {
		t.Fatal("business phone default changed", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM inboxes WHERE id=101 AND default_signature_id IS NULL`); err != nil || count != 1 {
		t.Fatal("signature default changed", count, err)
	}
	if err := db.Get(&count, `SELECT count(*) FROM roles r WHERE name IN ('Admin','Kundensupport')
 AND (SELECT count(*) FROM unnest(r.permissions) p WHERE p='service_emails:manage')=1`); err != nil || count != 2 {
		t.Fatal("service-address grant missing or duplicated", count, err)
	}
	db.MustExec(`INSERT INTO sidebar_folders(name,scope_type,scope_id,filters) VALUES('Service folder','service_mail',0,'[]')`)
	if err := migrations.V2_23_0(db, nil, nil); err != nil {
		t.Fatal("latest migration must preserve existing service folders", err)
	}
}
