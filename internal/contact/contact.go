package contact

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/abhinavxd/artemis/internal/utils"
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

	if err := utils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (m *Manager) Upsert(con models.Contact) (int64, error) {
	fmt.Println("con em", con.Email)
	var contactID int64
	if err := m.q.InsertContact.QueryRow(con.Source, con.SourceID, con.InboxID,
		con.FirstName, con.LastName, con.Email, con.PhoneNumber, con.AvatarURL).Scan(&contactID); err != nil {
		m.lo.Error("inserting contact", "error", err)
		return contactID, err
	}
	return contactID, nil
}
