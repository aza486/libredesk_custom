package main

import (
	"strconv"

	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetStatuses(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	out, err := app.status.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}


func handleCreateStatus(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		status = cmodels.Status{}
	)
	if err := r.Decode(&status, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if status.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty status `Name`", nil, envelope.InputError)
	}

	err := app.status.Create(status.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleDeleteStatus(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid status `id`.", nil, envelope.InputError)
	}

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty status `ID`", nil, envelope.InputError)
	}

	err = app.status.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleUpdateStatus(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		status = cmodels.Status{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid status `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&status, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if status.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty status `Name`", nil, envelope.InputError)
	}

	err = app.status.Update(id, status.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
