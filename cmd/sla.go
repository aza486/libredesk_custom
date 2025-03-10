package main

import (
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetSLAs returns all SLAs.
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

// handleGetSLA returns the SLA with the given ID.
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

// handleCreateSLA creates a new SLA.
func handleCreateSLA(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		name          = string(r.RequestCtx.PostArgs().Peek("name"))
		desc          = string(r.RequestCtx.PostArgs().Peek("description"))
		firstRespTime = string(r.RequestCtx.PostArgs().Peek("first_response_time"))
		resTime       = string(r.RequestCtx.PostArgs().Peek("resolution_time"))
	)
	// Validate time duration strings
	frt, err := time.ParseDuration(firstRespTime)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `first_response_time` duration.", nil, envelope.InputError)
	}
	rt, err := time.ParseDuration(resTime)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `resolution_time` duration.", nil, envelope.InputError)
	}
	if frt > rt {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "`first_response_time` should be less than `resolution_time`.", nil, envelope.InputError)
	}
	if err := app.sla.Create(name, desc, firstRespTime, resTime); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("SLA created successfully.")
}

// handleDeleteSLA deletes the SLA with the given ID.
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

// handleUpdateSLA updates the SLA with the given ID.
func handleUpdateSLA(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		name          = string(r.RequestCtx.PostArgs().Peek("name"))
		desc          = string(r.RequestCtx.PostArgs().Peek("description"))
		firstRespTime = string(r.RequestCtx.PostArgs().Peek("first_response_time"))
		resTime       = string(r.RequestCtx.PostArgs().Peek("resolution_time"))
	)
	// Validate time duration strings
	frt, err := time.ParseDuration(firstRespTime)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `first_response_time` duration.", nil, envelope.InputError)
	}
	rt, err := time.ParseDuration(resTime)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `resolution_time` duration.", nil, envelope.InputError)
	}
	if frt > rt {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "`first_response_time` should be less than `resolution_time`.", nil, envelope.InputError)
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
