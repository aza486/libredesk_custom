package role

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

func supportSnapshot(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	var state string
	if err := db.Get(&state, `SELECT jsonb_build_object('roles',(SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM roles r),'members',(SELECT jsonb_agg(to_jsonb(r) ORDER BY user_id,role_id) FROM user_roles r))::text`); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestCustomerSupportBootstrap(t *testing.T) {
	for i, extra := range []string{"", "messages:write", "conversations:add_system_tags"} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			db := testutil.NewDB(t, fmt.Sprintf("support_bootstrap_%d", i))
			if err := CheckCustomerSupportRole(t.Context(), db); err == nil {
				t.Fatal("missing role accepted")
			}
			if extra != "" {
				perms := append(customerSupportPermissions(), extra)
				db.MustExec(`INSERT INTO roles(id,name,description,permissions) VALUES(731,$1,'Existing', $2)`, models.RoleCustomerSupport, pq.Array(perms))
				db.MustExec(`INSERT INTO users(id,type,first_name) VALUES(41,'agent','Existing'); INSERT INTO user_roles(user_id,role_id) SELECT 41,731`)
			}
			if err := EnsureCustomerSupportRole(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			before := supportSnapshot(t, db)
			for range 2 {
				if err := EnsureCustomerSupportRole(t.Context(), db); err != nil {
					t.Fatal(err)
				}
			}
			if before != supportSnapshot(t, db) {
				t.Fatal("existing role or memberships modified")
			}
			if err := CheckCustomerSupportRole(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			var members, teams int
			db.Get(&members, `SELECT count(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE r.name=$1`, models.RoleCustomerSupport)
			want := 0
			if extra != "" {
				want = 1
			}
			if members != want {
				t.Fatal("automatic membership", members)
			}
			db.Get(&teams, `SELECT count(*) FROM teams`)
			if teams != 0 {
				t.Fatal("team created")
			}
			if extra == "" {
				var perms pq.StringArray
				if err := db.Get(&perms, `SELECT permissions FROM roles WHERE name=$1`, models.RoleCustomerSupport); err != nil {
					t.Fatal(err)
				}
				if strings.Join(perms, ",") != strings.Join(append(customerSupportPermissions(), authzmodels.PermMessageTemplatesSupport), ",") {
					t.Fatal(perms)
				}
			}
		})
	}
}

