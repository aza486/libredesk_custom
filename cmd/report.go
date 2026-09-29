package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/zerodha/fastglue"
)

// handleOverviewCounts retrieves general dashboard counts for all users.
func handleOverviewCounts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	counts, err := app.report.GetOverViewCounts(viewer)
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
	viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	charts, err := app.report.GetOverviewChart(days, viewer)
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
	viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	sla, err := app.report.GetOverviewSLA(days, viewer)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(sla)
}

// handleOverviewCSAT retrieves CSAT metrics for the dashboard.
func handleOverviewCSAT(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	csat, err := app.report.GetOverviewCSAT(days, viewer)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(csat)
}

// handleOverviewMessageVolume retrieves message volume metrics for the dashboard.
func handleOverviewMessageVolume(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	volume, err := app.report.GetOverviewMessageVolume(days, viewer)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(volume)
}

// handleOverviewTagDistribution retrieves tag distribution metrics for the dashboard.
func handleOverviewTagDistribution(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		days, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("days")))
	)
	viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	tags, err := app.report.GetOverviewTagDistribution(days, viewer)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tags)
}
