package tag

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func tagSnapshot(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	var snapshot string
	if err := db.Get(&snapshot, `SELECT jsonb_build_object(
 'tags',(SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM tags t),
 'links',(SELECT jsonb_agg(to_jsonb(t) ORDER BY conversation_id,tag_id) FROM conversation_tags t),
 'sequence',(SELECT jsonb_build_object('last_value',last_value,'is_called',is_called) FROM tags_id_seq))::text`); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestSystemTagBootstrapInstallAndIdempotency(t *testing.T) {
	for index, setup := range []string{
		"", // fresh schema, no tags
		`INSERT INTO tags(id,name) VALUES(8,'🏢Intern'),(27,'Ordinary')`,
		`ALTER SEQUENCE tags_id_seq RESTART WITH 1000`,
		`SELECT setval('tags_id_seq',1000,true)`,
	} {
		t.Run(fmt.Sprintf("case_%d", index), func(t *testing.T) {
			db := testutil.NewDB(t, fmt.Sprintf("system_tags_init_%d", index))
			if setup != "" {
				db.MustExec(setup)
			}
			var originals []struct {
				ID  int
				Row string
			}
			if err := db.Select(&originals, `SELECT id,to_jsonb(t)::text AS row FROM tags t`); err != nil {
				t.Fatal(err)
			}
			if err := CheckSystemTags(t.Context(), db); err == nil {
				t.Fatal("preflight accepted missing tags")
			}
			if err := EnsureSystemTags(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			if err := CheckSystemTags(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			for _, original := range originals {
				var after string
				if err := db.Get(&after, `SELECT to_jsonb(t)::text FROM tags t WHERE id=$1`, original.ID); err != nil || after != original.Row {
					t.Fatalf("initial bootstrap changed existing tag %d: %v", original.ID, err)
				}
			}
			for _, required := range requiredSystemTags {
				var name string
				if err := db.Get(&name, `SELECT name FROM tags WHERE id=$1`, required.ID); err != nil || name != required.Name {
					t.Fatalf("tag %d: %q %v", required.ID, name, err)
				}
			}
			// Include actual ticket associations and all tag timestamps in the
			// byte-for-byte state comparison on an existing installation.
			db.MustExec(`INSERT INTO users(id,type,email,first_name) VALUES(101,'contact','tag-test@example.test','Test');
 INSERT INTO inboxes(id,name,channel) VALUES(101,'Test','email');
 INSERT INTO conversations(contact_id,inbox_id,status_id) SELECT 101,101,id FROM conversation_statuses WHERE name='Open';
 INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,8 FROM conversations;`)
			before := tagSnapshot(t, db)
			for range 2 {
				if err := EnsureSystemTags(t.Context(), db); err != nil {
					t.Fatal(err)
				}
				if err := CheckSystemTags(t.Context(), db); err != nil {
					t.Fatal(err)
				}
			}
			if after := tagSnapshot(t, db); after != before {
				t.Fatalf("existing data or sequence changed: %s -> %s", before, after)
			}
			lo := logf.New(logf.Opts{})
			manager, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
			if err != nil {
				t.Fatal(err)
			}
			created, err := manager.Create("New ordinary tag")
			want := []int{15, 28, 1000, 1001}[index]
			if err != nil || created.ID != want {
				t.Fatalf("normal tag creation: ID %d want %d, %v", created.ID, want, err)
			}
		})
	}
}

func TestSystemTagBootstrapConflictsRollback(t *testing.T) {
	for index, tc := range []struct{ setup, message string }{
		{`INSERT INTO tags(id,name) VALUES(8,'Ordinary')`, `ID 8 is "Ordinary"`},
		{`INSERT INTO tags(id,name) VALUES(23,'🏢Intern')`, `"🏢Intern" has ID 23`},
		{`INSERT INTO tags(id,name) VALUES(12,'Spam')`, `ID 12 is "Spam"`},
		{`INSERT INTO tags(id,name) VALUES(8,'Ordinary'),(23,'🏢Intern')`, `expected`},
	} {
		t.Run(fmt.Sprintf("case_%d", index), func(t *testing.T) {
			db := testutil.NewDB(t, fmt.Sprintf("system_tags_conflict_%d", index))
			db.MustExec(tc.setup)
			before := tagSnapshot(t, db)
			for _, check := range []func(context.Context, *sqlx.DB) error{CheckSystemTags, EnsureSystemTags} {
				err := check(t.Context(), db)
				if err == nil || !strings.Contains(err.Error(), tc.message) {
					t.Fatalf("wrong error: %v", err)
				}
				if after := tagSnapshot(t, db); after != before {
					t.Fatal("conflicting initialization changed data or sequence")
				}
			}
		})
	}
}

func TestSystemTagBootstrapSequenceFailureRollback(t *testing.T) {
	db := testutil.NewDB(t, "system_tags_sequence_failure")
	db.MustExec(`ALTER SEQUENCE tags_id_seq MAXVALUE 14`)
	before := tagSnapshot(t, db)
	if err := EnsureSystemTags(t.Context(), db); err == nil {
		t.Fatal("expected sequence exhaustion error")
	}
	if after := tagSnapshot(t, db); after != before {
		t.Fatal("sequence failure left partial tags or sequence changes")
	}
}

func TestSystemTagBootstrapCommitFailureRollsBackSequence(t *testing.T) {
	db := testutil.NewDB(t, "system_tags_commit_failure")
	db.MustExec(`CREATE FUNCTION reject_system_tag_commit() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'synthetic deferred commit failure'; END $$;
 CREATE CONSTRAINT TRIGGER reject_system_tag_commit AFTER INSERT ON tags
 DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_system_tag_commit();`)
	before := tagSnapshot(t, db)
	if err := EnsureSystemTags(t.Context(), db); err == nil || !strings.Contains(err.Error(), "synthetic deferred commit failure") {
		t.Fatalf("expected deferred commit failure, got %v", err)
	}
	if after := tagSnapshot(t, db); before != after {
		t.Fatal("commit failure left tag rows or advanced sequence state")
	}
}

func TestSystemTagBootstrapInvalidatesCachedIDs(t *testing.T) {
	for _, complete := range []bool{false, true} {
		t.Run(fmt.Sprint(complete), func(t *testing.T) {
			db := testutil.NewDB(t, fmt.Sprintf("system_tags_cached_ids_%v", complete))
			db.MustExec(`ALTER SEQUENCE tags_id_seq CACHE 20`)
			conn, err := db.Connx(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			var id int
			if err := conn.GetContext(t.Context(), &id, `SELECT nextval('tags_id_seq')`); err != nil || id != 1 {
				t.Fatalf("cannot reserve cached IDs: %d %v", id, err)
			}
			if complete {
				for _, required := range requiredSystemTags {
					db.MustExec(`INSERT INTO tags(id,name) VALUES($1,$2)`, required.ID, required.Name)
				}
			}
			if err := EnsureSystemTags(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			if err := conn.GetContext(t.Context(), &id, `INSERT INTO tags(name) VALUES('Cached connection tag') RETURNING id`); err != nil || id != 21 {
				t.Fatalf("stale cached IDs were reused: %d %v", id, err)
			}
		})
	}
}

func TestSystemTagBootstrapIndividualMissingTags(t *testing.T) {
	db := testutil.NewDB(t, "system_tags_individual")
	if err := EnsureSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	for _, required := range requiredSystemTags {
		t.Run(fmt.Sprint(required.ID), func(t *testing.T) {
			db.MustExec(`DELETE FROM tags WHERE id=$1`, required.ID)
			var before, after string
			const existing = `SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM tags t WHERE id<>$1`
			if err := db.Get(&before, existing, required.ID); err != nil {
				t.Fatal(err)
			}
			if err := EnsureSystemTags(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			if err := CheckSystemTags(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			if err := db.Get(&after, existing, required.ID); err != nil || before != after {
				t.Fatalf("existing tags changed: %v", err)
			}
		})
	}
}

func TestSystemTagPreflightReadOnly(t *testing.T) {
	db := testutil.NewDB(t, "system_tags_read_only")
	db.SetMaxOpenConns(1)
	before := tagSnapshot(t, db)
	db.MustExec(`SET default_transaction_read_only=on`)
	if err := CheckSystemTags(t.Context(), db); err == nil || !strings.Contains(err.Error(), "missing system tag") {
		t.Fatalf("wrong read-only check result: %v", err)
	}
	if after := tagSnapshot(t, db); before != after {
		t.Fatal("preflight changed data")
	}
	db.MustExec(`SET default_transaction_read_only=off`)
	if err := EnsureSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`SET default_transaction_read_only=on`)
	if err := CheckSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}

func TestSystemTagBootstrapConcurrent(t *testing.T) {
	db := testutil.NewDB(t, "system_tags_concurrent")
	db.SetMaxOpenConns(12)
	errs := make(chan error, 32)
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() { errs <- EnsureSystemTags(t.Context(), db) })
	}
	wg.Wait()
	if err := CheckSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	for index := range 8 {
		wg.Go(func() { errs <- EnsureSystemTags(t.Context(), db) })
		wg.Go(func() {
			_, err := db.Exec(`INSERT INTO tags(name) VALUES($1)`, fmt.Sprintf("Concurrent regular %d", index))
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM tags`); err != nil || count != 13 {
		t.Fatalf("unexpected tags: %d %v", count, err)
	}
}

func TestSystemTagBootstrapRespectsTagWriteLock(t *testing.T) {
	db := testutil.NewDB(t, "system_tags_lock")
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO tags(id,name) VALUES(8,'Ordinary')`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()
	if err := EnsureSystemTags(ctx, db); err == nil {
		t.Fatal("bootstrap ignored concurrent tag write lock")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := EnsureSystemTags(t.Context(), db); err == nil || !strings.Contains(err.Error(), "Ordinary") {
		t.Fatalf("bootstrap ignored committed conflicting tag: %v", err)
	}
}
