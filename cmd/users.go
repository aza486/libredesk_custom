package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
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

func handleGetUser(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid user `id`.", nil, envelope.InputError)
	}
	user, err := app.userManager.GetUser(id, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(user)
}

func handleUpdateUser(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = umodels.User{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid SIP `id`.", nil, envelope.InputError)
	}

	if _, err := fastglue.ScanArgs(r.RequestCtx.PostArgs(), &user, `json`); err != nil {
		return envelope.NewError(envelope.InputError,
			fmt.Sprintf("Invalid request (%s)", err.Error()), nil)
	}
	err = app.userManager.UpdateUser(id, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
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
		return r.SendErrorEnvelope(http.StatusBadRequest, "Empty `email`", nil, envelope.InputError)
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
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	u, err := app.userManager.GetUser(user.ID, "")
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(u)
}