func TestCustomerSupportBootstrapDivergentPermissions(t *testing.T) {
	for i, missing := range customerSupportPermissions() {
		t.Run(missing, func(t *testing.T) {
			db := testutil.NewDB(t, fmt.Sprintf("support_divergent_%d", i))
			var perms []string
			for _, p := range customerSupportPermissions() {
				if p != missing {
					perms = append(perms, p)
				}
			}
			db.MustExec(`INSERT INTO roles(id,name,permissions) VALUES(731,$1,$2);`, models.RoleCustomerSupport, pq.Array(perms))
			db.MustExec(`INSERT INTO users(id,type,first_name) VALUES(41,'agent','Existing'); INSERT INTO user_roles(user_id,role_id) SELECT 41,731`)
			before := supportSnapshot(t, db)
			for _, check := range []func(context.Context, *sqlx.DB) error{EnsureCustomerSupportRole, func(ctx context.Context, db *sqlx.DB) error { return CheckCustomerSupportRole(ctx, db) }} {
				if err := check(t.Context(), db); err == nil || !strings.Contains(err.Error(), missing) {
					t.Fatalf("missing diagnosis: %v", err)
				}
			}
			if before != supportSnapshot(t, db) {
				t.Fatal("existing permissions expanded")
			}
			tx := db.MustBegin()
			defer tx.Rollback()
			tx.MustExec(`SET TRANSACTION READ ONLY`)
			if err := CheckCustomerSupportRole(t.Context(), tx); err == nil {
				t.Fatal("readonly check missed divergence")
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
			// Existing role descriptions are nullable. Administrators must be able
			// to repair a diagnosed role explicitly without replacing its ID.
			lo := logf.New(logf.Opts{})
			m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m.Update(731, models.Role{Name: models.RoleCustomerSupport, Description: "Explicit repair", Permissions: customerSupportPermissions()}); err != nil {
				t.Fatal(err)
			}
			if err := CheckCustomerSupportRole(t.Context(), db); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCustomerSupportBootstrapParallelAndRollback(t *testing.T) {
	db := testutil.NewDB(t, "support_parallel")
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- EnsureCustomerSupportRole(t.Context(), db) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM roles WHERE name=$1`, models.RoleCustomerSupport)
	if count != 1 {
		t.Fatal(count)
	}
	db.MustExec(`DELETE FROM roles WHERE name='Kundensupport'; CREATE FUNCTION reject_support_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.name='Kundensupport' THEN RAISE EXCEPTION 'synthetic support commit failure'; END IF; RETURN NEW; END $$; CREATE CONSTRAINT TRIGGER reject_support_commit AFTER INSERT ON roles DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_support_commit()`)
	before := supportSnapshot(t, db)
	if err := EnsureCustomerSupportRole(t.Context(), db); err == nil || !strings.Contains(err.Error(), "synthetic support commit failure") {
		t.Fatal(err)
	}
	if before != supportSnapshot(t, db) {
		t.Fatal("commit failure left data")
	}
}

func TestCustomerSupportRoleProtection(t *testing.T) {
	db := testutil.NewDB(t, "support_protection")
	if err := EnsureCustomerSupportRole(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	var id int
	db.Get(&id, `SELECT id FROM roles WHERE name=$1`, models.RoleCustomerSupport)
	ordinary, err := m.Create(models.Role{Name: "Ordinary", Description: "Ordinary", Permissions: []string{"conversations:read"}})
	if err != nil {
		t.Fatal(err)
	}
	before := supportSnapshot(t, db)
	if err := m.Delete(id); err == nil {
		t.Fatal("protected delete")
	}
	if _, err := m.Create(models.Role{Name: models.RoleCustomerSupport, Permissions: customerSupportPermissions()}); err == nil {
		t.Fatal("reserved create")
	}
	for _, tc := range []struct {
		id    int
		name  string
		perms []string
	}{
		{id, "Renamed", customerSupportPermissions()},
		{ordinary.ID, models.RoleCustomerSupport, customerSupportPermissions()},
		{id, models.RoleCustomerSupport, []string{"conversations:read"}},
	} {
		if _, err := m.Update(tc.id, models.Role{Name: tc.name, Permissions: tc.perms}); err == nil {
			t.Fatal("unsafe role update", tc)
		}
	}
	if before != supportSnapshot(t, db) {
		t.Fatal("rejected write changed state")
	}
	if _, err := m.Update(id, models.Role{Name: models.RoleCustomerSupport, Description: "Explicit administrative update", Permissions: append(customerSupportPermissions(), "messages:write")}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Update(ordinary.ID, models.Role{Name: "Renamed ordinary", Description: "Updated", Permissions: []string{"messages:read"}}); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(ordinary.ID); err != nil {
		t.Fatal(err)
	}
}

func TestCustomerSupportBootstrapAmbiguousIdentity(t *testing.T) {
	db := testutil.NewDB(t, "support_duplicate_identity")
	db.MustExec(`ALTER TABLE roles DROP CONSTRAINT roles_name_key; INSERT INTO roles(name,permissions) VALUES('Kundensupport',ARRAY['conversations:read']),('Kundensupport',ARRAY['messages:read'])`)
	before := supportSnapshot(t, db)
	if err := EnsureCustomerSupportRole(t.Context(), db); err == nil || !strings.Contains(err.Error(), "found 2") {
		t.Fatal(err)
	}
	if before != supportSnapshot(t, db) {
		t.Fatal("conflicting roles modified")
	}
}
