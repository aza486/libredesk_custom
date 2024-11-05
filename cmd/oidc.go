package main

import (
	"fmt"
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/oidc/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	redirectURI = "/api/oidc/finish?id=%d"
)

// handleGetAllOIDC returns all OIDC records
func handleGetAllOIDC(r *fastglue.Request) error {
	app := r.Context.(*App)

	out, err := app.oidc.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

// handleGetOIDC returns an OIDC record by id.
func handleGetOIDC(r *fastglue.Request) error {
	app := r.Context.(*App)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid OIDC `id`", nil, envelope.InputError)
	}

	o, err := app.oidc.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	o.RedirectURI = fmt.Sprintf("%s%s", app.consts.AppBaseURL, fmt.Sprintf(redirectURI, o.ID))

	return r.SendEnvelope(o)
}

func handleCreateOIDC(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.OIDC{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, envelope.GeneralError)
	}
	err := app.oidc.Create(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdateOIDC(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.OIDC{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid oidc `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, envelope.GeneralError)
	}

	err = app.oidc.Update(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleDeleteOIDC(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid oidc `id`.", nil, envelope.InputError)
	}
	err = app.oidc.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
