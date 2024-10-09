// Package media provides functionality for managing files backed by fs or S3.
package media

import (
	"database/sql"
	"embed"
	"errors"
	"io"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/media/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
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
	store      Store
	lo         *logf.Logger
	queries    queries
}

// Opts provides options for configuring the Manager.
type Opts struct {
	Store      Store
	Lo         *logf.Logger
	DB         *sqlx.DB
}

// New initializes and returns a new Manager instance for handling media operations.
func New(opt Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opt.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		store:      opt.Store,
		lo:         opt.Lo,
		queries:    q,
	}, nil
}

// queries holds the prepared SQL statements.
type queries struct {
	Insert     *sqlx.Stmt `query:"insert-media"`
	Get        *sqlx.Stmt `query:"get-media"`
	GetByUUID  *sqlx.Stmt `query:"get-media-by-uuid"`
	Delete     *sqlx.Stmt `query:"delete-media"`
	Attach     *sqlx.Stmt `query:"attach-to-model"`
	GetByModel *sqlx.Stmt `query:"get-model-media"`
}

// UploadAndInsert uploads file on storage and inserts an entry in db.
func (m *Manager) UploadAndInsert(srcFilename, contentType, contentID, modelType string, modelID int, content io.ReadSeeker, fileSize int, disposition string, meta []byte) (models.Media, error) {
	var uuid = uuid.New()
	_, err := m.Upload(uuid.String(), contentType, content)
	if err != nil {
		return models.Media{}, err
	}

	media, err := m.Insert(srcFilename, contentType, contentID, modelType, disposition, uuid.String(), modelID, fileSize, meta)
	if err != nil {
		m.store.Delete(uuid.String())
		return models.Media{}, err
	}
	return media, nil
}

// Upload saves the media file to the storage backend and returns the generated filename.
func (m *Manager) Upload(fileName, contentType string, content io.ReadSeeker) (string, error) {
	fName, err := m.store.Put(fileName, contentType, content)
	if err != nil {
		m.lo.Error("error uploading media", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error uploading media", nil)
	}
	return fName, nil
}

// Insert inserts media details into the database and returns the inserted media record.
func (m *Manager) Insert(fileName, contentType, contentID, modelType, disposition, uuid string, modelID int, fileSize int, meta []byte) (models.Media, error) {
	var id int
	if err := m.queries.Insert.QueryRow(m.store.Name(), fileName, contentType, fileSize, meta, modelID, modelType, disposition, contentID, uuid).Scan(&id); err != nil {
		m.lo.Error("error inserting media", "error", err)
	}
	return m.Get(id)
}

// Get retrieves the media record by its ID and returns the media.
func (m *Manager) Get(id int) (models.Media, error) {
	var media models.Media
	if err := m.queries.Get.Get(&media, id); err != nil {
		m.lo.Error("error fetching media", "error", err)
		return media, envelope.NewError(envelope.GeneralError, "Error fetching media", nil)
	}
	media.URL = m.store.GetURL(media.UUID)
	return media, nil
}

// GetByUUID retrieves a media record by the uuid.
func (m *Manager) GetByUUID(uuid string) (models.Media, error) {
	var media models.Media
	if err := m.queries.GetByUUID.Get(&media, uuid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return media, envelope.NewError(envelope.GeneralError, "File not found", nil)
		}
		m.lo.Error("error fetching media", "error", err)
		return media, envelope.NewError(envelope.GeneralError, "Error fetching media", nil)
	}
	media.URL = m.store.GetURL(uuid)
	return media, nil
}

// GetBlob retrieves the raw binary content of a media file by its name.
func (m *Manager) GetBlob(name string) ([]byte, error) {
	return m.store.GetBlob(name)
}

// GetURL returns the URL for accessing a media file by its name.
func (m *Manager) GetURL(name string) string {
	return m.store.GetURL(name)
}

// Attach associates a media file with a specific model by its ID and model name.
func (m *Manager) Attach(id int, model string, modelID int) error {
	if _, err := m.queries.Attach.Exec(id, model, modelID); err != nil {
		m.lo.Error("error attaching media to model", "model", model, "model_id", modelID, "error", err)
		return err
	}
	return nil
}

// GetByModel retrieves all media files attached to a specific model.
func (m *Manager) GetByModel(modelID int, model string) ([]models.Media, error) {
	var media = make([]models.Media, 0)
	err := m.queries.GetByModel.Select(&media, model, modelID)
	if err != nil {
		m.lo.Error("error getting model media", "model", model, "model_id", modelID, "error", err)
		return nil, err
	}
	return media, nil
}

// Delete deletes a media file from both the storage backend and the database.
func (m *Manager) Delete(name string) error {
	if err := m.store.Delete(name); err != nil {
		m.lo.Error("error deleting media from store", "error", err)
		return err
	}
	if _, err := m.queries.Delete.Exec(name); err != nil {
		m.lo.Error("error deleting media from db", "error", err)
		return err
	}
	return nil
}
