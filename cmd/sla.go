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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SLA `id`.", nil, envelope.InputError)
	}

	sla, err := app.sla.Get(id)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if err := validateSLA(&sla); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.sla.Create(sla.Name, sla.Description, sla.FirstResponseTime, sla.ResolutionTime, sla.Notifications); err != nil {
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
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SLA `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&sla, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if err := validateSLA(&sla); err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.sla.Update(id, sla.Name, sla.Description, sla.FirstResponseTime, sla.ResolutionTime, sla.Notifications); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("SLA updated successfully.")
}

// handleDeleteSLA deletes the SLA with the given ID.
func handleDeleteSLA(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid SLA `id`.", nil, envelope.InputError)
	}

	if err = app.sla.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// validateSLA validates the SLA policy and returns an envelope.Error if any validation fails.
func validateSLA(sla *smodels.SLAPolicy) error {
	if sla.Name == "" {
		return envelope.NewError(envelope.InputError, "SLA `name` is required", nil)
	}
	if sla.FirstResponseTime == "" {
		return envelope.NewError(envelope.InputError, "SLA `first_response_time` is required", nil)
	}
	if sla.ResolutionTime == "" {
		return envelope.NewError(envelope.InputError, "SLA `resolution_time` is required", nil)
	}

	// Validate notifications if any
	for _, n := range sla.Notifications {
		if n.Type == "" {
			return envelope.NewError(envelope.InputError, "SLA notification `type` is required", nil)
		}
		if n.TimeDelayType == "" {
			return envelope.NewError(envelope.InputError, "SLA notification `time_delay_type` is required", nil)
		}
		if n.TimeDelayType != "immediately" {
			if n.TimeDelay == "" {
				return envelope.NewError(envelope.InputError, "SLA notification `time_delay` is required", nil)
			}
		}
		if len(n.Recipients) == 0 {
			return envelope.NewError(envelope.InputError, "SLA notification `recipients` is required", nil)
		}
	}

	// Validate time duration strings
	frt, err := time.ParseDuration(sla.FirstResponseTime)
	if err != nil {
		return envelope.NewError(envelope.InputError, "Invalid `first_response_time` duration", nil)
	}
	if frt.Minutes() < 1 {
		return envelope.NewError(envelope.InputError, "`first_response_time` should be greater than 1 minute", nil)
	}

	rt, err := time.ParseDuration(sla.ResolutionTime)
	if err != nil {
		return envelope.NewError(envelope.InputError, "Invalid `resolution_time` duration", nil)
	}
	if rt.Minutes() < 1 {
		return envelope.NewError(envelope.InputError, "`resolution_time` should be greater than 1 minute", nil)
	}
	if frt > rt {
		return envelope.NewError(envelope.InputError, "`first_response_time` should be less than `resolution_time`", nil)
	}

	return nil
}
