package main

import (
	"github.com/zerodha/fastglue"
)

func handleGetCannedResponses(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	c, err := app.cannedResp.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}
