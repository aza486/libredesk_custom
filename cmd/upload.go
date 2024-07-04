package main

import (
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/zerodha/fastglue"
)

func handleFileUpload(r *fastglue.Request) error {
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

	// Read file into the memory
	file, err := form.File["files"][0].Open()
	srcFileName := form.File["files"][0].Filename
	srcContentType := form.File["files"][0].Header.Get("Content-Type")
	srcFileSize := strconv.FormatInt(form.File["files"][0].Size, 10)
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
	url, err := app.uploadMgr.Upload(srcFileName, srcContentType, file)
	if err != nil {
		app.lo.Error("error uploading file", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Error uploading file", nil, "GeneralException")
	}

	return r.SendEnvelope(map[string]string{
		"url":          url,
		"content_type": srcContentType,
		"name":         srcFileName,
		"size":         srcFileSize,
	})
}

func handleViewFile(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("file_uuid").(string)
	)
	url := app.uploadMgr.Store.GetURL(uuid)
	r.RequestCtx.Response.Header.Set("Location", url)
	return r.Redirect(url, http.StatusFound, nil, "")
}
