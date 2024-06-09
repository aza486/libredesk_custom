package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetUsers(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.userMgr.GetUsers()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(agents)
}

func handleGetCurrentUser(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		userUUID = r.RequestCtx.UserValue("user_uuid").(string)
	)
	u, err := app.userMgr.GetUser(userUUID)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(u)
}
