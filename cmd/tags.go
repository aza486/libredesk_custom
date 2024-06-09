package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetTags(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	t, err := app.tagMgr.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(t)
}
