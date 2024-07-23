package main

import "github.com/zerodha/fastglue"

func handleGetSettings(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
	)
	teams, err := app.settingsManager.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(teams)
}
