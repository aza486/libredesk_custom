package main

import (
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/zerodha/fastglue"
)

func handleAttachmentUpload(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		form, err = r.RequestCtx.MultipartForm()
	)

	if err != nil {
		app.lo.Error("error parsing media form data.", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Error parsing data", nil, "GeneralException")
	}

	if files, ok := form.File["files"]; !ok || len(files) == 0 {
		return r.SendErrorEnvelope(http.StatusBadRequest, "File not found", nil, "InputException")
	}

	if _, ok := form.Value["disposition"]; !ok || len(form.Value["disposition"]) == 0 {
		return r.SendErrorEnvelope(http.StatusBadRequest, "Disposition required", nil, "InputException")
	}

	if form.Value["disposition"][0] != attachment.DispositionAttachment && form.Value["disposition"][0] != attachment.DispositionInline {
		return r.SendErrorEnvelope(http.StatusBadRequest, "Invalid disposition", nil, "InputException")
	}

	// Read file into the memory
	file, err := form.File["files"][0].Open()
	srcFileName := form.File["files"][0].Filename
	srcContentType := form.File["files"][0].Header.Get("Content-Type")
	srcFileSize := form.File["files"][0].Size
	srcDisposition := form.Value["disposition"][0]
	if err != nil {
		app.lo.Error("reading file into the memory", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Error reading file", nil, "GeneralException")
	}
	defer file.Close()

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	// Checking if file type is allowed.
	if !slices.Contains(app.constants.AllowedFileUploadExtensions, "*") {
		if !slices.Contains(app.constants.AllowedFileUploadExtensions, ext) {
			return r.SendErrorEnvelope(http.StatusBadRequest, "Unsupported file type", nil, "GeneralException")
		}
	}

	// Reset the ptr.
	file.Seek(0, 0)
	url, mediaUUID, _, err := app.attachmentMgr.Upload("" /**message uuid**/, srcFileName, srcContentType, srcDisposition, strconv.FormatInt(srcFileSize, 10), file)
	if err != nil {
		app.lo.Error("error uploading file", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Error uploading file", nil, "GeneralException")
	}

	return r.SendEnvelope(map[string]string{
		"url":          url,
		"uuid":         mediaUUID,
		"content_type": srcContentType,
		"name":         srcFileName,
		"size":         strconv.FormatInt(srcFileSize, 10),
	})
}

func handleGetAttachment(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("conversation_uuid").(string)
	)
	url := app.attachmentMgr.Store.GetURL(uuid)
	return r.Redirect(url, http.StatusFound, nil, "")
}
