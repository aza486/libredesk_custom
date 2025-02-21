package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/oidc/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetAllEnabledOIDC returns all enabled OIDC records
func handleGetAllEnabledOIDC(r *fastglue.Request) error {
	app := r.Context.(*App)
	out, err := app.oidc.GetAllEnabled()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

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
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid OIDC `id`", nil, envelope.InputError)
	}
	o, err := app.oidc.Get(id, false)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(o)
}

// handleTestOIDC tests an OIDC provider URL by doing a discovery on the provider URL.
func handleTestOIDC(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		providerURL = string(r.RequestCtx.PostArgs().Peek("provider_url"))
	)
	if err := app.auth.TestProvider(providerURL); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("OIDC provider discovered successfully")
}

// handleCreateOIDC creates a new OIDC record.
func handleCreateOIDC(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.OIDC{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, envelope.GeneralError)
	}

	if err := app.oidc.Create(req); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Reload the auth manager to update the OIDC providers.
	if err := reloadAuth(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, envelope.GeneralError)
	}
	return r.SendEnvelope("OIDC created successfully")
}

// handleUpdateOIDC updates an OIDC record.
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

	if err = app.oidc.Update(id, req); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Reload the auth manager to update the OIDC providers.
	if err := reloadAuth(app); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, envelope.GeneralError)
	}
	return r.SendEnvelope("OIDC updated successfully")
}

// handleDeleteOIDC deletes an OIDC record.
func handleDeleteOIDC(r *fastglue.Request) error {
	var app = r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid oidc `id`.", nil, envelope.InputError)
	}
	if err = app.oidc.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("OIDC deleted successfully")
}
