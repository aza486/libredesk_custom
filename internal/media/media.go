// Package media provides functionality for managing media files in the system.
// It allows uploading, retrieving, and deleting media files, and interfaces with a storage backend.
package media

import (
	"context"
	"embed"
	"io"
	"regexp"
	"time"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/media/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS

	// This matches filenames, sans extensions, of the format
	// filename_(number). The number is incremented in case
	// new file uploads conflict with existing filenames.
	FnameRegexp = regexp.MustCompile(`(.+?)_([0-9]+)$`)
)

// Store defines the interface for media storage operations.
type Store interface {
	Put(name, contentType string, content io.ReadSeeker) (string, error)
	Delete(name string) error
	GetURL(name string) string
	GetBlob(name string) ([]byte, error)
	Name() string
}

// Manager manages media files, including their upload and retrieval.
type Manager struct {
	Store      Store
	lo         *logf.Logger
	queries    queries
	appBaseURL string
}

// Opts provides options for configuring the Manager.
type Opts struct {
	Store      Store
	Lo         *logf.Logger
	DB         *sqlx.DB
	AppBaseURL string
}

// New initializes a new Manager with the provided options.
// It scans and prepares SQL queries needed for media management.
func New(opt Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		Store:      opt.Store,
		lo:         opt.Lo,
		queries:    q,
		appBaseURL: opt.AppBaseURL,
	}, nil
}

// queries holds the prepared SQL statements for media operations.
type queries struct {
	InsertMedia      *sqlx.Stmt `query:"insert-media"`
	GetMedia         *sqlx.Stmt `query:"get-media"`
	DeleteMedia      *sqlx.Stmt `query:"delete-media"`
	Attach           *sqlx.Stmt `query:"attach-to-model"`
	GetModelMedia    *sqlx.Stmt `query:"get-model-media"`
	GetUnlinkedMedia *sqlx.Stmt `query:"get-unlinked-media"`
}

func (m *Manager) UploadAndInsert(fileName, contentType string, content io.ReadSeeker, fileSize int, meta []byte) (models.Media, error) {
	// Upload the media to storage
	uploadedFileName, err := m.Upload(fileName, contentType, content)
	if err != nil {
		return models.Media{}, err
	}

	// Insert media details into the database
	media, err := m.Insert(uploadedFileName, contentType, fileSize, meta)
	if err != nil {
		m.Store.Delete(uploadedFileName)
		return models.Media{}, err
	}

	return media, nil
}

// Upload saves the media to storage and returns the filename.
func (m *Manager) Upload(fileName, contentType string, content io.ReadSeeker) (string, error) {
	fName, err := m.Store.Put(fileName, contentType, content)
	if err != nil {
		m.lo.Error("error uploading media", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error uploading media", nil)
	}

	return fName, nil
}

func (m *Manager) Insert(fileName, contentType string, fileSize int, meta []byte) (models.Media, error) {
	var id int
	if err := m.queries.InsertMedia.QueryRow(m.Store.Name(), fileName, contentType, fileSize, meta).Scan(&id); err != nil {
		m.lo.Error("error inserting media", "error", err)
	}
	return m.Get(id)
}

// Get retrieves the media record by UUID and returns it along with its URL.
// If an error occurs, it returns the error and an empty media record.
func (m *Manager) Get(id int) (models.Media, error) {
	var media models.Media
	if err := m.queries.GetMedia.Get(&media, id); err != nil {
		m.lo.Error("error fetching media", "error", err)
		return media, envelope.NewError(envelope.GeneralError, "Error fetching media", nil)
	}
	media.URL = m.Store.GetURL(media.Filename)
	return media, nil
}

func (m *Manager) GetBlob(name string) ([]byte, error) {
	return m.Store.GetBlob(name)
}

func (m *Manager) GetURL(name string) string {
	return m.Store.GetURL(name)
}

func (m *Manager) Attach(id int, model string, modelID int) error {
	if _, err := m.queries.Attach.Exec(id, model, modelID); err != nil {
		m.lo.Error("error attaching media to model", "model", model, "model_id", modelID, "error", err)
		return err
	}
	return nil
}

func (m *Manager) GetModelMedia(modelID int, model string) ([]models.Media, error) {
	var media = make([]models.Media, 0)
	err := m.queries.GetModelMedia.Select(&media, model, modelID)
	if err != nil {
		m.lo.Error("error getting model media", "model", model, "model_id", modelID, "error", err)
		return nil, err
	}
	return media, nil
}

// Delete deletes media from store.
func (m *Manager) Delete(name string) error {
	if err := m.Store.Delete(name); err != nil {
		m.lo.Error("error deleting media from store", "error", err)
		return err
	}
	return nil
}

// DeleteMedia deletes media record from DB.
func (m *Manager) DeleteMedia(name string) error {
	if _, err := m.queries.DeleteMedia.Exec(name); err != nil {
		m.lo.Error("error deleting media from db", "error", err)
		return err
	}
	return nil
}

// DeleteMediaAndStore deletes media from both the database and the store.
func (m *Manager) DeleteMediaAndStore(name string) error {
	if err := m.DeleteMedia(name); err != nil {
		return err
	}
	m.Delete(name)
	return nil
}

// CleanMedia periodically deletes media entries that are not linked to any model.
// This function should be run as a goroutine to avoid blocking. It uses a ticker to
// trigger the deletion process every 12 hours.
func (m *Manager) CleanMedia(ctx context.Context) {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.deleteUnlinkedMedia()
		}
	}
}

// deleteUnlinkedMedia fetches media entries that are not linked to any model
// and are older than 3 days, then deletes them.
func (m *Manager) deleteUnlinkedMedia() {
	var unlinkedMedia []int
	if err := m.queries.GetUnlinkedMedia.Select(&unlinkedMedia, time.Now().AddDate(0, 0, -3)); err != nil {
		m.lo.Error("error fetching unlinked media", "error", err)
		return
	}

	m.lo.Info("found unlinked media to delete", "count", len(unlinkedMedia))

	for _, id := range unlinkedMedia {
		_, err := m.queries.DeleteMedia.Exec(id)
		if err != nil {
			m.lo.Error("error deleting unlinked media", "ID", id, "error", err)
		}
	}
}
