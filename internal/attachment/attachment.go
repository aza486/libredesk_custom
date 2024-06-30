package attachment

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"net/textproto"

	"github.com/abhinavxd/artemis/internal/attachment/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	uriAttachment = "/api/attachment/%s"
)

const (
	DispositionInline     = "inline"
	DispositionAttachment = "attachment"
)

// Store holds functions to store and retrieve attachments.
type Store interface {
	Put(string, string, io.ReadSeeker) (string, error)
	Delete(string) error
	GetURL(string) string
	GetBlob(string) ([]byte, error)
	Name() string
}

// Manager is the attachment manager.
type Manager struct {
	Store      Store
	lo         *logf.Logger
	queries    queries
	appBaseURL string
}

type Opts struct {
	Store      Store
	Lo         *logf.Logger
	DB         *sqlx.DB
	AppBaseURL string
}

// New creates a new attachment manager instance.
func New(opt Opts) (*Manager, error) {
	var q queries

	// Scan SQL file

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

type queries struct {
	InsertAttachment      *sqlx.Stmt `query:"insert-attachment"`
	DeleteAttachment      *sqlx.Stmt `query:"delete-attachment"`
	AttachMessage         *sqlx.Stmt `query:"attach-message"`
	GetMessageAttachments *sqlx.Stmt `query:"get-message-attachments"`
}

// Upload inserts the attachment details into the db and uploads the attachment.
func (m *Manager) Upload(msgUUID, fileName, contentType, contentDisposition, fileSize string, content io.ReadSeeker) (string, string, int, error) {
	var (
		uuid string
		id   int
	)

	if err := m.queries.InsertAttachment.QueryRow(m.Store.Name(), fileName, contentType, fileSize, sql.NullString{String: msgUUID, Valid: msgUUID != ""}, contentDisposition).Scan(&uuid, &id); err != nil {
		return "", uuid, id, err
	}

	if _, err := m.Store.Put(uuid, contentType, content); err != nil {
		m.queries.DeleteAttachment.Exec(id)
		return "", uuid, id, err
	}

	return m.appBaseURL + fmt.Sprintf(uriAttachment, uuid), uuid, id, nil
}

// AttachMessage attaches given attachments to a message.
func (m *Manager) AttachMessage(attachments models.Attachments, msgID int) error {
	var err error
	for _, attachment := range attachments {
		if attachment.UUID == "" {
			continue
		}
		_, err = m.queries.AttachMessage.Exec(attachment.UUID, msgID)
		if err != nil {
			m.lo.Error("attaching attachments to message", "attachment_uuid", attachment.UUID, "msg_id", msgID, "error", err)
			continue
		}
	}
	if err != nil {
		return fmt.Errorf("attaching attachments to message %d: %w", msgID, err)
	}
	return nil
}

func (m *Manager) GetMessageAttachments(msgID int) (models.Attachments, error) {
	var attachments models.Attachments
	if err := m.queries.GetMessageAttachments.Select(&attachments, msgID); err != nil {
		m.lo.Error("error fetching message attachments", "error", err)
		return attachments, fmt.Errorf("fetching message attachments %d: %w", msgID, err)
	}
	return attachments, nil
}

// MakeHeaders is a helper function that returns a
// textproto.MIMEHeader tailored for attachments, primarily
// email. If no encoding is given, base64 is assumed.
func MakeHeaders(filename, encoding, contentType, disposition string) textproto.MIMEHeader {
	if encoding == "" {
		encoding = "base64"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if disposition == "" {
		disposition = "attachment"
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", disposition+"; filename="+filename)
	h.Set("Content-Type", fmt.Sprintf("%s; name=\""+filename+"\"", contentType))
	h.Set("Content-Transfer-Encoding", encoding)
	return h
}
