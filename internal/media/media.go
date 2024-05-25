package media

import (
	"embed"
	"io"

	"github.com/abhinavxd/artemis/internal/utils"
	"github.com/jmoiron/sqlx"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS
)

// Store represents functions to store and retrieve media (files).
type Store interface {
	Put(string, string, io.ReadSeeker) (string, error)
	Delete(string) error
	GetURL(string) string
	Name() string
}

// Manager is a media store manager.
type Manager struct {
	store   Store
	queries queries
}

// New creates a new Media instance.
func New(store Store, db *sqlx.DB) (*Manager, error) {
	var (
		m = &Manager{
			store: store,
		}
	)

	// Scan SQL file
	if err := utils.ScanSQLFile("queries.sql", &m.queries, db, efs); err != nil {
		return nil, err
	}
	return m, nil
}

type queries struct {
	InsertMedia *sqlx.Stmt `query:"insert-media"`
}

// UploadMedia uploads media to store and inserts it into the DB.
func (m *Manager) UploadMedia(srcFileName, contentType string, content io.ReadSeeker) (string, error) {
	var mediaUUID string

	if err := m.queries.InsertMedia.QueryRow(m.store.Name(), srcFileName, contentType).Scan(&mediaUUID); err != nil {
		return "", err
	}

	if _, err := m.store.Put(mediaUUID, contentType, content); err != nil {
		return "", err
	}

	return m.store.GetURL(mediaUUID), nil
}
