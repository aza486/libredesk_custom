package main

import (
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/zerodha/fastglue"
)

func handleLogin(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		email    = string(p.Peek("email"))
		password = p.Peek("password")
	)

	user, err := app.userManager.Login(email, password)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	sess, err := app.sessManager.Acquire(r, r, nil)
	if err != nil {
		app.lo.Error("error acquiring session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}

	// Set email in the session.
	if err := sess.Set("user_email", email); err != nil {
		app.lo.Error("error setting session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSettingSession"), nil))
	}

	// Set user DB ID in the session.
	if err := sess.Set("user_id", user.ID); err != nil {
		app.lo.Error("error setting session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSettingSession"), nil))
	}

	// Set user UUID in the session.
	if err := sess.Set("user_uuid", user.UUID); err != nil {
		app.lo.Error("error setting session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSettingSession"), nil))
	}

	// Commit session.
	if err := sess.Commit(); err != nil {
		app.lo.Error("error comitting session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSettingSession"), nil))
	}

	// Fetch & return the user details.
	user, err = app.userManager.GetUser(user.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorSettingSession"), nil))
	}

	return r.SendEnvelope(user)
}

func handleLogout(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	sess, err := app.sessManager.Acquire(r, r, nil)
	if err != nil {
		app.lo.Error("error acquiring session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	if err := sess.Clear(); err != nil {
		app.lo.Error("error clearing session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	return r.SendEnvelope("ok")
}
