package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetTeams(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
	)
	teams, err := app.userDB.GetTeams()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}
	return r.SendEnvelope(teams)
}
