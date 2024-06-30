package auth

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Engine struct {
	q  queries
	lo *logf.Logger
}

type queries struct {
	HasPermission *sqlx.Stmt `query:"has-permission"`
}

func New(db *sqlx.DB, lo *logf.Logger) (*Engine, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}
	return &Engine{
		q:  q,
		lo: lo,
	}, nil
}

func (e *Engine) HasPermission(userID int, perm string) (bool, error) {
	var hasPerm bool
	if err := e.q.HasPermission.Get(&hasPerm, userID, perm); err != nil {
		e.lo.Error("error fetching user permissions", "user_id", userID, "error", err)
		return hasPerm, err
	}
	return hasPerm, nil
}
