package main

import (
	"net/mail"
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetInboxes(r *fastglue.Request) error {
	var app = r.Context.(*App)
	inboxes, err := app.inbox.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(inboxes)
}

func handleGetInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	inbox, err := app.inbox.GetDBRecord(id)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error fetching inbox", nil, envelope.GeneralError)
	}
	if err := inbox.ClearPasswords(); err != nil {
		app.lo.Error("error clearing out passwords", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error fetching inbox", nil)
	}
	return r.SendEnvelope(inbox)
}

func handleCreateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if err := app.inbox.Create(inbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := validateInbox(inbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error reloading inboxes", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

// handleUpdateInbox updates an inbox
func handleUpdateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid inbox `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if err := validateInbox(inbox); err != nil {
		return sendErrorEnvelope(r, err)
	}

	err = app.inbox.Update(id, inbox)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Could not update inbox.", nil, envelope.GeneralError)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error reloading inboxes, Please restart the app.", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(inbox)
}

// handleToggleInbox toggles an inbox
func handleToggleInbox(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid inbox `id`.", nil, envelope.InputError)
	}

	if err = app.inbox.Toggle(id); err != nil {
		return err
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error reloading inboxes", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

// handleDeleteInbox deletes an inbox
func handleDeleteInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		id, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	err := app.inbox.SoftDelete(id)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Could not update inbox.", nil, envelope.GeneralError)
	}

	if err := reloadInboxes(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error reloading inboxes", nil, envelope.GeneralError)
	}

	return r.SendEnvelope(true)
}

// validateInbox validates the inbox
func validateInbox(inbox imodels.Inbox) error {
	// Make sure it's a valid from email address.
	if _, err := mail.ParseAddress(inbox.From); err != nil {
		return envelope.NewError(envelope.InputError, "Invalid from email address format, make sure it's a valid email address in the format `Name <mail@example.com>`", nil)
	}

	if len(inbox.Config) == 0 {
		return envelope.NewError(envelope.InputError, "Empty config provided for inbox", nil)
	}

	if inbox.Name == "" {
		return envelope.NewError(envelope.InputError, "Empty name provided for inbox", nil)
	}

	if inbox.Channel == "" {
		return envelope.NewError(envelope.InputError, "Empty channel provided for inbox", nil)
	}
	return nil
}
