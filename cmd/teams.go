package main

import (
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetTeams(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		teams, err = app.teamManager.GetAll()
	)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}

func handleGetTeam(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid team `id`.", nil, envelope.InputError)
	}
	team, err := app.teamManager.GetTeam(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(team)
}
