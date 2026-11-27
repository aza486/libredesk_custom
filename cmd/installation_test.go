package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

func installationFixture(t *testing.T, name string) (*sqlx.DB, stuffbin.FileSystem) {
	t.Helper()
	db := testutil.NewDB(t, name)
	db.MustExec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	schema, err := filepath.Abs("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", schema+":/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	return db, fs
}
func installForTest(t *testing.T, db *sqlx.DB, fs stuffbin.FileSystem) {
	t.Helper()
	if err := install(t.Context(), db, fs, true, false); err != nil {
		t.Fatal(err)
	}
}

// Snapshot every application table, including timestamps, encrypted settings,
// histories, memberships and private attributes. Sequence reservations are
// technical metadata and deliberately compared separately by bootstrap tests.
func installationSnapshot(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	var tables []string
	if err := db.Select(&tables, `SELECT tablename FROM pg_tables WHERE schemaname=current_schema() ORDER BY tablename`); err != nil {
		t.Fatal(err)
	}
	data := map[string]json.RawMessage{}
	for _, table := range tables {
		var raw []byte
		if err := db.Get(&raw, `SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]'::jsonb) FROM "`+table+`" t`); err != nil {
			t.Fatal(err)
		}
		data[table] = raw
	}
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestInstallationLifecycle(t *testing.T) {
	t.Setenv("LIBREDESK_SYSTEM_USER_PASSWORD", "Synthetic-Install-43!Password")
	db, fs := installationFixture(t, "install_lifecycle")
	if err := prepareApplicationDatabase(t.Context(), db); err == nil {
		t.Fatal("empty database accepted at startup")
	}
	installForTest(t, db, fs)
	if err := checkInstalledDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if err := prepareApplicationDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var history []byte
	db.Get(&history, `SELECT value FROM settings WHERE key='migrations'`)
	var versions []string
	json.Unmarshal(history, &versions)
	if len(versions) != len(migList) || versions[len(versions)-1] != "v2.25.0" {
		t.Fatal("install did not run full registered chain")
	}
	before := installationSnapshot(t, db)
	// Existing passwords must not be replaced, even by another env password.
	t.Setenv("LIBREDESK_SYSTEM_USER_PASSWORD", "Different-But-Not-Applied-58!")
	installForTest(t, db, fs)
	if err := upgrade(db, fs, false); err != nil {
		t.Fatal(err)
	}
	if err := install(t.Context(), db, fs, false, false); err == nil {
		t.Fatal("destructive install accepted over existing schema")
	}
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("repeated lifecycle changed stored rows")
	}
}

func TestInstallationInterruptedBaseline(t *testing.T) {
	db, fs := installationFixture(t, "install_baseline_repair")
	if err := installSchema(db, fs); err != nil {
		t.Fatal(err)
	}
	if err := prepareApplicationDatabase(t.Context(), db); err == nil {
		t.Fatal("unmigrated baseline accepted at startup")
	}
	installForTest(t, db, fs)
	if err := checkInstalledDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
}

func TestInstallationRepairsMissingMandatoryData(t *testing.T) {
	for i, tc := range []struct{ name, sql string }{
		{"system account", `DELETE FROM users WHERE email='System'`},
		{"support role", `DELETE FROM roles WHERE name='Kundensupport'`},
		{"system tag", `DELETE FROM tags WHERE id=12`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, fs := installationFixture(t, fmt.Sprintf("install_repair_%d", i))
			installForTest(t, db, fs)
			db.MustExec(tc.sql)
			installForTest(t, db, fs)
			if err := checkInstalledDatabase(t.Context(), db); err != nil {
				t.Fatal(err)
			}
			var members, teams int
			db.Get(&members, `SELECT count(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE r.name='Kundensupport'`)
			db.Get(&teams, `SELECT count(*) FROM teams`)
			if members != 0 || teams != 0 {
				t.Fatal("bootstrap assigned employees or created teams")
			}
		})
	}
}

