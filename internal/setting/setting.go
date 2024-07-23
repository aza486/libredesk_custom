package setting

import (
	"embed"
	"encoding/json"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/setting/models"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q queries
}

type Opts struct {
	DB *sqlx.DB
}

type queries struct {
	GetAll *sqlx.Stmt `query:"get-all"`
}

func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q: q,
	}, nil
}

func (m *Manager) GetAll() (models.Settings, error) {
	var (
		b   types.JSONText
		out models.Settings
	)

	if err := m.q.GetAll.Get(&b); err != nil {
		return out, err
	}

	if err := json.Unmarshal([]byte(b), &out); err != nil {
		return out, err
	}

	return out, nil
}

func (m *Manager) GetAllJSON() (types.JSONText, error) {
	var (
		b types.JSONText
	)

	if err := m.q.GetAll.Get(&b); err != nil {
		return b, err
	}

	return b, nil
}
