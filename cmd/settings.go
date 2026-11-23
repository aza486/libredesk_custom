package main

import (
	"encoding/json"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetGeneralSettings fetches general settings, this endpoint is not behind auth as it has no sensitive data and is required for the app to function.
func handleGetGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	out, err := app.setting.GetByPrefix("app")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Unmarshal to set the app.update to the settings, so the frontend can show that an update is available.
	var settings map[string]interface{}
	if err := json.Unmarshal(out, &settings); err != nil {
		app.lo.Error("error unmarshalling settings", "err", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Set the app.update to the settings, adding `app` prefix to the key to match the settings structure in db.
	settings["app.update"] = app.update
	// Set app version.
	settings["app.version"] = versionString
	// Set restart required flag.
	settings["app.restart_required"] = app.restartRequired
	return r.SendEnvelope(settings)
}

// handleUpdateGeneralSettings updates general settings.
func handleUpdateGeneralSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.General{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Trim whitespace from string fields.
	req.SiteName = strings.TrimSpace(req.SiteName)
	req.FaviconURL = strings.TrimSpace(req.FaviconURL)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	req.Timezone = strings.TrimSpace(req.Timezone)
	if req.Timezone != "" && !stringutil.IsValidTimezone(req.Timezone) {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid timezone.", nil, envelope.InputError)
	}
	// Trim whitespace and trailing slash from root URL.
	req.RootURL = strings.TrimRight(strings.TrimSpace(req.RootURL), "/")

	// Get current language before update.
	app.Lock()
	oldLang := ko.String("app.lang")
	app.Unlock()

	if err := app.setting.Update(req); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Reload the settings and templates.
	if err := reloadSettings(app); err != nil {
		app.lo.Error("error reloading settings", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	// Check if language changed and reload i18n if needed.
	app.Lock()
	newLang := ko.String("app.lang")
	if oldLang != newLang {
		app.lo.Info("language changed, reloading i18n", "old_lang", oldLang, "new_lang", newLang)
		app.i18n = initI18n(app.fs)
		app.lo.Info("reloaded i18n", "old_lang", oldLang, "new_lang", newLang)
	}
	app.Unlock()

	if err := reloadTemplates(app); err != nil {
		app.lo.Error("error reloading templates", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(true)
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
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	if notif.Password != "" {
		notif.Password = strings.Repeat(stringutil.PasswordDummy, 10)
	}
	records, err := app.inbox.GetRoutingRecords()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	options := make([]map[string]string, 0)
	for _, record := range records {
		if record.Enabled && record.Channel == "email" && record.AccessMode == "public" && mailboxAddress(record.From) != "" {
			unique := true
			for _, other := range records {
				if other.ID != record.ID && other.Enabled && other.Channel == "email" && other.AccessMode == "public" && mailboxAddress(other.From) == mailboxAddress(record.From) {
					unique = false
				}
			}
			if unique {
				options = append(options, map[string]string{"uuid": record.UUID, "name": record.Name, "email": mailboxAddress(record.From)})
			}
		}
	}
	identity, contact, _ := internalEmailConfiguration(app)
	// Invalid legacy choices must not prevent administrators from repairing settings.
	if identity != "" {
		if _, err := resolveNotificationInbox(app.inbox, identity, false); err != nil {
			identity = ""
		}
	}
	return r.SendEnvelope(struct {
		models.EmailNotification
		Options          []map[string]string `json:"notification.email.inbox_options"`
		EffectiveInbox   string              `json:"notification.email.effective_inbox_uuid"`
		EffectiveContact string              `json:"notification.email.effective_team_contact_email"`
	}{notif, options, identity, contact})
}

// handleUpdateEmailNotificationSettings updates email notification settings.
func handleUpdateEmailNotificationSettings(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = models.EmailNotification{}
	)

	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Trim whitespace from string fields (Password intentionally NOT trimmed).
	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.EmailAddress = strings.TrimSpace(req.EmailAddress)
	req.HelloHostname = strings.TrimSpace(req.HelloHostname)
	req.IdleTimeout = strings.TrimSpace(req.IdleTimeout)
	req.WaitTimeout = strings.TrimSpace(req.WaitTimeout)

	var provided map[string]json.RawMessage
	if err := json.Unmarshal(r.RequestCtx.PostBody(), &provided); err != nil {
		return sendErrorEnvelope(r, notificationInput("Expected a notification settings object"))
	}
	if err := app.setting.UpdateEmailNotification(req, func(next *models.EmailNotification) error {
		if next.InboxUUID != nil && strings.TrimSpace(*next.InboxUUID) != "" {
			identity := strings.TrimSpace(*next.InboxUUID)
			record, err := resolveNotificationInbox(app.inbox, identity, false)
			if err != nil {
				return err
			}
			next.InboxUUID = &record.UUID
			next.EmailAddress = mailboxAddress(record.From)
		} else if mailboxAddress(next.EmailAddress) == "" {
			return notificationInput(app.i18n.T("validation.invalidFromAddress"))
		}
		// Deprecated compatibility field: never used for ticket routing.
		if _, explicitlySet := provided["notification.email.team_contact_email"]; explicitlySet && next.TeamContactEmail != nil {
			contact := strings.ToLower(strings.TrimSpace(*next.TeamContactEmail))
			if contact != "" && !stringutil.ValidEmail(contact) {
				return notificationInput("Invalid team-ticket recipient email address")
			}
			if contact != "" && contact == mailboxAddress(next.EmailAddress) {
				return notificationInput("Team-ticket recipient must differ from the sending mailbox address")
			}
			next.TeamContactEmail = &contact
		}
		return nil
	}, provided); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
