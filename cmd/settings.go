package main

import (
	"fmt"

	"github.com/abhinavxd/artemis/internal/setting/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	out, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

func handleUpdateGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.General{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error decoding request", nil, "")
	}

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleGetUploadSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.UploadProvider{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error decoding request", nil, "")
	}

	out, err := app.setting.GetByPrefix("upload")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(out)
}

func handleUpdateUploadSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.UploadProvider{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error decoding request", nil, "")
	}

	fmt.Println("values -> ", req)

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
