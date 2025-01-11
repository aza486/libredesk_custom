package main

import (
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetSLAs(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	slas, err := app.sla.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(slas)
}

func handleGetSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SLA `id`.", nil, envelope.InputError)
	}

	sla, err := app.sla.Get(id)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(sla)
}

func handleCreateSLA(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		name          = string(r.RequestCtx.PostArgs().Peek("name"))
		desc          = string(r.RequestCtx.PostArgs().Peek("description"))
		firstRespTime = string(r.RequestCtx.PostArgs().Peek("first_response_time"))
		resTime       = string(r.RequestCtx.PostArgs().Peek("resolution_time"))
	)

	// Validate time duration strings
	if _, err := time.ParseDuration(firstRespTime); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `first_response_time` duration.", nil, envelope.InputError)
	}
	if _, err := time.ParseDuration(resTime); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `resolution_time` duration.", nil, envelope.InputError)
	}

	if err := app.sla.Create(name, desc, firstRespTime, resTime); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleDeleteSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SLA `id`.", nil, envelope.InputError)
	}

	if err = app.sla.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleUpdateSLA(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		name          = string(r.RequestCtx.PostArgs().Peek("name"))
		desc          = string(r.RequestCtx.PostArgs().Peek("description"))
		firstRespTime = string(r.RequestCtx.PostArgs().Peek("first_response_time"))
		resTime       = string(r.RequestCtx.PostArgs().Peek("resolution_time"))
	)

	// Validate time duration strings
	if _, err := time.ParseDuration(firstRespTime); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `first_response_time` duration.", nil, envelope.InputError)
	}
	if _, err := time.ParseDuration(resTime); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `resolution_time` duration.", nil, envelope.InputError)
	}

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SLA `id`.", nil, envelope.InputError)
	}

	if err := app.sla.Update(id, name, desc, firstRespTime, resTime); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
