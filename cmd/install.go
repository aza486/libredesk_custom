package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
)

// install initializes only a genuinely empty database. Repetition completes the
// registered upgrade/bootstrap chain and never replaces an existing schema.
func install(ctx context.Context, db *sqlx.DB, fs stuffbin.FileSystem, idempotentInstall, prompt bool) error {
	return withInstallationLock(ctx, db, func() error {
		installed, err := checkSchema(db)
		if err != nil {
			return err
		}
		if installed && !idempotentInstall {
			return fmt.Errorf("database already contains an installation; refusing destructive --install; use --install --idempotent-install or --upgrade")
		}
		password := os.Getenv("LIBREDESK_SYSTEM_USER_PASSWORD")
		if !installed {
			if password != "" && !user.IsStrongPassword(password) {
				return fmt.Errorf("invalid initial system password: %s", user.PasswordHint)
			}
			if prompt {
				var ok string
				fmt.Print("Install in this empty database (y/n)? ")
				if _, err := fmt.Scanf("%s", &ok); err != nil {
					return err
				}
				if !strings.EqualFold(ok, "y") {
					return fmt.Errorf("installation cancelled")
				}
			}
			if err := installSchema(db, fs); err != nil {
				return fmt.Errorf("installing base schema (transaction rolled back): %w", err)
			}
		}
		if err := runPendingMigrations(ctx, db, fs); err != nil {
			return err
		}
		return completeInstallation(ctx, db, true, password)
	})
}

// setSystemUserPass prompts for pass and sets system user password.
func setSystemUserPass(ctx context.Context, db *sqlx.DB) {
	user.ChangeSystemUserPassword(ctx, db)
}

// installSchema reads the schema file and installs it in the database.
func installSchema(db *sqlx.DB, fs stuffbin.FileSystem) error {
	q, err := fs.Read("/schema.sql")
	if err != nil {
		return err
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(string(q)); err != nil {
		return err
	}
	return tx.Commit()
}
