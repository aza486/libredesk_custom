package filterstore

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/abhinavxd/artemis/internal/user/filterstore/models"
	"github.com/jmoiron/sqlx"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q queries
}

type queries struct {
	GetFilters *sqlx.Stmt `query:"get-user-filters"`
}

func New(db *sqlx.DB) (*Manager, error) {
	var q queries

	if err := dbutils.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q: q,
	}, nil
}

func (m *Manager) GetFilters(userID int, page string) ([]models.Filter, error) {
	var filters []models.Filter
	if err := m.q.GetFilters.Select(&filters, userID, page); err != nil {
		return filters, err
	}
	return filters, nil
}
