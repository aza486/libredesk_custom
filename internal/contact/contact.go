// Package contact provides functionality to manage contacts in the system.
package contact

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles the operations related to contacts.
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
	UpsertContact *sqlx.Stmt `query:"upsert-contact"`
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

// Upsert inserts or updates a contact and returns the contact ID.
func (m *Manager) Upsert(con models.Contact) (int, error) {
	var contactID int
	if err := m.q.UpsertContact.QueryRow(con.Source, con.SourceID, con.InboxID,
		con.FirstName, con.LastName, con.Email, con.PhoneNumber, con.AvatarURL).Scan(&contactID); err != nil {
		m.lo.Error("error upserting contact", "error", err)
		return contactID, err
	}
	return contactID, nil
}
