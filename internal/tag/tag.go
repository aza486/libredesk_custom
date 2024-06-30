package tag

import (
	"embed"
	"fmt"
	"time"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Tag struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}

type Manager struct {
	q  queries
	lo *logf.Logger
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetAllTags *sqlx.Stmt `query:"get-all-tags"`
	InsertTag  *sqlx.Stmt `query:"insert-tag"`
	DeleteTag  *sqlx.Stmt `query:"delete-tag"`
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

func (t *Manager) GetAll() ([]Tag, error) {
	var tt []Tag
	if err := t.q.GetAllTags.Select(&tt); err != nil {
		t.lo.Error("fetching tags", "error", err)
		return tt, fmt.Errorf("error fetching tags")
	}
	return tt, nil
}

func (t *Manager) AddTag(name string) error {
	if _, err := t.q.InsertTag.Exec(name); err != nil {
		t.lo.Error("inserting tag", "error", err)
		return fmt.Errorf("error inserting tag")
	}
	return nil
}

func (t *Manager) DeleteTag(id int) error {
	if _, err := t.q.DeleteTag.Exec(id); err != nil {
		t.lo.Error("deleting tag", "error", err)
		return fmt.Errorf("error deleting tag")
	}
	return nil
}
