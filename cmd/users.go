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
		app    = r.Context.(*App)
		userID = r.RequestCtx.UserValue("user_id").(int)
	)
	u, err := app.userMgr.GetUser(userID, "")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(u)
}

func handleGetUserFilters(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		userID = r.RequestCtx.UserValue("user_id").(int)
		page   = r.RequestCtx.UserValue("page").(string)
	)
	filters, err := app.userFilterMgr.GetFilters(userID, page)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(filters)
}
