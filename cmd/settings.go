package main

import (
	"encoding/json"
	"strings"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/setting/models"
	"github.com/abhinavxd/artemis/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetGeneralSettings fetches general settings.
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

// handleUpdateGeneralSettings updates general settings.
func handleUpdateGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.General{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, "")
	}

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Settings updated successfully")
}

// handleGetEmailNotificationSettings fetches email notification settings.
func handleGetEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		notif = models.EmailNotification{}
	)

	out, err := app.setting.GetByPrefix("notification.email")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Unmarshal and filter out password.
	if err := json.Unmarshal(out, &notif); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, "Error fetching settings", nil))
	}
	if notif.Password != "" {
		notif.Password = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	return r.SendEnvelope(notif)
}

// handleUpdateEmailNotificationSettings updates email notification settings.
func handleUpdateEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.EmailNotification{}
		cur = models.EmailNotification{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Bad request", nil, envelope.InputError)
	}

	out, err := app.setting.GetByPrefix("notification.email")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := json.Unmarshal(out, &cur); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, "Error updating settings", nil))
	}

	if req.Password == "" {
		req.Password = cur.Password
	}

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Settings updated successfully")
}
