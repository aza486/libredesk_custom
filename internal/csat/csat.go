// Package csat contains the logic for managing CSAT.
package csat

import (
	"database/sql"
	"embed"

	"github.com/abhinavxd/artemis/internal/csat/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager manages CSAT.
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
	Insert *sqlx.Stmt `query:"insert"`
	Get    *sqlx.Stmt `query:"get"`
	Update *sqlx.Stmt `query:"update"`
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

// Create creates a new CSAT for the given conversation ID.
func (m *Manager) Create(conversationID, assignedAgentID int) (models.CSATResponse, error) {
	var (
		uuid string
		rsp  models.CSATResponse
	)
	err := m.q.Insert.QueryRow(conversationID, assignedAgentID).Scan(&uuid)
	if err != nil {
		if dbutil.IsUniqueViolationError(err) {
			m.lo.Error("error creating CSAT", "err", err)
			return rsp, envelope.NewError(envelope.InputError, "CSAT already exists", nil)
		}
		m.lo.Error("error creating CSAT", "err", err)
		return rsp, envelope.NewError(envelope.GeneralError, "Error creating CSAT", nil)
	}
	return m.Get(uuid)
}

// Get retrieves the CSAT for the given UUID.
func (m *Manager) Get(uuid string) (models.CSATResponse, error) {
	var csat models.CSATResponse
	err := m.q.Get.Get(&csat, uuid)
	if err != nil {
		if err == sql.ErrNoRows {
			return csat, envelope.NewError(envelope.InputError, "CSAT not found", nil)
		}
		m.lo.Error("error getting CSAT", "err", err)
		return csat, err
	}
	return csat, nil
}

// UpdateResponse updates the CSAT response for the given csat.
func (m *Manager) UpdateResponse(uuid string, score int, feedback string) error {
	csat, err := m.Get(uuid)
	if err != nil {
		return err
	}

	if csat.Score > 0 || !csat.ResponseTimestamp.IsZero() {
		return envelope.NewError(envelope.InputError, "CSAT already submitted", nil)
	}

	_, err = m.q.Update.Exec(uuid, score, feedback)
	if err != nil {
		m.lo.Error("error updating CSAT", "err", err)
		return envelope.NewError(envelope.GeneralError, "Error updating CSAT", nil)
	}
	return nil
}
