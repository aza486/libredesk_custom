// Package template manages templates including creation, retrieval and rendering.
package template

import (
	"database/sql"
	"embed"
	"errors"
	"html/template"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/template/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs                   embed.FS
	ErrTemplateNotFound   = errors.New("template not found")
	TypeEmailOutgoing     = "email_outgoing"
	TypeEmailNotification = "email_notification"
)

// Manager handles template-related operations.
type Manager struct {
	tpls    *template.Template
	webTpls *template.Template
	funcMap template.FuncMap
	q       queries
	lo      *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	InsertTemplate     *sqlx.Stmt `query:"insert"`
	UpdateTemplate     *sqlx.Stmt `query:"update"`
	DeleteTemplate     *sqlx.Stmt `query:"delete"`
	GetDefaultTemplate *sqlx.Stmt `query:"get-default"`
	GetAllTemplates    *sqlx.Stmt `query:"get-all"`
	GetTemplate        *sqlx.Stmt `query:"get-template"`
	GetByName          *sqlx.Stmt `query:"get-by-name"`
	IsBuiltIn          *sqlx.Stmt `query:"is-builtin"`
}

// New creates and returns a new instance of the Manager.
func New(lo *logf.Logger, db *sqlx.DB, webTpls *template.Template, tpls *template.Template, funcMap template.FuncMap) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}
	return &Manager{tpls, webTpls, funcMap, q, lo}, nil
}

// Update updates a new template with the given name, and body.
func (m *Manager) Update(id int, t models.Template) error {
	if _, err := m.q.UpdateTemplate.Exec(id, t.Name, t.Body, t.IsDefault, t.Subject, t.Type); err != nil {
		m.lo.Error("error updating template", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating template", nil)
	}
	return nil
}

// Create creates a template.
func (m *Manager) Create(t models.Template) error {
	if t.IsDefault {
		t.Type = TypeEmailOutgoing
	}
	if _, err := m.q.InsertTemplate.Exec(t.Name, t.Body, t.IsDefault, t.Subject, t.Type); err != nil {
		m.lo.Error("error inserting template", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating template", nil)
	}
	return nil
}

// GetAll returns all templates by type.
func (m *Manager) GetAll(typ string) ([]models.Template, error) {
	var templates = make([]models.Template, 0)
	if err := m.q.GetAllTemplates.Select(&templates, typ); err != nil {
		m.lo.Error("error fetching templates", "error", err)
		return templates, envelope.NewError(envelope.GeneralError, "Error fetching templates", nil)
	}
	return templates, nil
}

// Get returns a template by id.
func (m *Manager) Get(id int) (models.Template, error) {
	var templates = models.Template{}
	if err := m.q.GetTemplate.Get(&templates, id); err != nil {
		if err == sql.ErrNoRows {
			return templates, envelope.NewError(envelope.NotFoundError, "Template not found", nil)
		}
		m.lo.Error("error fetching template", "error", err)
		return templates, envelope.NewError(envelope.GeneralError, "Error fetching template", nil)
	}
	return templates, nil
}

// Delete deletes a template by id.
func (m *Manager) Delete(id int) error {
	// Do not allow deletion of built-in templates.
	isBuiltIn, err := m.isBuiltIn(id)
	if err != nil {
		return err
	}
	if isBuiltIn {
		return envelope.NewError(envelope.PermissionError, "Cannot delete built-in templates", nil)
	}
	if _, err := m.q.DeleteTemplate.Exec(id); err != nil {
		m.lo.Error("error deleting template", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting template", nil)
	}
	return nil
}

// isBuiltIn returns true if the template is built-in.
func (m *Manager) isBuiltIn(id int) (bool, error) {
	var isBuiltIn bool
	if err := m.q.IsBuiltIn.Get(&isBuiltIn, id); err != nil {
		m.lo.Error("error fetching template", "error", err)
		return false, envelope.NewError(envelope.GeneralError, "Error fetching template", nil)
	}
	return isBuiltIn, nil
}

// getDefaultOutgoingEmailTemplate returns the default outgoing email template.
func (m *Manager) getDefaultOutgoingEmailTemplate() (models.Template, error) {
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

// getByName returns a template by name.
func (m *Manager) getByName(name string) (models.Template, error) {
	var template models.Template
	if err := m.q.GetByName.Get(&template, name); err != nil {
		if err == sql.ErrNoRows {
			return template, ErrTemplateNotFound
		}
		m.lo.Error("error fetching default template", "error", err)
		return template, envelope.NewError(envelope.GeneralError, "Error fetching template", nil)
	}
	return template, nil
}
