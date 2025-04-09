package main

import (
	"path/filepath"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

// handleGetContacts returns a list of contacts from the database.
func handleGetContacts(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	contacts, err := app.user.GetContacts(page, pageSize, order, orderBy, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(contacts) > 0 {
		total = contacts[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    contacts,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetTags returns a contact from the database.
func handleGetContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}
	c, err := app.user.GetContact(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// handleUpdateContact updates a contact in the database.
func handleUpdateContact(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	form, err := r.RequestCtx.MultipartForm()
	if err != nil {
		app.lo.Error("error parsing form data", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), nil, envelope.GeneralError)
	}

	// Parse form data
	firstName := ""
	if v, ok := form.Value["first_name"]; ok && len(v) > 0 {
		firstName = string(v[0])
	}
	lastName := ""
	if v, ok := form.Value["last_name"]; ok && len(v) > 0 {
		lastName = string(v[0])
	}
	email := ""
	if v, ok := form.Value["email"]; ok && len(v) > 0 {
		email = string(v[0])
	}
	phoneNumber := ""
	if v, ok := form.Value["phone_number"]; ok && len(v) > 0 {
		phoneNumber = string(v[0])
	}
	phoneNumberCallingCode := ""
	if v, ok := form.Value["phone_number_calling_code"]; ok && len(v) > 0 {
		phoneNumberCallingCode = string(v[0])
	}
	enabled := false
	if v, ok := form.Value["enabled"]; ok && len(v) > 0 {
		enabled = string(v[0]) == "true"
	}

	user := models.User{
		FirstName:              firstName,
		LastName:               lastName,
		Email:                  null.StringFrom(email),
		PhoneNumber:            null.StringFrom(phoneNumber),
		PhoneNumberCallingCode: null.StringFrom(phoneNumberCallingCode),
		Enabled:                enabled,
	}

	if err := app.user.UpdateContact(id, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upload avatar?
	files, ok := form.File["files"]
	if ok && len(files) > 0 {
		contact, err := app.user.GetContact(id, "")
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if err := uploadUserAvatar(r, &contact, files); err != nil {
			app.lo.Error("error uploading avatar", "error", err)
			return err
		}
	}
	return r.SendEnvelope(true)
}

// handleDeleteContactAvatar deletes contact avatar from storage and database.
func handleDeleteContactAvatar(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	// Get user
	contact, err := app.user.GetContact(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Valid str?
	if contact.AvatarURL.String == "" {
		return r.SendEnvelope(true)
	}

	fileName := filepath.Base(contact.AvatarURL.String)

	// Delete file from the store.
	if err := app.media.Delete(fileName); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err = app.user.UpdateAvatar(contact.ID, ""); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
