package main

import (
	"net/http"

	"github.com/abhinavxd/artemis/internal/envelope"
	imodels "github.com/abhinavxd/artemis/internal/inbox/models"
	"github.com/zerodha/fastglue"
)

func handleGetInboxes(r *fastglue.Request) error {
	var app = r.Context.(*App)
	inboxes, err := app.inboxManager.GetAll()
	// TODO: Clear out passwords.
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Could not fetch inboxes", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(inboxes)
}

func handleCreateInbox(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		inbox = imodels.Inbox{}
	)
	if err := r.Decode(&inbox, "json"); err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}
	err := app.inboxManager.Create(inbox)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Inbox created successfully.")
}
