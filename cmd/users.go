package main

import (
	"fmt"
	"mime/multipart"
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
	maxAvatarSizeMB = 5
)

// handleGetAgents returns all agents.
func handleGetAgents(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.user.GetAgents()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetAgentsCompact returns all agents in a compact format.
func handleGetAgentsCompact(r *fastglue.Request) error {
	var app = r.Context.(*App)
	agents, err := app.user.GetAgentsCompact()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agents)
}

// handleGetAgent returns an agent.
func handleGetAgent(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	agent, err := app.user.GetAgent(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(agent)
}

// handleUpdateAgentAvailability updates the current agent availability.
func handleUpdateAgentAvailability(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		auser  = r.RequestCtx.UserValue("user").(amodels.User)
		status = string(r.RequestCtx.PostArgs().Peek("status"))
	)
	if err := app.user.UpdateAvailability(auser.ID, status); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetCurrentAgentTeams returns the teams of an agent.
func handleGetCurrentAgentTeams(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	teams, err := app.team.GetUserTeams(agent.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

// handleUpdateCurrentAgent updates the current agent.
func handleUpdateCurrentAgent(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	files, ok := form.File["files"]

	// Upload avatar?
	if ok && len(files) > 0 {
		if err := uploadUserAvatar(r, &agent, files); err != nil {
			return err
		}
	}
	return r.SendEnvelope(true)
}

// handleCreateAgent creates a new agent.
func handleCreateAgent(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = models.User{}
	)
	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if user.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	if user.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`role`"), nil, envelope.InputError)
	}

	if user.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`first_name`"), nil, envelope.InputError)
	}

	// Right now, only agents can be created.
	if err := app.user.CreateAgent(&user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert user teams.
	if len(user.Teams) > 0 {
		if err := app.team.UpsertUserTeams(user.ID, user.Teams.Names()); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}

	if user.SendWelcomeEmail {
		// Generate reset token.
		resetToken, err := app.user.SetResetPasswordToken(user.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}

		// Render template and send email.
		content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplWelcome, map[string]any{
			"ResetToken": resetToken,
			"Email":      user.Email.String,
		})
		if err != nil {
			app.lo.Error("error rendering template", "error", err)
			return r.SendEnvelope(true)
		}

		if err := app.notifier.Send(notifier.Message{
			RecipientEmails: []string{user.Email.String},
			Subject:         "Welcome to Libredesk",
			Content:         content,
			Provider:        notifier.ProviderEmail,
		}); err != nil {
			app.lo.Error("error sending notification message", "error", err)
			return r.SendEnvelope(true)
		}
	}
	return r.SendEnvelope(true)
}

// handleUpdateAgent updates an agent.
func handleUpdateAgent(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = models.User{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&user, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.InputError)
	}

	if user.Email.String == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	if user.Roles == nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`role`"), nil, envelope.InputError)
	}

	if user.FirstName == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`first_name`"), nil, envelope.InputError)
	}

	// Update agent.
	if err = app.user.UpdateAgent(id, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert agent teams.
	if err := app.team.UpsertUserTeams(id, user.Teams.Names()); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleDeleteAgent soft deletes an agent.
func handleDeleteAgent(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser   = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.terms.user} `id`"), nil, envelope.InputError)
	}

	// Disallow if self-deleting.
	if id == auser.ID {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userCannotDeleteSelf"), nil, envelope.InputError)
	}

	// Soft delete user.
	if err = app.user.SoftDeleteAgent(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unassign all open conversations assigned to the user.
	if err := app.conversation.UnassignOpen(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleGetCurrentAgent returns the current logged in agent.
func handleGetCurrentAgent(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	u, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(u)
}

// handleDeleteCurrentAgentAvatar deletes the current agent's avatar.
func handleDeleteCurrentAgentAvatar(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	// Get user
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Valid str?
	if agent.AvatarURL.String == "" {
		return r.SendEnvelope(true)
	}

	fileName := filepath.Base(agent.AvatarURL.String)

	// Delete file from the store.
	if err := app.media.Delete(fileName); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.user.UpdateAvatar(agent.ID, ""); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleResetPassword generates a reset password token and sends an email to the agent.
func handleResetPassword(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		p         = r.RequestCtx.PostArgs()
		auser, ok = r.RequestCtx.UserValue("user").(amodels.User)
		email     = string(p.Peek("email"))
	)
	if ok && auser.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userAlreadyLoggedIn"), nil, envelope.InputError)
	}

	if email == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`email`"), nil, envelope.InputError)
	}

	agent, err := app.user.GetAgent(0, email)
	if err != nil {
		// Send 200 even if user not found, to prevent email enumeration.
		return r.SendEnvelope("Reset password email sent successfully.")
	}

	token, err := app.user.SetResetPasswordToken(agent.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Send email.
	content, err := app.tmpl.RenderInMemoryTemplate(tmpl.TmplResetPassword, map[string]string{
		"ResetToken": token,
	})
	if err != nil {
		app.lo.Error("error rendering template", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}

	if err := app.notifier.Send(notifier.Message{
		RecipientEmails: []string{agent.Email.String},
		Subject:         "Reset Password",
		Content:         content,
		Provider:        notifier.ProviderEmail,
	}); err != nil {
		app.lo.Error("error sending password reset email", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.errorSendingPasswordResetEmail"), nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

// handleSetPassword resets the password with the provided token.
func handleSetPassword(r *fastglue.Request) error {
	var (
		app       = r.Context.(*App)
		agent, ok = r.RequestCtx.UserValue("user").(amodels.User)
		p         = r.RequestCtx.PostArgs()
		password  = string(p.Peek("password"))
		token     = string(p.Peek("token"))
	)

	if ok && agent.ID > 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("user.userAlreadyLoggedIn"), nil, envelope.InputError)
	}

	if password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "{globals.terms.password}"), nil, envelope.InputError)
	}

	if err := app.user.ResetPassword(token, password); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// uploadUserAvatar uploads the user avatar.
func uploadUserAvatar(r *fastglue.Request, user *models.User, files []*multipart.FileHeader) error {
	var app = r.Context.(*App)

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error opening uploaded file", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorReading", "name", "{globals.terms.file}"), nil, envelope.GeneralError)
	}
	defer file.Close()

	// Sanitize filename.
	srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
	srcContentType := fileHeader.Header.Get("Content-Type")
	srcFileSize := fileHeader.Size
	srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	if !slices.Contains(image.Exts, srcExt) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.fileTypeisNotAnImage"), nil, envelope.InputError)
	}

	// Check file size
	if bytesToMegabytes(srcFileSize) > maxAvatarSizeMB {
		app.lo.Error("error uploaded file size is larger than max allowed", "size", bytesToMegabytes(srcFileSize), "max_allowed", maxAvatarSizeMB)
		return r.SendErrorEnvelope(
			http.StatusRequestEntityTooLarge,
			app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprintf("%dMB", maxAvatarSizeMB)),
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
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.file}"), nil, envelope.GeneralError)
	}

	// Delete current avatar.
	if user.AvatarURL.Valid {
		fileName := filepath.Base(user.AvatarURL.String)
		app.media.Delete(fileName)
	}

	// Save file path.
	path, err := stringutil.GetPathFromURL(media.URL)
	if err != nil {
		app.lo.Debug("error getting path from URL", "url", media.URL, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorUploading", "name", "{globals.terms.file}"), nil, envelope.GeneralError)
	}
	fmt.Println("path", path)
	if err := app.user.UpdateAvatar(user.ID, path); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return nil
}
