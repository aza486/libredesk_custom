package main

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/user"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// A pinned session lock serializes schema/bootstrap commands across processes.
// The local mutex also prevents multiple waiters exhausting one connection pool.
var installationMu sync.Mutex

func withInstallationLock(ctx context.Context, db *sqlx.DB, fn func() error) error {
	installationMu.Lock()
	defer installationMu.Unlock()
	limit := db.Stats().MaxOpenConnections
	if limit == 1 {
		db.SetMaxOpenConns(2)
		defer db.SetMaxOpenConns(limit)
	}
	conn, err := db.Connx(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock(hashtext('libredesk.schema.lifecycle'))`); err != nil {
		_ = conn.Conn.Raw(func(any) error { return driver.ErrBadConn })
		return err
	}
	defer func() {
		// Never return a still-locked connection to the pool after cancellation.
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlockCtx, `SELECT pg_advisory_unlock(hashtext('libredesk.schema.lifecycle'))`); err != nil {
			_ = conn.Conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	return fn()
}

var errSystemAccountMissing = errors.New("system account is missing; run --install --idempotent-install to complete initialization")

func checkSystemAccount(ctx context.Context, db *sqlx.DB) error {
	var accounts []struct {
		ID       int
		Email    string
		Type     string
		Deleted  bool
		Password string
		Admin    bool
	}
	if err := db.SelectContext(ctx, &accounts, `SELECT u.id,COALESCE(u.email,'') AS email,u.type, u.deleted_at IS NOT NULL AS deleted,
 COALESCE(u.password,'') AS password,EXISTS(SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=u.id AND r.name='Admin') AS admin
 FROM users u WHERE lower(btrim(u.email))=lower($1)`, umodels.SystemUserEmail); err != nil {
		return fmt.Errorf("checking system account: %w", err)
	}
	if len(accounts) == 0 {
		return errSystemAccountMissing
	}
	if len(accounts) != 1 {
		return fmt.Errorf("ambiguous system identity: found %d accounts; no accounts or memberships changed", len(accounts))
	}
	a := accounts[0]
	if a.Email != umodels.SystemUserEmail || a.Type != umodels.UserTypeAgent || a.Deleted || !a.Admin {
		return fmt.Errorf("invalid system account ID %d: canonical agent identity, non-deleted account and existing Admin membership required; no automatic repair", a.ID)
	}
	if _, err := bcrypt.Cost([]byte(a.Password)); err != nil {
		return fmt.Errorf("system account ID %d has no valid password hash; no password reset performed", a.ID)
	}
	return nil
}

func completeInstallation(ctx context.Context, db *sqlx.DB, createAccount bool, password string) error {
	// Validate the runtime structure BEFORE any bootstrap writes.
	if err := checkRuntimeSchema(ctx, db); err != nil {
		return err
	}
	if err := checkSystemAccount(ctx, db); err != nil {
		if !createAccount || !errors.Is(err, errSystemAccountMissing) {
			return err
		}
		var admins int
		if err := db.GetContext(ctx, &admins, `SELECT count(*) FROM roles WHERE name='Admin'`); err != nil {
			return err
		}
		if admins != 1 {
			return fmt.Errorf("cannot create system account: expected exactly one Admin role, found %d", admins)
		}
		if password != "" && !user.IsStrongPassword(password) {
			return fmt.Errorf("invalid initial system password: %s", user.PasswordHint)
		}
		if err := user.CreateSystemUser(ctx, password, db); err != nil {
			return err
		}
	}
	if err := ensureSystemData(ctx, db); err != nil {
		return err
	}
	if err := checkSystemAccount(ctx, db); err != nil {
		return err
	}
	return checkSystemData(ctx, db)
}

// Normal starts never invent a missing account or apply migrations.
func prepareApplicationDatabase(ctx context.Context, db *sqlx.DB) error {
	return withInstallationLock(ctx, db, func() error {
		installed, err := checkSchema(db)
		if err != nil {
			return err
		}
		if !installed {
			return errors.New("database is empty; run --install --idempotent-install before starting")
		}
		if err := pendingUpgradeError(db); err != nil {
			return err
		}
		return completeInstallation(ctx, db, false, "")
	})
}

// checkInstalledDatabase is the read-only form of the startup gate.
func checkInstalledDatabase(ctx context.Context, db *sqlx.DB) error {
	installed, err := checkSchema(db)
	if err != nil {
		return err
	}
	if !installed {
		return errors.New("database is empty; install first")
	}
	if err := checkRuntimeSchema(ctx, db); err != nil {
		return err
	}
	if err := checkSystemAccount(ctx, db); err != nil {
		return err
	}
	return checkSystemData(ctx, db)
}
