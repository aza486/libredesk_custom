package main

import (
	"strconv"

	"github.com/zerodha/fastglue"
)

// handleOverviewCounts retrieves general dashboard counts for all users.
func handleOverviewCounts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	counts, err := app.report.GetOverViewCounts()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(counts)
}

// handleOverviewCharts retrieves general dashboard chart data.
func handleOverviewCharts(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	charts, err := app.report.GetOverviewChart(days)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(charts)
}

// handleOverviewSLA retrieves SLA data for the dashboard.
func handleOverviewSLA(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	sla, err := app.report.GetOverviewSLA(days)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(sla)
}
