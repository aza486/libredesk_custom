package main

import (
	"context"
	"errors"

	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/tag"
	"github.com/jmoiron/sqlx"
)

// Reuse the same initialization gates for install, upgrade and normal startup.
// Each initializer is independently transactional and never repairs conflicts.
func ensureSystemData(ctx context.Context, db *sqlx.DB) error {
	if err := tag.EnsureSystemTags(ctx, db); err != nil {
		return err
	}
	return role.EnsureCustomerSupportRole(ctx, db)
}

// Retain the existing read-only CLI preflight, including its no-worker behavior.
func checkSystemData(ctx context.Context, db *sqlx.DB) error {
	return errors.Join(tag.CheckSystemTags(ctx, db), role.CheckCustomerSupportRole(ctx, db))
}
