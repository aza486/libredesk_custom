package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/image"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	tmpl "github.com/abhinavxd/libredesk/internal/template"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const (
	maxAvatarSizeMB = 20
)

// handleGetUsers returns all users.
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

// handleGetUsersCompact returns all users in a compact format.
func handleGetUsersCompact(r *fastglue.Request) error {
	var app = r.Context.(*App)
	agents, err := app.user.GetAllCompact()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(agents)
}

// handleGetUser returns a user.
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

// handleUpdateUserAvailability updates the current user availability.
func handleUpdateUserAvailability(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		auser  = r.RequestCtx.UserValue("user").(amodels.User)
		status = string(r.RequestCtx.PostArgs().Peek("status"))
	)
	if err := app.user.UpdateAvailability(auser.ID, status); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("User availability updated successfully.")
}

// handleGetCurrentUserTeams returns the teams of a user.
func handleGetCurrentUserTeams(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	teams, err := app.team.GetUserTeams(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

// handleUpdateCurrentUser updates the current user.
func handleUpdateCurrentUser(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

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
		linkedModel := null.StringFrom(mmodels.ModelUser)
		linkedID := null.IntFrom(user.ID)
		disposition := null.NewString("", false)
		contentID := ""
		meta := []byte("{}")
		media, err := app.media.UploadAndInsert(srcFileName, srcContentType, contentID, linkedModel, linkedID, file, int(srcFileSize), disposition, meta)
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
	return r.SendEnvelope("User updated successfully.")
}

// handleCreateUser creates a new user.
func handleCreateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = models.User{}
	)
	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if user.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `email`", nil, envelope.InputError)
	}

	if user.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Please select at least one role", nil, envelope.InputError)
	}

	if user.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `first_name`", nil, envelope.InputError)
	}

	// Right now, only agents can be created.
	if err := app.user.CreateAgent(&user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if err := app.team.UpsertUserTeams(user.ID, user.Teams.Names()); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if user.SendWelcomeEmail {
		// Generate reset token.
		resetToken, err := app.user.SetResetPasswordToken(user.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Render template and send email.
		content, err := app.tmpl.RenderTemplate(tmpl.TmplWelcome, map[string]interface{}{
			"ResetToken": resetToken,
			"Email":      user.Email,
		})
		if err != nil {
			app.lo.Error("error rendering template", "error", err)
			return r.SendEnvelope("User created successfully, but error rendering welcome email.")
		}

		if err := app.notifier.Send(notifier.Message{
			UserIDs:  []int{user.ID},
			Subject:  "Welcome",
			Content:  content,
			Provider: notifier.ProviderEmail,
		}); err != nil {
			app.lo.Error("error sending notification message", "error", err)
			return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, "User created successfully, but could not send welcome email.", nil))
		}
	}
	return r.SendEnvelope("User created successfully.")
}

// handleUpdateUser updates a user.
func handleUpdateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = models.User{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid user `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if user.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `email`", nil, envelope.InputError)
	}

	if user.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Please select at least one role", nil, envelope.InputError)
	}

	if user.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `first_name`", nil, envelope.InputError)
	}

	// Update user.
	if err = app.user.Update(id, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if err := app.team.UpsertUserTeams(id, user.Teams.Names()); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("User updated successfully.")
}

// handleDeleteUser soft deletes a user.
func handleDeleteUser(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid user `id`.", nil, envelope.InputError)
	}

	// Soft delete user.
	if err = app.user.SoftDelete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unassign all open conversations assigned to the user.
	if err := app.conversation.UnassignOpen(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("User deleted successfully.")
}

// handleGetCurrentUser returns the current logged in user.
func handleGetCurrentUser(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	u, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(u)
}

// handleDeleteAvatar deletes a user avatar.
func handleDeleteAvatar(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	// Get user
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Valid str?
	if user.AvatarURL.String == "" {
		return r.SendEnvelope("Avatar deleted successfully.")
	}

	fileName := filepath.Base(user.AvatarURL.String)

	// Delete file from the store.
	if err := app.media.Delete(fileName); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.user.UpdateAvatar(user.ID, ""); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Avatar deleted successfully.")
}

// handleResetPassword generates a reset password token and sends an email to the user.
func handleResetPassword(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		p         = r.RequestCtx.PostArgs()
		auser, ok = r.RequestCtx.UserValue("user").(amodels.User)
		email     = string(p.Peek("email"))
	)
	if ok && auser.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User is already logged in, Please logout to reset password.", nil, envelope.InputError)
	}

	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `email`", nil, envelope.InputError)
	}

	user, err := app.user.GetByEmail(email)
	if err != nil {
		// Send 200 even if user not found, to prevent email enumeration.
		return r.SendEnvelope("Reset password email sent successfully.")
	}

	token, err := app.user.SetResetPasswordToken(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Send email.
	content, err := app.tmpl.RenderTemplate(tmpl.TmplResetPassword,
		map[string]string{
			"ResetToken": token,
		})
	if err != nil {
		app.lo.Error("error rendering template", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error rendering template", nil, envelope.GeneralError)
	}

	if err := app.notifier.Send(notifier.Message{
		UserIDs:  []int{user.ID},
		Subject:  "Reset Password",
		Content:  content,
		Provider: notifier.ProviderEmail,
	}); err != nil {
		app.lo.Error("error sending password reset email", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error sending password reset email", nil, envelope.GeneralError)
	}

	return r.SendEnvelope("Reset password email sent successfully.")
}

// handleSetPassword resets the password with the provided token.
func handleSetPassword(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		user, ok = r.RequestCtx.UserValue("user").(amodels.User)
		p        = r.RequestCtx.PostArgs()
		password = string(p.Peek("password"))
		token    = string(p.Peek("token"))
	)

	if ok && user.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "User is already logged in", nil, envelope.InputError)
	}

	if password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `password`", nil, envelope.InputError)
	}

	if err := app.user.ResetPassword(token, password); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("Password reset successfully.")
}
