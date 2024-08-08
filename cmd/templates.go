package main

import (
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/template/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetTemplates(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	t, err := app.tmpl.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

func handleGetTemplate(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid template `id`.", nil, envelope.InputError)
	}
	t, err := app.tmpl.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(t)
}

func handleCreateTemplate(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Template{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, envelope.GeneralError)
	}

	err := app.tmpl.Create(req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdateTemplate(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.Template{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid template `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, envelope.GeneralError)
	}

	err = app.tmpl.Update(id, req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
