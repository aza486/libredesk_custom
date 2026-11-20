package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/knadh/stuffbin"
)

func TestCustomerSupportInstallUpgradeLifecycle(t *testing.T) {
	db := testutil.NewDB(t, "support_install_upgrade")
	db.MustExec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	schema, err := filepath.Abs("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", schema+":/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIBREDESK_SYSTEM_USER_PASSWORD", "Test-only-install-39!Password")
	if err := install(t.Context(), db, fs, true, false); err != nil {
		t.Fatal(err)
	}
	if err := checkSystemData(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var original string
	snapshot := func() string {
		t.Helper()
		var state string
		if err := db.Get(&state, `SELECT jsonb_build_object('roles',(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM roles r),'members',(SELECT jsonb_agg(to_jsonb(r) ORDER BY user_id,role_id) FROM user_roles r))::text`); err != nil {
			t.Fatal(err)
		}
		return state
	}
	upgrade(db, fs, false)
	original = snapshot()
	if err := install(t.Context(), db, fs, true, false); err != nil {
		t.Fatal(err)
	}
	upgrade(db, fs, false)
	if snapshot() != original {
		t.Fatal("repeated install/upgrade changed existing roles or memberships")
	}
	tx := db.MustBegin()
	tx.MustExec(`SET TRANSACTION READ ONLY`)
	if err := role.CheckCustomerSupportRole(t.Context(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	// Both existing-schema gates must initialize a missing technical role.
	db.MustExec(`DELETE FROM roles WHERE name='Kundensupport'`)
	if err := install(t.Context(), db, fs, true, false); err != nil {
		t.Fatal(err)
	}
	if err := checkSystemData(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`DELETE FROM roles WHERE name='Kundensupport'`)
	upgrade(db, fs, false)
	if err := checkSystemData(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var memberships, teams int
	if err := db.Get(&memberships, `SELECT count(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE r.name='Kundensupport'`); err != nil || memberships != 0 {
		t.Fatal(memberships, err)
	}
	if err := db.Get(&teams, `SELECT count(*) FROM teams`); err != nil || teams != 0 {
		t.Fatal(teams, err)
	}
	db.MustExec(`UPDATE roles SET permissions=ARRAY['conversations:read','messages:read'] WHERE name='Kundensupport'`)
	original = snapshot()
	if err := install(t.Context(), db, fs, true, false); err == nil || !strings.Contains(err.Error(), "service_emails:manage") {
		t.Fatal("existing rights silently widened", err)
	}
	if err := checkSystemData(t.Context(), db); err == nil {
		t.Fatal("preflight accepted missing right")
	}
	if snapshot() != original {
		t.Fatal("diagnosis modified existing roles")
	}
}
