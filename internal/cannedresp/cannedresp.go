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

type CannedResp struct {
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
	GetAllCannedResponses *sqlx.Stmt `query:"get-all-canned-responses"`
}

func New(opts Opts) (*CannedResp, error) {
	var q queries

	if err := utils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &CannedResp{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (t *CannedResp) GetAllCannedResponses() ([]CannedResponse, error) {
	var c []CannedResponse
	if err := t.q.GetAllCannedResponses.Select(&c); err != nil {
		t.lo.Error("fetching canned responses", "error", err)
		return c, fmt.Errorf("error fetching canned responses")
	}
	return c, nil
}
