package role

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/role/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q  queries
	lo *logf.Logger
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	Get    *sqlx.Stmt `query:"get-role"`
	GetAll *sqlx.Stmt `query:"get-all"`
	Delete *sqlx.Stmt `query:"delete-role"`
	Insert *sqlx.Stmt `query:"insert-role"`
	Update *sqlx.Stmt `query:"update-role"`
}

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

func (t *Manager) GetAll() ([]models.Role, error) {
	var roles = make([]models.Role, 0)
	if err := t.q.GetAll.Select(&roles); err != nil {
		t.lo.Error("error fetching roles", "error", err)
		return roles, envelope.NewError(envelope.GeneralError, "Error fetching roles", nil)
	}
	return roles, nil
}

func (t *Manager) Get(id int) (models.Role, error) {
	var role = models.Role{}
	if err := t.q.Get.Get(&role, id); err != nil {
		t.lo.Error("error fetching role", "error", err)
		return role, envelope.NewError(envelope.GeneralError, "Error fetching role", nil)
	}
	return role, nil
}

func (t *Manager) Delete(id int) error {
	if _, err := t.q.Delete.Exec(id); err != nil {
		t.lo.Error("error deleting role", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting role", nil)
	}
	return nil
}

func (u *Manager) Create(r models.Role) error {
	if _, err := u.q.Insert.Exec(r.Name, r.Description, pq.Array(r.Permissions)); err != nil {
		u.lo.Error("error inserting role", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating role", nil)
	}
	return nil
}

func (u *Manager) Update(id int, r models.Role) error {
	if _, err := u.q.Update.Exec(id, r.Name, r.Description, pq.Array(r.Permissions)); err != nil {
		u.lo.Error("error updating role", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating role", nil)
	}
	return nil
}
