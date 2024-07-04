package upload

import (
	"embed"
	"fmt"
	"io"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	uriUploads = "/upload/%s"
)

// Manager is the uploads manager.
type Manager struct {
	Store      attachment.Store
	lo         *logf.Logger
	queries    queries
	appBaseURL string
}

type Opts struct {
	Lo         *logf.Logger
	DB         *sqlx.DB
	AppBaseURL string
}

// New creates a new attachment manager instance.
func New(store attachment.Store, opt Opts) (*Manager, error) {
	var q queries

	// Scan SQL file
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		queries:    q,
		Store:      store,
		lo:         opt.Lo,
		appBaseURL: opt.AppBaseURL,
	}, nil
}

type queries struct {
	Insert *sqlx.Stmt `query:"insert-upload"`
	Delete *sqlx.Stmt `query:"delete-upload"`
}

// Upload inserts the attachment details into the db and uploads the attachment.
func (m *Manager) Upload(fileName, contentType string, content io.ReadSeeker) (string, error) {
	var uuid string

	if err := m.queries.Insert.QueryRow(fileName).Scan(&uuid); err != nil {
		m.lo.Error("error inserting upload", "error", err)
		return uuid, err
	}

	if _, err := m.Store.Put(uuid, contentType, content); err != nil {
		m.Delete(uuid)
		return uuid, err
	}

	return m.appBaseURL + fmt.Sprintf(uriUploads, uuid), nil
}

// AttachMessage attaches given attachments to a message.
func (m *Manager) Delete(uuid string) error {
	if err := m.Store.Delete(uuid); err != nil {
		return err
	}
	return nil
}
