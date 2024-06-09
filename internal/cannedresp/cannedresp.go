package cannedresp

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/artemis/internal/utils"
	"github.com/jmoiron/sqlx"
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

type CannedResponse struct {
	ID      string `db:"id" json:"id"`
	Title   string `db:"title" json:"title"`
	Content string `db:"content" json:"content"`
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetAll *sqlx.Stmt `query:"get-all"`
}

func New(opts Opts) (*Manager, error) {
	var q queries

	if err := utils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (t *Manager) GetAll() ([]CannedResponse, error) {
	var c []CannedResponse
	if err := t.q.GetAll.Select(&c); err != nil {
		t.lo.Error("fetching canned responses", "error", err)
		return c, fmt.Errorf("error fetching canned responses")
	}
	return c, nil
}
