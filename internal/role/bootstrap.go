package role

import (
	"context"
	"fmt"
	"slices"
	"strings"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// These are the minimum capabilities needed by the existing customer views,
// message reader and v2.23.0 service-address management contract. Operational
// grants (writing, assignment, integration privileges) remain explicit choices.
func customerSupportPermissions() []string {
	return []string{authzmodels.PermConversationsRead, authzmodels.PermMessagesRead, authzmodels.PermServiceEmailsManage}
}

func validateCustomerSupportPermissions(permissions []string) error {
	var missing []string
	for _, permission := range customerSupportPermissions() {
		if !slices.Contains(permissions, permission) {
			missing = append(missing, permission)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("role %q is missing required permissions: %s; review and explicitly update the role, existing permissions and memberships were not changed", models.RoleCustomerSupport, strings.Join(missing, ", "))
	}
	return nil
}

// CheckCustomerSupportRole is read-only. The exact, protected role name is the
// stable technical identity; IDs and additional permissions are installation-local.
func CheckCustomerSupportRole(ctx context.Context, db sqlx.QueryerContext) error {
	var roles []models.Role
	if err := sqlx.SelectContext(ctx, db, &roles, `SELECT id,name,permissions FROM roles WHERE name=$1`, models.RoleCustomerSupport); err != nil {
		return fmt.Errorf("reading customer support role (install/upgrade the schema first): %w", err)
	}
	if len(roles) != 1 {
		return fmt.Errorf("expected exactly one role %q, found %d", models.RoleCustomerSupport, len(roles))
	}
	if err := validateCustomerSupportPermissions(roles[0].Permissions); err != nil {
		return fmt.Errorf("role ID %d, current permissions %v: %w", roles[0].ID, []string(roles[0].Permissions), err)
	}
	return nil
}

// EnsureCustomerSupportRole only inserts a missing role. It never edits existing
// roles, IDs or user_roles. The table lock also serializes ordinary role writes.
func EnsureCustomerSupportRole(ctx context.Context, db *sqlx.DB) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `LOCK TABLE roles IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("locking customer support role: %w", err)
	}
	var count int
	if err = tx.GetContext(ctx, &count, `SELECT count(*) FROM roles WHERE name=$1`, models.RoleCustomerSupport); err != nil {
		return err
	}
	if count == 0 {
		if _, err = tx.ExecContext(ctx, `INSERT INTO roles(name,description,permissions) VALUES($1,$2,$3)`, models.RoleCustomerSupport, "Technical customer visibility and service-address management role", pq.Array(append(customerSupportPermissions(), authzmodels.PermMessageTemplatesSupport))); err != nil {
			return fmt.Errorf("creating customer support role: %w", err)
		}
	}
	if err = CheckCustomerSupportRole(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
