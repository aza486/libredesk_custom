package contact

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q  queries
	lo *logf.Logger
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	InsertContact *sqlx.Stmt `query:"insert-contact"`
}

func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (m *Manager) Upsert(con models.Contact) (int, error) {
	var contactID int
	if err := m.q.InsertContact.QueryRow(con.Source, con.SourceID, con.InboxID,
		con.FirstName, con.LastName, con.Email, con.PhoneNumber, con.AvatarURL).Scan(&contactID); err != nil {
		m.lo.Error("inserting contact", "error", err)
		return contactID, err
	}
	return contactID, nil
}
