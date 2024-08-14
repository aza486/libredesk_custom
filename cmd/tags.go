package main

import (
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	tmodels "github.com/abhinavxd/artemis/internal/tag/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetTags(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	t, err := app.tag.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(t)
}

func handleCreateTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		tag = tmodels.Tag{}
	)
	if err := r.Decode(&tag, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if tag.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty tag `Name`", nil, envelope.InputError)
	}

	err := app.tag.Create(tag.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleDeleteTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid tag `id`.", nil, envelope.InputError)
	}

	if id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty tag `ID`", nil, envelope.InputError)
	}

	err = app.tag.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleUpdateTag(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		tag = tmodels.Tag{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid tag `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&tag, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if tag.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty tag `Name`", nil, envelope.InputError)
	}

	err = app.tag.Update(id, tag.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