func TestInstallationRejectsDamagedState(t *testing.T) {
	for i, tc := range []struct{ name, sql, diagnostic string }{
		{"missing table", `DROP TABLE conversation_messages CASCADE`, "missing required tables"},
		{"missing runtime table", `DROP TABLE manual_review_annotations CASCADE`, "missing required tables"},
		{"missing column", `ALTER TABLE users DROP COLUMN work_phone_number`, "missing columns"},
		{"missing trigger", `DROP TRIGGER libredesk_mail_input_clock ON conversation_messages`, "triggers"},
		{"disabled trigger", `ALTER TABLE conversation_messages DISABLE TRIGGER libredesk_mail_input_clock`, "triggers"},
		{"missing SLA index with duplicate legacy data", `DROP INDEX index_applied_slas_unique_pending_per_conv;
 INSERT INTO users(email,type,first_name) VALUES('sla-contact@example.test','contact','SLA');
 INSERT INTO inboxes(name,channel) VALUES('SLA fixture','email');
 INSERT INTO conversations(contact_id,inbox_id,status_id) SELECT u.id,i.id,s.id FROM users u CROSS JOIN inboxes i CROSS JOIN conversation_statuses s WHERE u.email='sla-contact@example.test' AND i.name='SLA fixture' AND s.name='Open';
 INSERT INTO sla_policies(name,first_response_time,resolution_time) VALUES('SLA fixture','1h','8h');
 INSERT INTO applied_slas(conversation_id,sla_policy_id) SELECT c.id,p.id FROM conversations c CROSS JOIN sla_policies p WHERE p.name='SLA fixture';
 INSERT INTO applied_slas(conversation_id,sla_policy_id) SELECT c.id,p.id FROM conversations c CROSS JOIN sla_policies p WHERE p.name='SLA fixture';`, "unique indexes"},
		{"missing constraint", `ALTER TABLE conversations DROP CONSTRAINT conversations_contact_required`, "constraints"},
		{"missing setting", `DELETE FROM settings WHERE key='app.lang'`, "settings"},
		{"wrong system type", `UPDATE users SET type='contact' WHERE email='System'`, "invalid system account"},
		{"missing system role", `DELETE FROM user_roles WHERE user_id IN(SELECT id FROM users WHERE email='System')`, "invalid system account"},
		{"conflicting system identity", `INSERT INTO users(email,type,first_name) VALUES('system','contact','Conflict')`, "ambiguous system identity"},
		{"conflicting tag", `UPDATE tags SET name='Conflict' WHERE id=8`, "conflict"},
		{"missing role permission", `UPDATE roles SET permissions=ARRAY['conversations:read'] WHERE name='Kundensupport'`, "required permissions"},
		{"interrupted migration", `INSERT INTO settings(key,value) VALUES('migration_in_progress','{"version":"v2.24.0"}')`, "unfinished migration"},
		{"rewound service-mail migration", `UPDATE settings SET value='["v2.22.0"]' WHERE key='migrations'`, "behind schema"},
		{"rewound valid history", `UPDATE settings SET value='["v2.21.0"]' WHERE key='migrations'`, "behind schema"},
		{"history removed", `DELETE FROM settings WHERE key='migrations'`, "history is missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, fs := installationFixture(t, fmt.Sprintf("install_damage_%d", i))
			installForTest(t, db, fs)
			db.MustExec(tc.sql)
			before := installationSnapshot(t, db)
			err := install(t.Context(), db, fs, true, false)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), tc.diagnostic) {
				t.Fatalf("wanted %s, got %v", tc.diagnostic, err)
			}
			if err := prepareApplicationDatabase(t.Context(), db); err == nil {
				t.Fatal("damaged state accepted at startup")
			}
			if after := installationSnapshot(t, db); after != before {
				t.Fatal("rejected lifecycle changed data")
			}
		})
	}
}

