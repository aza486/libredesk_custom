// Package setting handles the management of application settings.
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

// Manager handles setting-related operations.
type Manager struct {
	q queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
}

// queries contains prepared SQL queries.
type queries struct {
	GetAll *sqlx.Stmt `query:"get-all"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q: q,
	}, nil
}

// GetAll retrieves all settings as a models.Settings struct.
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

// GetAllJSON retrieves all settings as JSON.
func (m *Manager) GetAllJSON() (types.JSONText, error) {
	var b types.JSONText

	if err := m.q.GetAll.Get(&b); err != nil {
		return b, err
	}

	return b, nil
}
