package rbac

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/jmoiron/sqlx"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Engine struct {
	q queries
}

type queries struct {
	HasPermission *sqlx.Stmt `query:"has-permission"`
}

func New(db *sqlx.DB) (*Engine, error) {
	var q queries

	if err := dbutils.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	return &Engine{
		q: q,
	}, nil
}

func (e *Engine) HasPermission(userID int, perm string) (bool, error) {
	var hasPerm bool

	if err := e.q.HasPermission.Get(&hasPerm, userID, perm); err != nil {
		return hasPerm, err
	}

	return hasPerm, nil
}
