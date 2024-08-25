// Package cannedresp provides functionality to manage canned responses in the system.
package cannedresp

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/cannedresp/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles the operations related to canned responses.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts holds the options for creating a new Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetAll *sqlx.Stmt `query:"get-all"`
	Create *sqlx.Stmt `query:"create"`
	Update *sqlx.Stmt `query:"update"`
	Delete *sqlx.Stmt `query:"delete"`
}

// New initializes a new Manager.
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

// GetAll retrieves all canned responses.
func (t *Manager) GetAll() ([]models.CannedResponse, error) {
	var c = make([]models.CannedResponse, 0)
	if err := t.q.GetAll.Select(&c); err != nil {
		t.lo.Error("error fetching canned responses", "error", err)
		return c, envelope.NewError(envelope.GeneralError, "Error fetching canned responses", nil)
	}
	return c, nil
}

// Create adds a new canned response.
func (t *Manager) Create(title, content string) error {
	if _, err := t.q.Create.Exec(title, content); err != nil {
		t.lo.Error("error creating canned response", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating canned response", nil)
	}
	return nil
}

// Update modifies an existing canned response.
func (t *Manager) Update(id int, title, content string) error {
	result, err := t.q.Update.Exec(id, title, content)
	if err != nil {
		t.lo.Error("error updating canned response", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating canned response", nil)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return envelope.NewError(envelope.NotFoundError, "Canned response not found", nil)
	}
	return nil
}

// Delete removes a canned response by ID.
func (t *Manager) Delete(id int) error {
	result, err := t.q.Delete.Exec(id)
	if err != nil {
		t.lo.Error("error deleting canned response", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting canned response", nil)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return envelope.NewError(envelope.NotFoundError, "Canned response not found", nil)
	}
	return nil
}
