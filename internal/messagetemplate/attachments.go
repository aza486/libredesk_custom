package messagetemplate

import (
	"database/sql"
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/envelope"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Stage registers an uploaded private object. It is readable only by its
// uploader until an atomic template save publishes the association.
func (m *Manager) Stage(a Access, mediaID int, totalLimit int64) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock($1)`, int64(a.UserID)); err != nil {
		return err
	}
	var allowed bool
	if err = tx.Get(&allowed, `SELECT `+actorSQL, a.UserID); err != nil {
		return err
	}
	if !allowed {
		return denied()
	}
	var used int64
	if err = tx.Get(&used, `SELECT COALESCE(sum(m.size),0) FROM media m JOIN message_template_attachments ta ON ta.media_id=m.id WHERE ta.uploaded_by=$1 AND ta.template_id IS NULL`, a.UserID); err != nil {
		return err
	}
	var metadataBytes int
	if err = tx.Get(&metadataBytes, `SELECT COALESCE(sum(octet_length(row_to_json(m)::text)),0) FROM media m JOIN message_template_attachments ta ON ta.media_id=m.id WHERE ta.uploaded_by=$1 AND ta.template_id IS NULL`, a.UserID); err != nil {
		return err
	}
	var incomingMetadata int
	if err = tx.Get(&incomingMetadata, `SELECT octet_length(row_to_json(m)::text) FROM media m WHERE id=$1`, mediaID); err != nil {
		return err
	}
	if metadataBytes+incomingMetadata > attachment.MaxDraftMetadataBytes {
		return envelope.NewError(envelope.InputError, "Zu viele noch nicht gespeicherte Anhangsmetadaten. Bitte Änderungen speichern oder Uploads verwerfen.", nil)
	}
	var size int64
	if err = tx.Get(&size, `SELECT size FROM media WHERE id=$1 AND model_type='message_templates' AND model_id IS NULL FOR UPDATE`, mediaID); err != nil {
		return err
	}
	if totalLimit > 0 && used+size > totalLimit {
		return envelope.NewError(envelope.InputError, "Der Speicher für noch nicht gespeicherte Vorlagenanhänge ist ausgeschöpft. Bitte Änderungen speichern oder Uploads verwerfen.", nil)
	}
	if _, err = tx.Exec(`INSERT INTO message_template_attachments(media_id,uploaded_by) VALUES($1,$2)`, mediaID, a.UserID); err != nil {
		return err
	}
	return tx.Commit()
}
func denied() error {
	return envelope.NewError(envelope.PermissionError, "Keine Berechtigung für diesen Anhang.", nil)
}
func (m *Manager) Attachments(a Access, t Template) ([]mmodels.Media, error) {
	var files = []mmodels.Media{}
	err := m.db.Select(&files, `SELECT m.* FROM media m JOIN message_template_attachments ta ON ta.media_id=m.id WHERE ta.template_id=$3 AND EXISTS(SELECT 1 FROM message_templates WHERE id=$3 AND `+visibleSQL+`) ORDER BY m.id`, a.UserID, a.ConversationUUID, t.ID)
	return files, err
}

// AuthorizeFile never treats knowledge of a media UUID as permission. Source
// objects use template ACLs; copies use the established message/draft boundary.
func (m *Manager) AuthorizeFile(a Access, mediaUUID string) error {
	var ta struct {
		TemplateUUID sql.NullString `db:"template_uuid"`
		UploadedBy   sql.NullInt64  `db:"uploaded_by"`
	}
	err := m.db.Get(&ta, `SELECT t.uuid AS template_uuid,ta.uploaded_by FROM message_template_attachments ta JOIN media m ON m.id=ta.media_id LEFT JOIN message_templates t ON t.id=ta.template_id WHERE m.uuid=$1::uuid`, mediaUUID)
	if err == sql.ErrNoRows {
		return notFound()
	}
	if err != nil {
		return err
	}
	if ta.TemplateUUID.Valid {
		_, err = m.Get(a, ta.TemplateUUID.String)
		return err
	}
	var active bool
	if err = m.db.Get(&active, `SELECT `+actorSQL, a.UserID); err != nil {
		return err
	}
	if !active || !ta.UploadedBy.Valid || int(ta.UploadedBy.Int64) != a.UserID {
		return notFound()
	}
	return nil
}
func (m *Manager) DiscardUpload(a Access, uuid string) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var locked int
	if err = tx.Get(&locked, `SELECT id FROM media WHERE uuid=$1::uuid FOR UPDATE`, uuid); err != nil {
		return notFound()
	}
	result, err := tx.Exec(`DELETE FROM message_template_attachments WHERE media_id=(SELECT id FROM media WHERE uuid=$3::uuid) AND template_id IS NULL AND uploaded_by=$1 AND `+actorSQL+` AND $2::text IS NOT NULL`, a.UserID, a.ConversationUUID, uuid)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return notFound()
	}
	return tx.Commit()
}
func syncAttachments(tx *sqlx.Tx, a Access, templateID int, ids []int) error {
	if ids == nil {
		return nil
	} // v2.25 clients preserve existing attachments.
	unique := map[int]bool{}
	for _, id := range ids {
		if id <= 0 || unique[id] {
			return envelope.NewError(envelope.InputError, "Ungültige oder doppelte Anhangsreferenz.", nil)
		}
		unique[id] = true
		var valid bool
		// Lock the media row before publishing. GC uses the same lock.
		if err := tx.Get(&valid, `SELECT EXISTS(SELECT 1 FROM message_template_attachments ta WHERE ta.media_id=m.id AND (ta.template_id=$2 OR (ta.template_id IS NULL AND ta.uploaded_by=$3))) FROM media m WHERE m.id=$1 AND m.model_type='message_templates' FOR UPDATE`, id, templateID, a.UserID); err != nil {
			if err == sql.ErrNoRows {
				return notFound()
			}
			return err
		}
		if !valid {
			return notFound()
		}
	}
	var files []mmodels.Media
	if err := tx.Select(&files, `SELECT * FROM media WHERE id=ANY($1::int[])`, pq.Array(ids)); err != nil {
		return err
	}
	metadata, _ := json.Marshal(files)
	if len(metadata) > attachment.MaxDraftMetadataBytes {
		return envelope.NewError(envelope.InputError, "Die Anhangsmetadaten überschreiten das bestehende Entwurfslimit.", nil)
	}
	if _, err := tx.Exec(`DELETE FROM message_template_attachments WHERE template_id=$1 AND NOT(media_id=ANY($2::int[]))`, templateID, pq.Array(ids)); err != nil {
		return err
	}
	_, err := tx.Exec(`UPDATE message_template_attachments SET template_id=$1 WHERE media_id=ANY($2::int[])`, templateID, pq.Array(ids))
	return err
}

type ApplyInput struct {
	Revision      int64  `json:"revision"`
	DraftType     string `json:"draft_type"`
	AttachmentIDs []int  `json:"attachment_ids"`
	// Accepted for existing v2.26 clients; adoption does not estimate transport size.
	Content string `json:"content"`
}
type Applied struct {
	Content     string          `json:"content"`
	Attachments []mmodels.Media `json:"attachments"`
}

// Apply creates only independent media metadata. File bytes stay immutable in
// the existing store. The transaction rolls back all references on any failure.
func (m *Manager) Apply(a Access, uuid string, in ApplyInput, perFile int64) (Applied, error) {
	out := Applied{Attachments: []mmodels.Media{}}
	if in.DraftType != "reply" && in.DraftType != "private_note" {
		return out, envelope.NewError(envelope.InputError, "Ungültiger Entwurfsmodus.", nil)
	}
	t, err := m.Get(a, uuid)
	if err != nil {
		return out, err
	}
	tx, err := m.db.Beginx()
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	var conversationID int
	if err = tx.Get(&conversationID, `SELECT id FROM conversations WHERE uuid=$1::uuid FOR UPDATE`, a.ConversationUUID); err != nil {
		return out, err
	}
	// Lock against concurrent template changes and recheck authorization/revision.
	if err = tx.Get(&t, `SELECT * FROM message_templates WHERE uuid=$3::uuid AND `+visibleSQL+` FOR SHARE`, a.UserID, a.ConversationUUID, uuid); err != nil {
		if err == sql.ErrNoRows {
			err = notFound()
		}
		return out, err
	}
	if in.Revision > 0 && t.Revision != in.Revision {
		return out, envelope.NewError(envelope.ConflictError, "Die Vorlage wurde zwischenzeitlich geändert. Bitte erneut auswählen.", nil)
	}
	var files []mmodels.Media
	if err = tx.Select(&files, `SELECT m.* FROM media m JOIN message_template_attachments ta ON ta.media_id=m.id WHERE ta.template_id=$1 ORDER BY m.id FOR SHARE OF m`, t.ID); err != nil {
		return out, err
	}
	existing := map[int]bool{}
	sources := map[int]bool{}
	for _, id := range in.AttachmentIDs {
		if existing[id] {
			continue
		}
		existing[id] = true
		var f mmodels.Media
		if err = tx.Get(&f, `SELECT * FROM media WHERE id=$1 FOR SHARE`, id); err != nil {
			return out, notFound()
		}
		if (f.Model.String != "" && f.Model.String != "messages") || f.ModelID.Int > 0 {
			return out, denied()
		}
		var ref struct {
			Source       int    `db:"source_media_id"`
			User         int    `db:"user_id"`
			Conversation int    `db:"conversation_id"`
			Type         string `db:"draft_type"`
		}
		err = tx.Get(&ref, `SELECT source_media_id,COALESCE(user_id,0) AS user_id,conversation_id,draft_type FROM media_blob_references WHERE media_id=$1`, id)
		if err != nil && err != sql.ErrNoRows {
			return out, err
		}
		if err == nil {
			if ref.User != a.UserID || ref.Conversation != conversationID || ref.Type != in.DraftType {
				return out, denied()
			}
			sources[ref.Source] = true
		}
		out.Attachments = append(out.Attachments, f)
	}
	for _, f := range files {
		if sources[f.ID] {
			continue
		}
		if perFile > 0 && int64(f.Size) > perFile {
			return out, envelope.NewError(envelope.InputError, "Ein Vorlagenanhang überschreitet das aktuelle Uploadlimit.", nil)
		}
	}
	for _, f := range files {
		if sources[f.ID] {
			continue
		}
		sources[f.ID] = true
		var cloned mmodels.Media
		err = tx.Get(&cloned, `SELECT m.* FROM media m JOIN media_blob_references ref ON ref.media_id=m.id WHERE ref.source_media_id=$1 AND ref.user_id=$2 AND ref.conversation_id=$3 AND ref.draft_type=$4 AND m.model_id IS NULL ORDER BY m.id LIMIT 1 FOR SHARE OF m`, f.ID, a.UserID, conversationID, in.DraftType)
		if err == sql.ErrNoRows {
			meta, _ := json.Marshal(map[string]any{"template_source_uuid": f.UUID})
			err = tx.Get(&cloned, `INSERT INTO media(store,filename,content_type,size,model_type,disposition,private,meta,content_id) VALUES($1,$2,$3,$4,'messages','attachment',true,$5,'') RETURNING *`, f.Store, f.Filename, f.ContentType, f.Size, string(meta))
			if err != nil {
				return out, err
			}
			if _, err = tx.Exec(`INSERT INTO media_blob_references(media_id,source_media_id,user_id,conversation_id,draft_type) VALUES($1,$2,$3,$4,$5)`, cloned.ID, f.ID, a.UserID, conversationID, in.DraftType); err != nil {
				return out, err
			}
		} else if err != nil {
			return out, err
		}
		out.Attachments = append(out.Attachments, cloned)
	}
	// Respect the existing draft metadata ceiling, including URL/header overhead.
	metadata, _ := json.Marshal(out.Attachments)
	if len(metadata) > attachment.MaxDraftMetadataBytes {
		return out, envelope.NewError(envelope.InputError, "Zu viele Anhangsmetadaten für einen Nachrichtenentwurf. Es wurde nichts übernommen.", nil)
	}
	out.Content = t.Content
	if err = tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}

// DraftFileOwner guards independent, not-yet-sent references. Once linked, the
// normal message ACL applies instead; the original template is never exposed.
func (m *Manager) DraftFileOwner(a Access, mediaID int) (string, error) {
	var row struct {
		UUID string `db:"uuid"`
		User int    `db:"user_id"`
	}
	err := m.db.Get(&row, `SELECT c.uuid,COALESCE(ref.user_id,0) AS user_id FROM media_blob_references ref JOIN conversations c ON c.id=ref.conversation_id WHERE ref.media_id=$1`, mediaID)
	if err != nil {
		return "", notFound()
	}
	if row.User != a.UserID {
		return "", notFound()
	}
	return row.UUID, nil
}
