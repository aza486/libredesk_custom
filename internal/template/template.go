// Package template manages email templates including insertion and retrieval.
package template

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/template/models"
	"github.com/jmoiron/sqlx"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles template-related operations.
type Manager struct {
	q queries
}

// queries contains prepared SQL queries.
type queries struct {
	InsertTemplate     *sqlx.Stmt `query:"insert-template"`
	GetTemplate        *sqlx.Stmt `query:"get-template"`
	GetDefaultTemplate *sqlx.Stmt `query:"get-default-template"`
}

// New creates and returns a new instance of the Manager.
func New(db *sqlx.DB) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	return &Manager{q}, nil
}

// InsertTemplate inserts a new template with the given name, subject, and body.
func (m *Manager) InsertTemplate(name, subject, body string) error {
	if _, err := m.q.InsertTemplate.Exec(name, subject, body); err != nil {
		return err
	}
	return nil
}

// GetTemplate retrieves a template by its name.
func (m *Manager) GetTemplate(name string) (models.Template, error) {
	var template models.Template
	if err := m.q.GetTemplate.Get(&template, name); err != nil {
		return template, err
	}
	return template, nil
}

// GetDefaultTemplate retrieves the default template.
func (m *Manager) GetDefaultTemplate() (models.Template, error) {
	var template models.Template
	if err := m.q.GetDefaultTemplate.Get(&template); err != nil {
		return template, err
	}
	return template, nil
}
