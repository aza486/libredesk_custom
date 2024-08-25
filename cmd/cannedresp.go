package main

import (
	"strconv"

	cmodels "github.com/abhinavxd/artemis/internal/cannedresp/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetCannedResponses(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		c   []cmodels.CannedResponse
	)

	c, err := app.cannedResp.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

func handleCreateCannedResponse(r *fastglue.Request) error {
	var (
		app            = r.Context.(*App)
		cannedResponse = cmodels.CannedResponse{}
	)

	if err := r.Decode(&cannedResponse, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if cannedResponse.Title == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty canned response `Title`", nil, envelope.InputError)
	}

	if cannedResponse.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty canned response `Content`", nil, envelope.InputError)
	}

	err := app.cannedResp.Create(cannedResponse.Title, cannedResponse.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(cannedResponse)
}

func handleDeleteCannedResponse(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid canned response `id`.", nil, envelope.InputError)
	}

	if err := app.cannedResp.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

func handleUpdateCannedResponse(r *fastglue.Request) error {
	var (
		app            = r.Context.(*App)
		cannedResponse = cmodels.CannedResponse{}
	)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			"Invalid canned response `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&cannedResponse, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if cannedResponse.Title == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty canned response `Title`", nil, envelope.InputError)
	}

	if cannedResponse.Content == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty canned response `Content`", nil, envelope.InputError)
	}

	if err = app.cannedResp.Update(id, cannedResponse.Title, cannedResponse.Content); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(cannedResponse)
}
