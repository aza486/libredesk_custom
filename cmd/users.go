package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/image"
	mmodels "github.com/abhinavxd/artemis/internal/media/models"
	"github.com/abhinavxd/artemis/internal/stringutil"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	maxAvatarSizeMB = 5
)

func handleGetUsers(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.user.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(agents)
}

func handleGetUsersCompact(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.user.GetAllCompact()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(agents)
}

func handleGetUser(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid user `id`.", nil, envelope.InputError)
	}
	user, err := app.user.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(user)
}

func handleUpdateCurrentUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Get current user.
	currentUser, err := app.user.Get(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error parsing data", nil, envelope.GeneralError)
	}

	files, ok := form.File["files"]

	// Upload avatar?
	if ok && len(files) > 0 {
		fileHeader := files[0]
		file, err := fileHeader.Open()
		if err != nil {
			app.lo.Error("error reading uploaded", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error reading file", nil, envelope.GeneralError)
		}
		defer file.Close()

		// Sanitize filename.
		srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
		srcContentType := fileHeader.Header.Get("Content-Type")
		srcFileSize := fileHeader.Size
		srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

		if !slices.Contains(image.Exts, srcExt) {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "File type is not an image", nil, envelope.InputError)
		}

		// Check file size
		if bytesToMegabytes(srcFileSize) > maxAvatarSizeMB {
			app.lo.Error("error uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", maxAvatarSizeMB)
			return r.SendErrorEnvelope(
				http.StatusRequestEntityTooLarge,
				fmt.Sprintf("File size is too large. Please upload file lesser than %d MB", maxAvatarSizeMB),
				nil,
				envelope.GeneralError,
			)
		}

		// Reset ptr.
		file.Seek(0, 0)
		media, err := app.media.UploadAndInsert(srcFileName, srcContentType, "", mmodels.ModelUser, user.ID, file, int(srcFileSize), "", []byte("{}"))
		if err != nil {
			app.lo.Error("error uploading file", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error uploading file", nil, envelope.GeneralError)
		}

		// Delete current avatar.
		if currentUser.AvatarURL.Valid {
			fileName := filepath.Base(currentUser.AvatarURL.String)
			app.media.Delete(fileName)
		}

		// Save file path.
		path, err := stringutil.GetPathFromURL(media.URL)
		if err != nil {
			app.lo.Debug("error getting path from URL", "url", media.URL, "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error uploading file", nil, envelope.GeneralError)
		}
		if err := app.user.UpdateAvatar(user.ID, path); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	return r.SendEnvelope(true)
}

func handleCreateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = umodels.User{}
	)
	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if user.Email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `email`", nil, envelope.InputError)
	}

	err := app.user.Create(&user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if err := app.team.UpsertUserTeams(user.ID, user.Teams.Names()); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleUpdateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = umodels.User{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid user `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	// Update user.
	err = app.user.UpdateUser(id, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if err := app.team.UpsertUserTeams(id, user.Teams.Names()); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleGetCurrentUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	u, err := app.user.Get(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(u)
}

func handleDeleteAvatar(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Get user
	user, err := app.user.Get(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Valid str?
	if user.AvatarURL.String == "" {
		return r.SendEnvelope(true)
	}

	fileName := filepath.Base(user.AvatarURL.String)

	// Delete file from the store.
	if err := app.media.Delete(fileName); err != nil {
		return sendErrorEnvelope(r, err)
	}
	err = app.user.UpdateAvatar(user.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
