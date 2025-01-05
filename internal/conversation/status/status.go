// Package status handles the management of conversation statuses.
package status

import (
	"embed"
	"slices"

	"github.com/abhinavxd/artemis/internal/conversation/status/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles changes to statuses.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetStatus      *sqlx.Stmt `query:"get-status"`
	GetAllStatuses *sqlx.Stmt `query:"get-all-statuses"`
	InsertStatus   *sqlx.Stmt `query:"insert-status"`
	DeleteStatus   *sqlx.Stmt `query:"delete-status"`
	UpdateStatus   *sqlx.Stmt `query:"update-status"`
}

// New creates and returns a new instance of the Manager.
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

// GetAll retrieves all statuses.
func (m *Manager) GetAll() ([]models.Status, error) {
	var statuses = make([]models.Status, 0)
	if err := m.q.GetAllStatuses.Select(&statuses); err != nil {
		m.lo.Error("error fetching statuses", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching statuses", nil)
	}
	return statuses, nil
}

// Create creates a new status.
func (m *Manager) Create(name string) error {
	if _, err := m.q.InsertStatus.Exec(name); err != nil {
		m.lo.Error("error inserting status", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating status", nil)
	}
	return nil
}

// Delete deletes a status by ID.
func (m *Manager) Delete(id int) error {
	// Disallow deletion of default statuses.
	status, err := m.get(id)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, "Error fetching status", nil)
	}

	if slices.Contains(models.DefaultStatuses, status.Name) {
		return envelope.NewError(envelope.InputError, "Cannot delete default status", nil)
	}

	if _, err := m.q.DeleteStatus.Exec(id); err != nil {
		// Check if the error is a foreign key error.
		if dbutil.IsForeignKeyError(err) {
			return envelope.NewError(envelope.InputError, "Cannot delete status as it is in use, Please remove this status from all conversations before deleting.", nil)
		}
		m.lo.Error("error deleting status", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting status", nil)
	}
	return nil
}

// Update updates a status by id.
func (m *Manager) Update(id int, name string) error {
	if _, err := m.q.UpdateStatus.Exec(id, name); err != nil {
		m.lo.Error("error updating status", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating status", nil)
	}
	return nil
}

// get retrieves a status by ID.
func (m *Manager) get(id int) (models.Status, error) {
	var status models.Status
	if err := m.q.GetStatus.Get(&status, id); err != nil {
		m.lo.Error("error fetching status", "error", err)
		return status, err
	}
	return status, nil
}
