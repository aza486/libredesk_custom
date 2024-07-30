package main

import (
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/zerodha/fastglue"
)

// handleLogin logs in the user.
func handleLogin(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		email    = string(p.Peek("email"))
		password = p.Peek("password")
	)

	user, err := app.user.Login(email, password)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.auth.SaveSession(user, r); err != nil {
		app.lo.Error("error saving session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	return r.SendEnvelope(user)
}

// handleLogout logs out the user.
func handleLogout(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	if err := app.auth.DestroySession(r); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	return r.SendEnvelope(true)
}
