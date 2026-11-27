// Copyright Kailash Nadh (https://github.com/knadh/listmonk)
// SPDX-License-Identifier: AGPL-3.0
// Adapted from listmonk for Libredesk.

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	"golang.org/x/mod/semver"
)

// migFunc represents a migration function for a particular version.
// fn (generally) executes database migrations and additionally
// takes the filesystem and config objects in case there are additional bits
// of logic to be performed before executing upgrades. Some historical functions
// commit data changes in multiple steps; completed versions must never replay.
type migFunc struct {
	version string
	fn      func(*sqlx.DB, stuffbin.FileSystem, *koanf.Koanf) error
}

// migList is the list of available migList ordered by the semver.
// Each migration is a Go file in internal/migrations named after the semver.
// The functions are named as: v0.7.0 => migrations.V0_7_0() and are idempotent.
var migList = []migFunc{
	{"v0.3.0", migrations.V0_3_0},
	{"v0.4.0", migrations.V0_4_0},
	{"v0.5.0", migrations.V0_5_0},
	{"v0.6.0", migrations.V0_6_0},
	{"v0.7.0", migrations.V0_7_0},
	{"v0.7.4", migrations.V0_7_4},
	{"v0.8.5", migrations.V0_8_5},
	{"v0.9.1", migrations.V0_9_1},
	{"v0.10.0", migrations.V0_10_0},
	{"v1.0.1", migrations.V1_0_1},
	{"v2.0.0", migrations.V2_0_0},
	{"v2.2.0", migrations.V2_2_0},
	{"v2.3.0", migrations.V2_3_0},
	{"v2.4.0", migrations.V2_4_0},
	{"v2.5.0", migrations.V2_5_0},
	{"v2.6.0", migrations.V2_6_0},
	{"v2.8.0", migrations.V2_8_0},
	{"v2.9.0", migrations.V2_9_0},
	{"v2.10.0", migrations.V2_10_0},
	{"v2.11.0", migrations.V2_11_0},
	{"v2.12.0", migrations.V2_12_0},
	{"v2.13.0", migrations.V2_13_0},
	{"v2.14.0", migrations.V2_14_0},
	{"v2.15.0", migrations.V2_15_0},
	{"v2.16.0", migrations.V2_16_0},
	{"v2.17.0", migrations.V2_17_0},
	{"v2.18.0", migrations.V2_18_0},
	{"v2.19.0", migrations.V2_19_0},
	{"v2.20.0", migrations.V2_20_0},
	{"v2.21.0", migrations.V2_21_0},
	{"v2.22.0", migrations.V2_22_0},
	{"v2.23.0", migrations.V2_23_0},
	{"v2.24.0", migrations.V2_24_0},
	{"v2.25.0", migrations.V2_25_0},
}

// upgrade upgrades the database to the current version by running SQL migration files
// for all version from the last known version to the current one.
func upgrade(db *sqlx.DB, fs stuffbin.FileSystem, prompt bool) error {
	if prompt {
		var ok string
		fmt.Print("Take a backup before upgrading. Continue (y/n)? ")
		if _, err := fmt.Scanf("%s", &ok); err != nil {
			return err
		}
		if !strings.EqualFold(ok, "y") {
			return fmt.Errorf("upgrade cancelled")
		}
	}
	return withInstallationLock(context.Background(), db, func() error {
		installed, err := checkSchema(db)
		if err != nil {
			return err
		}
		if !installed {
			return fmt.Errorf("database is empty; run --install --idempotent-install first")
		}
		if err := runPendingMigrations(context.Background(), db, fs); err != nil {
			return err
		}
		return completeInstallation(context.Background(), db, true, "")
	})
}

// getPendingMigrations gets the pending migrations by comparing the last
// recorded migration in the DB against all migrations listed in `migrations`.
func getPendingMigrations(db *sqlx.DB) (string, []migFunc, error) {
	for i, m := range migList {
		if !semver.IsValid(m.version) || m.fn == nil || (i > 0 && semver.Compare(migList[i-1].version, m.version) >= 0) {
			return "", nil, fmt.Errorf("invalid migration registry at entry %d", i+1)
		}
	}
	last, err := getLastMigrationVersion(db)
	if err != nil {
		return "", nil, err
	}
	if err := checkMigrationFootprints(db, last); err != nil {
		return "", nil, err
	}
	start := 0
	if last != "v0.0.0" {
		start = migrationIndex(last) + 1
	}
	return last, migList[start:], nil
}

// pendingUpgradeError also validates the entire history, not only its tail.
func pendingUpgradeError(db *sqlx.DB) error {
	last, pending, err := getPendingMigrations(db)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}
	return fmt.Errorf("%d pending database migrations after %s; back up and run --upgrade (or --install --idempotent-install)", len(pending), last)
}