func TestMigrationHistoryValidation(t *testing.T) {
	db, fs := installationFixture(t, "install_history")
	installForTest(t, db, fs)
	for _, raw := range []string{`null`, `[]`, `{}`, `"v2.24.0"`, `[1]`, `["bad"]`, `["v999.0.0"]`, `["v2.24.0","v2.24.0"]`, `["v2.23.0","v2.22.0"]`, `["v2.21.0","v2.24.0"]`} {
		t.Run(raw, func(t *testing.T) {
			db.MustExec(`UPDATE settings SET value=$1::jsonb WHERE key='migrations'`, raw)
			before := installationSnapshot(t, db)
			if err := install(t.Context(), db, fs, true, false); err == nil {
				t.Fatal("invalid history accepted")
			}
			if after := installationSnapshot(t, db); after != before {
				t.Fatal("invalid history modified data")
			}
		})
	}
	// Older releases may have recorded only a contiguous suffix. Keep valid
	// legacy histories without inventing records for earlier migrations.
	db.MustExec(`UPDATE settings SET value='["v2.23.0","v2.24.0","v2.25.0"]' WHERE key='migrations'`)
	before := installationSnapshot(t, db)
	installForTest(t, db, fs)
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("valid legacy history rewritten")
	}
}

func TestInstallationConcurrent(t *testing.T) {
	db, fs := installationFixture(t, "install_parallel")
	db.SetMaxOpenConns(1)
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- install(context.Background(), db, fs, true, false) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if db.Stats().MaxOpenConnections != 1 {
		t.Fatal("connection limit not restored")
	}
	if err := checkInstalledDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM users WHERE email='System'`)
	if count != 1 {
		t.Fatal("duplicate system account")
	}
}

func TestInstallationSchemaRollback(t *testing.T) {
	db, _ := installationFixture(t, "install_schema_rollback")
	dir := t.TempDir()
	schema := filepath.Join(dir, "broken.sql")
	if err := os.WriteFile(schema, []byte(`CREATE TABLE settings(key text); SELECT 1/0;`), 0600); err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", schema+":/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	if err := install(t.Context(), db, fs, true, false); err == nil {
		t.Fatal("broken schema accepted")
	}
	installed, err := checkSchema(db)
	if err != nil || installed {
		t.Fatal("partial schema survived rollback", installed, err)
	}
}

func TestMigrationInterruptionIsNotReplayed(t *testing.T) {
	db, fs := installationFixture(t, "install_interruption")
	if err := installSchema(db, fs); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO settings(key,value) VALUES('migrations','["v0.3.0"]');
 UPDATE roles SET permissions=array_remove(permissions,'ai:manage') WHERE name='Admin';
 CREATE FUNCTION reject_migration_index() RETURNS event_trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic migration interruption'; END $$;
 CREATE EVENT TRIGGER reject_migration_index ON ddl_command_start WHEN TAG IN ('CREATE INDEX') EXECUTE FUNCTION reject_migration_index()`)
	err := install(t.Context(), db, fs, true, false)
	if err == nil || !strings.Contains(err.Error(), "v0.4.0 failed") {
		t.Fatal("failed migration not diagnosed", err)
	}
	var marker bool
	db.Get(&marker, `SELECT EXISTS(SELECT 1 FROM settings WHERE key='migration_in_progress' AND value->>'version'='v0.4.0')`)
	if !marker {
		t.Fatal("crash marker lost")
	}
	// The historical migration committed this first statement before its later
	// DDL failed. A retry must not guess that the earlier writes rolled back.
	var granted bool
	db.Get(&granted, `SELECT 'ai:manage'=ANY(permissions) FROM roles WHERE name='Admin'`)
	if !granted {
		t.Fatal("test did not exercise a partially committed historical migration")
	}
	db.MustExec(`DROP EVENT TRIGGER reject_migration_index; DROP FUNCTION reject_migration_index()`)
	before := installationSnapshot(t, db)
	if err := upgrade(db, fs, false); err == nil || !strings.Contains(err.Error(), "unfinished migration") {
		t.Fatal("interrupted migration replayed", err)
	}
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("interruption retry changed data")
	}
}

