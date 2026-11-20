// Package role handles role-related operations including creating, updating, fetching, and deleting roles.
package role

import (
	"database/sql"
	"embed"

	amodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles role-related operations.
type Manager struct {
	db   *sqlx.DB
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
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
		db:   opts.DB,
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll retrieves all roles.
func (u *Manager) GetAll() ([]models.Role, error) {
	var roles = make([]models.Role, 0)
	if err := u.q.GetAll.Select(&roles); err != nil {
		u.lo.Error("error fetching roles", "error", err)
		return roles, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return roles, nil
}

// Get retrieves a role by ID.
func (u *Manager) Get(id int) (models.Role, error) {
	var role = models.Role{}
	if err := u.q.Get.Get(&role, id); err != nil {
		if err == sql.ErrNoRows {
			return role, envelope.NewError(envelope.NotFoundError, u.i18n.T("validation.notFoundRole"), nil)
		}
		u.lo.Error("error fetching role", "error", err)
		return role, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return role, nil
}

// Delete deletes a role by ID.
func (u *Manager) Delete(id int) error {
	tx, err := u.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`LOCK TABLE roles IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var role models.Role
	err = tx.Stmtx(u.q.Get).Get(&role, id)
	if err == sql.ErrNoRows {
		return envelope.NewError(envelope.NotFoundError, u.i18n.T("validation.notFoundRole"), nil)
	}
	if err != nil {
		return err
	}
	for _, r := range models.DefaultRoles {
		if role.Name == r {
			if r == models.RoleCustomerSupport {
				return envelope.NewError(envelope.InputError, "Kundensupport is a protected technical role and cannot be deleted", nil)
			}
			return envelope.NewError(envelope.InputError, u.i18n.T("admin.role.cannotModifyAdminRole"), nil)
		}
	}
	if _, err := tx.Stmtx(u.q.Delete).Exec(id); err != nil {
		u.lo.Error("error deleting role", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tx.Commit()
}

// Create creates a new role.
func (u *Manager) Create(r models.Role) (models.Role, error) {
	if r.Name == models.RoleCustomerSupport {
		return models.Role{}, envelope.NewError(envelope.InputError, "Kundensupport is a reserved technical role; use installation/upgrade initialization", nil)
	}
	validPermissions, err := u.filterValidPermissions(r.Permissions)
	if err != nil {
		return models.Role{}, err
	}
	if len(validPermissions) == 0 {
		return models.Role{}, envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.empty", "name", u.i18n.P("globals.terms.permission")), nil)
	}
	var result models.Role
	if err := u.q.Insert.Get(&result, r.Name, r.Description, pq.Array(validPermissions)); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return models.Role{}, envelope.NewError(envelope.InputError, u.i18n.T("errors.alreadyExistsRole"), nil)
		}
		u.lo.Error("error inserting role", "error", err)
		return models.Role{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return result, nil
}

// Update updates an existing role.
func (u *Manager) Update(id int, r models.Role) (models.Role, error) {
	validPermissions, err := u.filterValidPermissions(r.Permissions)
	if err != nil {
		return models.Role{}, err
	}
	if len(validPermissions) == 0 {
		return models.Role{}, envelope.NewError(envelope.InputError, u.i18n.Ts("globals.messages.empty", "name", u.i18n.P("globals.terms.permission")), nil)
	}
	// Disallow updating `Admin` role, as the main System login requires it.
	tx, err := u.db.Beginx()
	if err != nil {
		return models.Role{}, err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`LOCK TABLE roles IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return models.Role{}, err
	}
	var role models.Role
	if err = tx.Stmtx(u.q.Get).Get(&role, id); err != nil {
		if err == sql.ErrNoRows {
			return models.Role{}, envelope.NewError(envelope.NotFoundError, u.i18n.T("validation.notFoundRole"), nil)
		}
		return models.Role{}, err
	}
	if role.Name == models.RoleAdmin {
		return models.Role{}, envelope.NewError(envelope.InputError, u.i18n.T("admin.role.cannotModifyAdminRole"), nil)
	}

	if role.Name == models.RoleCustomerSupport || r.Name == models.RoleCustomerSupport {
		if role.Name != r.Name {
			return models.Role{}, envelope.NewError(envelope.InputError, "Kundensupport is a protected technical role and cannot be renamed or replaced", nil)
		}
		if err := validateCustomerSupportPermissions(validPermissions); err != nil {
			return models.Role{}, envelope.NewError(envelope.InputError, err.Error(), nil)
		}
	}
	var result models.Role
	if err := tx.Stmtx(u.q.Update).Get(&result, id, r.Name, r.Description, pq.Array(validPermissions)); err != nil {
		u.lo.Error("error updating role", "error", err)
		return models.Role{}, envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		return models.Role{}, err
	}
	return result, nil
}

// filterValidPermissions filters out invalid permissions, logs warnings for unknown permissions.
func (u *Manager) filterValidPermissions(permissions []string) ([]string, error) {
	validPermissions := make([]string, 0, len(permissions))
	for _, perm := range permissions {
		if amodels.PermissionExists(perm) {
			validPermissions = append(validPermissions, perm)
		} else {
			u.lo.Warn("skipping unknown permission for role", "permission", perm)
		}
	}
	return validPermissions, nil
}

// ComparePermissions returns added and removed permissions between old and new permission sets.
func ComparePermissions(old, new []string) (added, removed []string) {
	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)
	for _, p := range old {
		oldSet[p] = true
	}
	for _, p := range new {
		newSet[p] = true
	}
	for _, p := range new {
		if !oldSet[p] {
			added = append(added, p)
		}
	}
	for _, p := range old {
		if !newSet[p] {
			removed = append(removed, p)
		}
	}
	return
}
