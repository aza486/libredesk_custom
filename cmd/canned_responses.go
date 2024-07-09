package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetCannedResponses(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	c, err := app.cannedRespManager.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Error fetching canned responses", nil, "")
	}
	return r.SendEnvelope(c)
}
