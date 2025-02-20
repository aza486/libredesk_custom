package main

import (
	"strconv"

	businessHours "github.com/abhinavxd/libredesk/internal/business_hours"
	models "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetBusinessHours returns all business hours.
func handleGetBusinessHours(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	businessHours, err := app.businessHours.GetAll()
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(businessHours)
}

// handleGetBusinessHour returns the business hour with the given id.
func handleGetBusinessHour(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid business hour `id`.", nil, envelope.InputError)
	}
	businessHour, err := app.businessHours.Get(id)
	if err != nil {
		if err == businessHours.ErrBusinessHoursNotFound {
			return r.SendErrorEnvelope(fasthttp.StatusNotFound, err.Error(), nil, envelope.NotFoundError)
		}
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error fetching business hour", nil, "")
	}
	return r.SendEnvelope(businessHour)
}

// handleCreateBusinessHours creates a new business hour.
func handleCreateBusinessHours(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		businessHours = models.BusinessHours{}
	)
	if err := r.Decode(&businessHours, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if businessHours.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty business hour `Name`", nil, envelope.InputError)
	}

	if err := app.businessHours.Create(businessHours.Name, businessHours.Description, businessHours.IsAlwaysOpen, businessHours.Hours, businessHours.Holidays); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleDeleteBusinessHour deletes the business hour with the given id.
func handleDeleteBusinessHour(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid business hour `id`.", nil, envelope.InputError)
	}

	err = app.businessHours.Delete(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleUpdateBusinessHours updates the business hour with the given id.
func handleUpdateBusinessHours(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		businessHours = models.BusinessHours{}
	)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid business hour `id`.", nil, envelope.InputError)
	}

	if err := r.Decode(&businessHours, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "decode failed", err.Error(), envelope.InputError)
	}

	if businessHours.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty business hour `Name`", nil, envelope.InputError)
	}

	if err := app.businessHours.Update(id, businessHours.Name, businessHours.Description, businessHours.IsAlwaysOpen, businessHours.Hours, businessHours.Holidays); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