func TestInstallationExistingBusinessDataIntegrity(t *testing.T) {
	db, fs := installationFixture(t, "install_business_integrity")
	installForTest(t, db, fs)
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Existing integration',ARRAY['conversations:read','messages:write','conversations:add_system_tags']);
 INSERT INTO users(email,type,first_name,last_name,enabled) VALUES('employee@example.test','agent','Existing','Employee',true),('customer@example.test','contact','Existing','Customer',true);
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.email='employee@example.test' AND r.name='Existing integration';
 INSERT INTO teams(name,conversation_assignment_type) VALUES('Existing team','Manual');
 INSERT INTO team_members(user_id,team_id) SELECT u.id,t.id FROM users u CROSS JOIN teams t WHERE u.email='employee@example.test' AND t.name='Existing team';
 INSERT INTO templates(name,type,subject,body) VALUES('Existing signature','email_outgoing','Existing subject','Existing signature body');
 INSERT INTO inboxes(name,channel,"from",default_signature_id) SELECT 'Existing mailbox','email','mailbox@example.test',id FROM templates WHERE name='Existing signature';
 INSERT INTO sla_policies(name,description,first_response_time,resolution_time) VALUES('Existing SLA','Existing SLA','1h','8h');
 INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes,assigned_team_id)
 SELECT u.id,i.id,s.id,'Existing customer',jsonb_build_object('customer_visibility',false,'visible_users',jsonb_build_array(a.id),'visibility_managers',jsonb_build_array(a.id)),t.id
 FROM users u CROSS JOIN users a CROSS JOIN inboxes i CROSS JOIN conversation_statuses s CROSS JOIN teams t WHERE u.email='customer@example.test' AND a.email='employee@example.test' AND i.name='Existing mailbox' AND s.name='Open' AND t.name='Existing team';
 INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes)
 SELECT NULL,i.id,s.id,'Existing private internal',jsonb_build_object('private',true,'internal',true,'creator_id',a.id,'visible_users',jsonb_build_array(a.id),'_spam_previous_state',jsonb_build_object('status_id',s.id))
 FROM users a CROSS JOIN inboxes i CROSS JOIN conversation_statuses s WHERE a.email='employee@example.test' AND i.name='Existing mailbox' AND s.name='Open';
 INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,content)
 SELECT c.id,u.id,'contact','incoming','received','Existing customer message' FROM conversations c CROSS JOIN users u WHERE c.subject='Existing customer' AND u.email='customer@example.test';
 INSERT INTO conversation_messages(conversation_id,sender_id,sender_type,type,status,content)
 SELECT c.id,u.id,'agent','internal','stored','Existing stored-only internal message' FROM conversations c CROSS JOIN users u WHERE c.subject='Existing private internal' AND u.email='employee@example.test';
 INSERT INTO applied_slas(conversation_id,sla_policy_id,status) SELECT c.id,p.id,'pending' FROM conversations c CROSS JOIN sla_policies p WHERE c.subject='Existing customer' AND p.name='Existing SLA';
 INSERT INTO applied_slas(conversation_id,sla_policy_id,status) SELECT c.id,p.id,'met' FROM conversations c CROSS JOIN sla_policies p WHERE c.subject='Existing customer' AND p.name='Existing SLA';
 INSERT INTO service_email_addresses(address) VALUES('existing-service@example.test');
 INSERT INTO webhooks(name,url,events,secret) VALUES('Existing webhook','https://example.invalid/hook',ARRAY['message.created']::webhook_event[],'synthetic-only');
 UPDATE settings SET value='"preserved-relay.example.test"',updated_at=now() WHERE key='notification.email.host';`)
	before := installationSnapshot(t, db)
	installForTest(t, db, fs)
	if err := upgrade(db, fs, false); err != nil {
		t.Fatal(err)
	}
	if err := prepareApplicationDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("business rows, permissions, settings or migration history changed")
	}
}

func TestMigrationCompletionRollback(t *testing.T) {
	db, fs := installationFixture(t, "install_completion_rollback")
	if err := installSchema(db, fs); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO settings(key,value) VALUES('migrations','["v0.3.0"]');
 CREATE FUNCTION reject_history_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.key='migrations' THEN RAISE EXCEPTION 'synthetic history commit failure'; END IF; RETURN NEW; END $$;
 CREATE CONSTRAINT TRIGGER reject_history_commit AFTER UPDATE ON settings DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_history_commit()`)
	err := install(t.Context(), db, fs, true, false)
	if err == nil || !strings.Contains(err.Error(), "recording migration v0.4.0") {
		t.Fatal("history commit failure not reported", err)
	}
	var history string
	db.Get(&history, `SELECT value::text FROM settings WHERE key='migrations'`)
	if history != `["v0.3.0"]` {
		t.Fatal("uncommitted migration appeared completed", history)
	}
	var marker bool
	db.Get(&marker, `SELECT EXISTS(SELECT 1 FROM settings WHERE key='migration_in_progress' AND value->>'version'='v0.4.0')`)
	if !marker {
		t.Fatal("failed completion lost interruption marker")
	}
}

