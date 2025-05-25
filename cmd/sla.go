package main

import (
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	smodels "github.com/abhinavxd/libredesk/internal/sla/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetSLAs returns all SLAs.
func handleGetSLAs(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	slas, err := app.sla.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(slas)
}

// handleGetSLA returns the SLA with the given ID.
func handleGetSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	sla, err := app.sla.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(sla)
}

// handleCreateSLA creates a new SLA.
func handleCreateSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		sla smodels.SLAPolicy
	)

	if err := r.Decode(&sla, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if err := validateSLA(app, &sla); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.sla.Create(sla.Name, sla.Description, sla.FirstResponseTime, sla.ResolutionTime, sla.NextResponseTime, sla.Notifications); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("SLA created successfully.")
}

// handleUpdateSLA updates the SLA with the given ID.
func handleUpdateSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		sla smodels.SLAPolicy
	)

	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err := r.Decode(&sla, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.errorParsing", "name", "{globals.terms.request}"), err.Error(), envelope.InputError)
	}

	if err := validateSLA(app, &sla); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.sla.Update(id, sla.Name, sla.Description, sla.FirstResponseTime, sla.ResolutionTime, sla.NextResponseTime, sla.Notifications); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleDeleteSLA deletes the SLA with the given ID.
func handleDeleteSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.invalid", "name", "`id`"), nil, envelope.InputError)
	}

	if err = app.sla.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateSLA validates the SLA policy and returns an envelope.Error if any validation fails.
func validateSLA(app *App, sla *smodels.SLAPolicy) error {
	if sla.Name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil)
	}
	if sla.FirstResponseTime == "" && sla.NextResponseTime == "" && sla.ResolutionTime == "" {
		return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "At least one of `first_response_time`, `next_response_time`, or `resolution_time` must be provided."), nil)
	}

	// Validate notifications if any.
	for _, n := range sla.Notifications {
		if n.Type == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`type`"), nil)
		}
		if n.TimeDelayType == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`time_delay_type`"), nil)
		}
		if n.Metric == "" {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`metric`"), nil)
		}
		if n.TimeDelayType != "immediately" {
			if n.TimeDelay == "" {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`time_delay`"), nil)
			}
			// Validate time delay duration.
			td, err := time.ParseDuration(n.TimeDelay)
			if err != nil {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`time_delay`"), nil)
			}
			if td.Minutes() < 1 {
				return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`time_delay`"), nil)
			}
		}
		if len(n.Recipients) == 0 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.empty", "name", "`recipients`"), nil)
		}
	}

	// Validate first response time duration string if not empty.
	if sla.FirstResponseTime != "" {
		frt, err := time.ParseDuration(sla.FirstResponseTime)
		if err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`first_response_time`"), nil)
		}
		if frt.Minutes() < 1 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`first_response_time`"), nil)
		}
	}

	// Validate resolution time duration string if not empty.
	if sla.ResolutionTime != "" {
		rt, err := time.ParseDuration(sla.ResolutionTime)
		if err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`resolution_time`"), nil)
		}
		if rt.Minutes() < 1 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`resolution_time`"), nil)
		}
		// Compare with first response time if both are present.
		if sla.FirstResponseTime != "" {
			frt, _ := time.ParseDuration(sla.FirstResponseTime)
			if frt > rt {
				return envelope.NewError(envelope.InputError, app.i18n.T("sla.firstResponseTimeAfterResolution"), nil)
			}
		}
	}

	// Validate next response time duration string if not empty.
	if sla.NextResponseTime != "" {
		nrt, err := time.ParseDuration(sla.NextResponseTime)
		if err != nil {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`next_response_time`"), nil)
		}
		if nrt.Minutes() < 1 {
			return envelope.NewError(envelope.InputError, app.i18n.Ts("globals.messages.invalid", "name", "`next_response_time`"), nil)
		}
	}

	return nil
}
