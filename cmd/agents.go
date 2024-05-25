package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetAgents(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.userDB.GetAgents()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope(agents)
}

func handleGetAgentProfile(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		userEmail, _ = r.RequestCtx.UserValue("user_email").(string)
	)
	agents, err := app.userDB.GetAgent(userEmail)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope(agents)
}
