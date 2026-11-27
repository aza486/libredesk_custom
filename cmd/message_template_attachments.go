package main

import (
	"fmt"
	"mime"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/messagetemplate"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

func umodelsForTemplate(id int) umodels.User { return umodels.User{ID: id} }
func templateFileURLs(files []mmodels.Media, context string) {
	for i := range files {
		files[i].URL = "/uploads/" + files[i].UUID
		if context != "" {
			files[i].URL += "?conversation_uuid=" + url.QueryEscape(context)
		}
	}
}
func templateDownloadContext(r *fastglue.Request) (messagetemplate.Access, error) {
	return messageTemplateAccess(r)
}
func handleTemplateAttachmentUpload(r *fastglue.Request) error {
	app := r.Context.(*App)
	a, err := templateDownloadContext(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err = app.messageTemplates.List(a); err != nil {
		return sendErrorEnvelope(r, err)
	}
	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Ungültiger Datei-Upload.", nil))
	}
	files := form.File["files"]
	if len(files) != 1 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Bitte jede Datei einzeln hochladen.", nil))
	}
	header := files[0]
	cfg := app.consts.Load().(*constants)
	maxBytes := int64(cfg.MaxFileUploadSizeMB) * 1024 * 1024
	filename := stringutil.SanitizeFilename(header.Filename)
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(filename)), ".")
	if header.Size <= 0 || header.Size > maxBytes {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, fmt.Sprintf("Die Datei ist leer oder größer als %d MB.", cfg.MaxFileUploadSizeMB), nil))
	}
	if !slices.Contains(cfg.AllowedUploadFileExtensions, "*") && !slices.Contains(cfg.AllowedUploadFileExtensions, ext) {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Dieser Dateityp ist gemäß Uploadrichtlinie nicht erlaubt.", nil))
	}
	file, err := header.Open()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	defer file.Close()
	// Detect bytes, never trust the browser's MIME claim. Active document types
	// remain downloads under a sandboxed, authenticated, no-store route.
	detected, err := mimetype.DetectReader(file)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	file.Seek(0, 0)
	contentType, _, _ := mime.ParseMediaType(detected.String())
	uploaded, err := app.media.UploadTemplate(filename, contentType, "", null.StringFrom(mmodels.ModelMessageTemplates), null.Int{}, file, int(header.Size), null.StringFrom("attachment"), []byte("{}"), true)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Staging storage is bounded by the same configured upload budget. Saved
	// attachments no longer count against this temporary per-employee budget.
	if err = app.messageTemplates.Stage(a, uploaded.ID, maxBytes); err != nil {
		_ = app.media.Delete(uploaded.UUID)
		return sendErrorEnvelope(r, err)
	}
	templateFileURLs([]mmodels.Media{uploaded}, a.ConversationUUID)
	uploaded.URL = "/uploads/" + uploaded.UUID + "?conversation_uuid=" + url.QueryEscape(a.ConversationUUID)
	return r.SendEnvelope(uploaded)
}
func handleDiscardTemplateAttachment(r *fastglue.Request) error {
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	id := r.RequestCtx.UserValue("file_uuid").(string)
	if _, err = uuid.Parse(id); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Ungültiger Anhang.", nil))
	}
	app := r.Context.(*App)
	if err = app.messageTemplates.DiscardUpload(a, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.media.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
func handleApplyMessageTemplate(r *fastglue.Request) error {
	app := r.Context.(*App)
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if a.ConversationUUID == "" {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Eine zugängliche Konversation ist erforderlich.", nil))
	}
	var in messagetemplate.ApplyInput
	if err = decodeMessageTemplate(r, &in); err != nil {
		return sendErrorEnvelope(r, err)
	}
	t, err := app.messageTemplates.Get(a, r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if in.Revision > 0 && in.Revision != t.Revision {
		return sendErrorEnvelope(r, envelope.NewError(envelope.ConflictError, "Die Vorlage wurde zwischenzeitlich geändert. Bitte erneut auswählen.", nil))
	}
	in.Revision = t.Revision
	for _, file := range t.Attachments {
		blob, readErr := app.media.GetBlob(file.UUID)
		if readErr != nil || len(blob) != file.Size {
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Ein Vorlagenanhang ist im Dateispeicher nicht verfügbar. Es wurde nichts übernommen.", nil))
		}
	}

	result, err := app.messageTemplates.Apply(a, r.RequestCtx.UserValue("uuid").(string), in, int64(app.consts.Load().(*constants).MaxFileUploadSizeMB)*1024*1024)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	templateFileURLs(result.Attachments, a.ConversationUUID)
	return r.SendEnvelope(result)
}
