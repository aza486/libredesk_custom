// Package report handles the management of reports.
package report

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/report/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
	db   *sqlx.DB
}

// Opts contains options for initializing the report Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetOverviewCharts string `query:"get-overview-charts"`
	GetOverviewCounts string `query:"get-overview-counts"`
	GetOverviewSLA    string `query:"get-overview-sla-counts"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
		db:   opts.DB,
	}, nil
}

// GetOverViewCounts returns overview counts
func (m *Manager) GetOverViewCounts() (json.RawMessage, error) {
	var counts = json.RawMessage{}
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}
	defer tx.Rollback()

	if err := tx.Get(&counts, m.q.GetOverviewCounts); err != nil {
		m.lo.Error("error fetching overview counts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}

	return counts, nil
}

// GetOverviewSLA returns overview SLA data
func (m *Manager) GetOverviewSLA(days int) (json.RawMessage, error) {
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}
	defer tx.Rollback()

	var result models.OverviewSLA
	// Format query with days parameter for both CTEs
	query := fmt.Sprintf(m.q.GetOverviewSLA, days, days)
	if err := tx.Get(&result, query); err != nil {
		m.lo.Error("error fetching overview SLA data", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}

	slaData, err := json.Marshal(result)
	if err != nil {
		m.lo.Error("error marshaling SLA data", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingCount", "name", "{globals.terms.overview}"), nil)
	}

	return slaData, nil
}

// GetOverviewChart returns overview chart data
func (m *Manager) GetOverviewChart(days int) (json.RawMessage, error) {
	var stats = json.RawMessage{}
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		m.lo.Error("error starting db txn", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingChart", "name", "{globals.terms.overview}"), nil)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(m.q.GetOverviewCharts, days, days)
	if err := tx.Get(&stats, query); err != nil {
		m.lo.Error("error fetching overview charts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetchingChart", "name", "{globals.terms.overview}"), nil)
	}
	return stats, nil
}
