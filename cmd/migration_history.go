package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

const migrationInProgressKey = "migration_in_progress"

func migrationIndex(version string) int {
	for i, m := range migList {
		if m.version == version {
			return i
		}
	}
	return -1
}

func readMigrationHistory(q sqlx.Queryer) ([]string, error) {
	var raw []byte
	if err := sqlx.Get(q, &raw, `SELECT value FROM settings WHERE key='migrations'`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading migration history: %w", err)
	}
	var versions []string
	if err := json.Unmarshal(raw, &versions); err != nil || len(versions) == 0 {
		return nil, errors.New("invalid migration history: expected a non-empty array of registered versions; no migrations replayed")
	}
	prev := -1
	for i, v := range versions {
		idx := migrationIndex(v)
		if idx < 0 || (i > 0 && idx != prev+1) {
			return nil, fmt.Errorf("inconsistent migration history at entry %d: unknown, duplicate, missing or out-of-order version; review history, do not reset it", i+1)
		}
		prev = idx
	}
	return versions, nil
}

func getLastMigrationVersion(db *sqlx.DB) (string, error) {
	var interrupted string
	err := db.Get(&interrupted, `SELECT COALESCE(value->>'version','') FROM settings WHERE key=$1`, migrationInProgressKey)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("reading migration state: %w", err)
	}
	if err == nil {
		if migrationIndex(interrupted) < 0 {
			interrupted = "unknown version"
		}
		return "", fmt.Errorf("unfinished migration %s recorded in settings.migration_in_progress; restore/review this migration and its history before manually acknowledging the interruption; automatic replay refused", interrupted)
	}
	versions, err := readMigrationHistory(db)
	if err != nil {
		return "", err
	}
	if len(versions) == 0 {
		return "v0.0.0", nil
	}
	return versions[len(versions)-1], nil
}

func recordMigrationVersion(ver string, db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`LOCK TABLE settings IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	versions, err := readMigrationHistory(tx)
	if err != nil {
		return err
	}
	next := 0
	if len(versions) > 0 {
		next = migrationIndex(versions[len(versions)-1]) + 1
	}
	if next >= len(migList) || migList[next].version != ver {
		return errors.New("refusing duplicate or non-sequential migration completion")
	}
	result, err := tx.Exec(`DELETE FROM settings WHERE key=$1 AND value->>'version'=$2`, migrationInProgressKey, ver)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("migration completion has no matching in-progress marker")
	}
	raw, _ := json.Marshal(append(versions, ver))
	if _, err = tx.Exec(`INSERT INTO settings(key,value) VALUES('migrations',$1::jsonb) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, string(raw)); err != nil {
		return err
	}
	return tx.Commit()
}

func runPendingMigrations(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem) error {
	_, pending, err := getPendingMigrations(db)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	// A legacy schema with no history and existing business data is ambiguous.
	// Only the empty interrupted baseline can safely enter the full old chain.
	last, err := getLastMigrationVersion(db)
	if err != nil {
		return err
	}
	if last == "v0.0.0" {
		var used bool
		if err := db.GetContext(ctx, &used, `SELECT EXISTS(SELECT 1 FROM users) OR EXISTS(SELECT 1 FROM conversations) OR EXISTS(SELECT 1 FROM inboxes) OR EXISTS(SELECT 1 FROM teams) OR EXISTS(SELECT 1 FROM applied_slas) OR EXISTS(SELECT 1 FROM roles WHERE name NOT IN ('Admin','Agent','Kundensupport')) OR (SELECT count(DISTINCT updated_at)>1 FROM settings) OR EXISTS(SELECT 1 FROM settings WHERE key='app.lang' AND value<>'"en-US"'::jsonb)`); err != nil {
			return err
		}
		if used {
			return errors.New("migration history is missing in a non-empty database; automatic execution of historical data migrations refused; restore verified history from backup")
		}
	}
	for _, m := range pending {
		// This committed marker intentionally survives a failed statement, process
		// exit, or a crash between the historical migration commit and recording it.
		if _, err := db.ExecContext(ctx, `INSERT INTO settings(key,value) VALUES($1,jsonb_build_object('version',$2::text,'started_at',now()))`, migrationInProgressKey, m.version); err != nil {
			return fmt.Errorf("marking migration %s: %w", m.version, err)
		}
		if err := m.fn(db, fs, ko); err != nil {
			return fmt.Errorf("migration %s failed; interruption marker retained, no automatic retry: %w", m.version, err)
		}
		if err := recordMigrationVersion(m.version, db); err != nil {
			return fmt.Errorf("recording migration %s; interruption marker retained: %w", m.version, err)
		}
	}
	return nil
}

// These migration-only objects are absent from schema.sql. Finding them ahead
// of the recorded history proves an ambiguous partial upgrade/rewound history.
// Baseline objects already present in schema.sql are deliberately not used.
func checkMigrationFootprints(db *sqlx.DB, last string) error {
	if migrationIndex(last) < migrationIndex("v2.23.0") {
		var ahead bool
		if err := db.Get(&ahead, `SELECT EXISTS(SELECT 1 FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=current_schema() AND t.relname='sidebar_folders' AND c.conname='sidebar_folders_scope_type_check' AND pg_get_constraintdef(c.oid) LIKE '%service_mail%')`); err != nil {
			return err
		}
		if ahead {
			return fmt.Errorf("migration history is missing or behind schema: v2.23.0 service-mail scope exists after recorded %s; automatic replay refused", last)
		}
	}
	for _, f := range []struct{ version, table, column string }{
		{"v2.14.0", "conversation_mail_versions", ""},
		{"v2.15.0", "manual_resolution_feedback", ""},
		{"v2.16.0", "manual_review_annotations", ""},
		{"v2.18.0", "sidebar_folders", ""},
		{"v2.22.0", "users", "work_phone_number"},
	} {
		if migrationIndex(last) >= migrationIndex(f.version) {
			continue
		}
		var present bool
		var err error
		if f.column == "" {
			err = db.Get(&present, `SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=current_schema() AND table_name=$1)`, f.table)
		} else {
			err = db.Get(&present, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 AND column_name=$2)`, f.table, f.column)
		}
		if err != nil {
			return err
		}
		if present {
			return fmt.Errorf("migration history is missing or behind schema: %s object %s exists after recorded %s; partial/rewound history requires manual review, automatic replay refused", f.version, f.table, last)
		}
	}
	return nil
}
