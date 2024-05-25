package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetMessages(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	messages, err := app.conversations.GetMessages(uuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope(messages)
}
