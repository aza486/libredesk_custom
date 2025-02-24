// Package role handles role-related operations including creating, updating, fetching, and deleting roles.
package role

import (
	"embed"
	"fmt"

	amodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles role-related operations.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	Get    *sqlx.Stmt `query:"get-role"`
	GetAll *sqlx.Stmt `query:"get-all"`
	Delete *sqlx.Stmt `query:"delete-role"`
	Insert *sqlx.Stmt `query:"insert-role"`
	Update *sqlx.Stmt `query:"update-role"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

// GetAll retrieves all roles.
func (t *Manager) GetAll() ([]models.Role, error) {
	var roles = make([]models.Role, 0)
	if err := t.q.GetAll.Select(&roles); err != nil {
		t.lo.Error("error fetching roles", "error", err)
		return roles, envelope.NewError(envelope.GeneralError, "Error fetching roles", nil)
	}
	return roles, nil
}

// Get retrieves a role by ID.
func (t *Manager) Get(id int) (models.Role, error) {
	var role = models.Role{}
	if err := t.q.Get.Get(&role, id); err != nil {
		t.lo.Error("error fetching role", "error", err)
		return role, envelope.NewError(envelope.GeneralError, "Error fetching role", nil)
	}
	return role, nil
}

// Delete deletes a role by ID.
func (t *Manager) Delete(id int) error {
	// Disallow deletion of default roles.
	role, err := t.Get(id)
	if err != nil {
		return err
	}
	for _, r := range models.DefaultRoles {
		if role.Name == r {
			return envelope.NewError(envelope.InputError, "Cannot delete default roles", nil)
		}
	}

	if _, err := t.q.Delete.Exec(id); err != nil {
		t.lo.Error("error deleting role", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting role", nil)
	}
	return nil
}

// Create creates a new role.
func (u *Manager) Create(r models.Role) error {
	if err := u.validatePermissions(r.Permissions); err != nil {
		return err
	}
	if _, err := u.q.Insert.Exec(r.Name, r.Description, pq.Array(r.Permissions)); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.InputError, "Role with this name already exists", nil)
		}
		u.lo.Error("error inserting role", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating role", nil)
	}
	return nil
}

// Update updates an existing role.
func (u *Manager) Update(id int, r models.Role) error {
	if err := u.validatePermissions(r.Permissions); err != nil {
		return err
	}

	// Disallow updating `Admin` role, as the main System login requires it.
	role, err := u.Get(id)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, "Error fetching role", nil)
	}
	if role.Name == models.RoleAdmin {
		return envelope.NewError(envelope.InputError, "Admin role cannot be updated, Please create a new role", nil)
	}

	if _, err := u.q.Update.Exec(id, r.Name, r.Description, pq.Array(r.Permissions)); err != nil {
		u.lo.Error("error updating role", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating role", nil)
	}
	return nil
}

// validatePermissions returns true if all given permissions are valid
func (u *Manager) validatePermissions(permissions []string) error {
	if len(permissions) == 0 {
		return envelope.NewError(envelope.InputError, "Permissions cannot be empty", nil)
	}
	for _, perm := range permissions {
		if !amodels.IsValidPermission(perm) {
			u.lo.Error("error unknown permission", "permission", perm)
			return envelope.NewError(envelope.InputError, fmt.Sprintf("Unknown permission: %s", perm), nil)
		}
	}
	return nil
}
