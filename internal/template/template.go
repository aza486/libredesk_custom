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

type Manager struct {
	q queries
}

type queries struct {
	InsertTemplate     *sqlx.Stmt `query:"insert-template"`
	GetTemplate        *sqlx.Stmt `query:"get-template"`
	GetDefaultTemplate *sqlx.Stmt `query:"get-default-template"`
}

func New(db *sqlx.DB) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	return &Manager{q}, nil
}

func (m *Manager) InsertTemplate(name, subject, body string) error {
	if _, err := m.q.InsertTemplate.Exec(name, subject, body); err != nil {
		return err
	}
	return nil
}

func (m *Manager) GetTemplate(name string) (models.Template, error) {
	var template models.Template
	if err := m.q.GetTemplate.Get(&template, name); err != nil {
		return template, err
	}
	return template, nil
}

func (m *Manager) GetDefaultTemplate() (models.Template, error) {
	var template models.Template
	if err := m.q.GetDefaultTemplate.Get(&template); err != nil {
		return template, err
	}
	return template, nil
}