func TestInstallationRefusesNonEmptyFragments(t *testing.T) {
	for i, sql := range []string{`CREATE TABLE settings(key text,value jsonb)`, `CREATE FUNCTION orphaned_installation() RETURNS int LANGUAGE sql AS 'SELECT 1'`, `CREATE TYPE orphaned_installation AS ENUM ('partial')`} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			db, fs := installationFixture(t, fmt.Sprintf("install_fragment_%d", i))
			db.MustExec(sql)
			before := installationSnapshot(t, db)
			if err := install(t.Context(), db, fs, true, false); err == nil {
				t.Fatal("non-empty fragment accepted as empty database")
			}
			if after := installationSnapshot(t, db); after != before {
				t.Fatal("fragment was overwritten")
			}
		})
	}
}

func TestInstallationUnversionedUsedBaseline(t *testing.T) {
	db, fs := installationFixture(t, "install_unversioned_used")
	if err := installSchema(db, fs); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO users(email,type,first_name) VALUES('existing-unversioned@example.test','agent','Existing')`)
	before := installationSnapshot(t, db)
	if err := install(t.Context(), db, fs, true, false); err == nil || !strings.Contains(err.Error(), "non-empty database") {
		t.Fatal("used baseline without history accepted", err)
	}
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("unversioned business data changed")
	}
}

func TestInstallationPendingV224Upgrade(t *testing.T) {
	db, fs := installationFixture(t, "install_pending_v224")
	installForTest(t, db, fs)
	db.MustExec(`UPDATE settings SET value=value-(jsonb_array_length(value)-1)-(jsonb_array_length(value)-2) WHERE key='migrations';
 DROP TABLE message_templates;
 ALTER TABLE conversations DROP CONSTRAINT conversations_contact_required;
 ALTER TABLE conversations ALTER COLUMN contact_id SET NOT NULL`)
	_, pending, err := getPendingMigrations(db)
	if err != nil || len(pending) != 2 || pending[0].version != "v2.24.0" || pending[1].version != "v2.25.0" {
		t.Fatal("legitimate pending upgrade rejected", err)
	}
	if err := prepareApplicationDatabase(t.Context(), db); err == nil {
		t.Fatal("startup accepted pending upgrade")
	}
	if err := upgrade(db, fs, false); err != nil {
		t.Fatal(err)
	}
	if err := checkInstalledDatabase(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	before := installationSnapshot(t, db)
	installForTest(t, db, fs)
	if after := installationSnapshot(t, db); after != before {
		t.Fatal("completed upgrade replayed")
	}
}
