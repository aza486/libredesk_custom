// Package template manages email templates including insertion and retrieval.
package template

import (
	"database/sql"
	"embed"
	"errors"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/template/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	ErrTemplateNotFound = errors.New("template not found")
)

// Manager handles template-related operations.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	InsertTemplate     *sqlx.Stmt `query:"insert"`
	UpdateTemplate     *sqlx.Stmt `query:"update"`
	DeleteTemplate     *sqlx.Stmt `query:"delete"`
	GetDefaultTemplate *sqlx.Stmt `query:"get-default"`
	GetAllTemplates    *sqlx.Stmt `query:"get-all"`
	GetTemplate        *sqlx.Stmt `query:"get-template"`
}

// New creates and returns a new instance of the Manager.
func New(lo *logf.Logger, db *sqlx.DB) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	return &Manager{q, lo}, nil
}

// Update updates a new template with the given name, and body.
func (m *Manager) Update(id int, t models.Template) error {
	if _, err := m.q.UpdateTemplate.Exec(id, t.Name, t.Body, t.IsDefault); err != nil {
		m.lo.Error("error updating template", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating template", nil)
	}
	return nil
}

// Create creates a template.
func (m *Manager) Create(t models.Template) error {
	if _, err := m.q.InsertTemplate.Exec(t.Name, t.Body, t.IsDefault); err != nil {
		m.lo.Error("error inserting template", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating template", nil)
	}
	return nil
}

// GetDefault retrieves the default template.
func (m *Manager) GetDefault() (models.Template, error) {
	var template models.Template
	if err := m.q.GetDefaultTemplate.Get(&template); err != nil {
		if err == sql.ErrNoRows {
			return template, ErrTemplateNotFound
		}
		m.lo.Error("error fetching default template", "error", err)
		return template, envelope.NewError(envelope.GeneralError, "Error fetching template", nil)
	}
	return template, nil
}

// GetAll returns all templates.
func (m *Manager) GetAll() ([]models.Template, error) {
	var templates = make([]models.Template, 0)
	if err := m.q.GetAllTemplates.Select(&templates); err != nil {
		m.lo.Error("error fetching templates", "error", err)
		return templates, envelope.NewError(envelope.GeneralError, "Error fetching templates", nil)
	}
	return templates, nil
}

// Get returns a template by id.
func (m *Manager) Get(id int) (models.Template, error) {
	var templates = models.Template{}
	if err := m.q.GetTemplate.Get(&templates, id); err != nil {
		m.lo.Error("error fetching template", "error", err)
		return templates, envelope.NewError(envelope.GeneralError, "Error fetching template", nil)
	}
	return templates, nil
}

// Delete deletes a template by id.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteTemplate.Exec(id); err != nil {
		m.lo.Error("error deleting template", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting template", nil)
	}
	return nil
}
