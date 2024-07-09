package main

import (
	"net/http"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/zerodha/fastglue"
)

func handleGetUsers(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	agents, err := app.userManager.GetUsers()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(agents)
}

func handleCreateUser(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = umodels.User{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(http.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if req.Email == "" {
		return r.SendErrorEnvelope(http.StatusBadRequest, "Empty email", nil, envelope.InputError)
	}

	err := app.userManager.Create(&req)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, envelope.GeneralError)
	}
	return r.SendEnvelope("User created successfully.")
}

func handleGetCurrentUser(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		userID = r.RequestCtx.UserValue("user_id").(int)
	)
	u, err := app.userManager.GetUser(userID, "")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(u)
}
