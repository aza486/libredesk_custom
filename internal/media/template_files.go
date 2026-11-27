package media

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v9"
)

// TemplateSource marks independent message metadata, never a public storage URL.
func TemplateSource(m models.Media) string {
	var meta struct {
		Source string `json:"template_source_uuid"`
	}
	_ = json.Unmarshal(m.Meta, &meta)
	return meta.Source
}
func (m *Manager) StorageUUID(uuid string) (string, error) {
	record, err := m.Get(0, uuid)
	if err != nil {
		return "", err
	}
	if TemplateSource(record) == "" {
		return record.UUID, nil
	}
	var source string
	err = m.db.Get(&source, `SELECT original.uuid FROM media_blob_references ref JOIN media original ON original.id=ref.source_media_id WHERE ref.media_id=$1`, record.ID)
	return source, err
}

// deleteTemplateMedia serializes with adoption and publication. A durable
// outbox bridges the DB/store boundary; a failed object delete is retried.
func (m *Manager) deleteTemplateMedia(record models.Media) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int
	if err = tx.Get(&id, `SELECT id FROM media WHERE id=$1 FOR UPDATE`, record.ID); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return err
	}
	var used bool
	if err = tx.Get(&used, `SELECT EXISTS(SELECT 1 FROM media_blob_references WHERE source_media_id=$1)
 OR EXISTS(SELECT 1 FROM message_template_attachments WHERE media_id=$1 AND (template_id IS NOT NULL OR created_at>now()-interval '7 days'))
 OR EXISTS(SELECT 1 FROM media m JOIN conversation_messages cm ON cm.id=m.model_id WHERE m.id=$1 AND m.model_type='messages' AND cm.meta->>'deleted_at' IS NULL)
 OR EXISTS(SELECT 1 FROM conversation_drafts d JOIN media_blob_references ref ON ref.media_id=$1 AND ref.user_id=d.user_id AND ref.conversation_id=d.conversation_id AND ref.draft_type=d.type WHERE jsonb_typeof(d.meta->'attachments')='array' AND EXISTS(SELECT 1 FROM jsonb_array_elements(CASE WHEN jsonb_typeof(d.meta->'attachments')='array' THEN d.meta->'attachments' ELSE '[]'::jsonb END) a WHERE a->>'uuid'=$2))`, id, record.UUID); err != nil {
		return err
	}
	if used {
		return nil
	}
	if _, err = tx.Exec(`DELETE FROM media WHERE id=$1`, id); err != nil {
		return err
	}
	if TemplateSource(record) == "" {
		if _, err = tx.Exec(`INSERT INTO media_storage_deletions(uuid,store) VALUES($1,$2) ON CONFLICT DO NOTHING`, record.UUID, record.Store); err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return m.flushStorageDeletions()
}
func (m *Manager) sweepTemplateMedia() {
	var ready bool
	if err := m.db.Get(&ready, `SELECT to_regclass('message_template_attachments') IS NOT NULL`); err != nil || !ready {
		return
	}
	var files []models.Media
	if err := m.db.Select(&files, `SELECT m.* FROM media m WHERE m.model_type='message_templates' AND m.created_at<now()-interval '7 days' AND NOT EXISTS(SELECT 1 FROM message_template_attachments ta WHERE ta.media_id=m.id AND (ta.template_id IS NOT NULL OR ta.created_at>now()-interval '7 days')) AND NOT EXISTS(SELECT 1 FROM media_blob_references ref WHERE ref.source_media_id=m.id)`); err != nil {
		m.lo.Error("template media cleanup failed", "error", err)
		return
	}
	for _, f := range files {
		if err := m.deleteTemplateMedia(f); err != nil {
			m.lo.Error("template file cleanup failed", "error", err)
		}
	}
	if err := m.flushStorageDeletions(); err != nil {
		m.lo.Error("storage deletion retry failed", "error", err)
	}
}
func (m *Manager) flushStorageDeletions() error {
	var uuids []string
	if err := m.db.Select(&uuids, `SELECT uuid FROM media_storage_deletions WHERE store=$1 ORDER BY created_at LIMIT 256`, m.store.Name()); err != nil {
		return err
	}
	for _, uuid := range uuids {
		err := m.store.Delete(uuid)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if _, err = m.db.Exec(`DELETE FROM media_storage_deletions WHERE uuid=$1`, uuid); err != nil {
			return err
		}
	}
	return nil
}

// CheckDraftReference rejects using a protected source object or another
// employee's draft reference in the existing send endpoint.
func (m *Manager) CheckDraftReference(record models.Media, userID int, conversationUUID string) error {
	if record.Model.String == models.ModelMessageTemplates {
		return envelope.NewError(envelope.PermissionError, "Vorlagendateien müssen zuerst autorisiert in den Entwurf übernommen werden.", nil)
	}
	if TemplateSource(record) == "" {
		return nil
	}
	if record.ModelID.Int > 0 || conversationUUID == "" {
		return envelope.NewError(envelope.PermissionError, "Dieser Anhang ist keinem verfügbaren Entwurf zugeordnet.", nil)
	}
	var allowed bool
	err := m.db.Get(&allowed, `SELECT EXISTS(SELECT 1 FROM media_blob_references ref JOIN conversations c ON c.id=ref.conversation_id WHERE ref.media_id=$1 AND ref.user_id=$2 AND c.uuid=$3::uuid)`, record.ID, userID, conversationUUID)
	if err != nil {
		return err
	}
	if !allowed {
		return envelope.NewError(envelope.PermissionError, "Der Anhang gehört zu einem anderen Entwurf.", nil)
	}
	return nil
}

// UploadTemplate records the object identity before writing storage. A crash
// therefore leaves a collectable row, rather than an untracked physical file.
func (m *Manager) UploadTemplate(filename, contentType, contentID string, model null.String, modelID null.Int, content io.ReadSeeker, size int, disposition null.String, meta []byte, private bool) (models.Media, error) {
	name := uuid.NewString()
	record, err := m.Insert(disposition, filename, contentType, contentID, model, name, modelID, size, meta, private)
	if err != nil {
		return record, err
	}
	if _, _, err = m.Upload(name, contentType, content); err != nil {
		_ = m.Delete(name)
		return models.Media{}, err
	}
	return record, nil
}
