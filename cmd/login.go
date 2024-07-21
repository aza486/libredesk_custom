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

	user, err := app.userManager.Login(email, password)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	sess, err := app.sessManager.Acquire(r.RequestCtx, r, r)
	if err != nil {
		app.lo.Error("error acquiring session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}

	// Set user details in the session.
	if err := sess.SetMulti(map[string]interface{}{
		"id":          user.ID,
		"email":       user.Email,
		"first_name":  user.FirstName,
		"last_name":   user.LastName,
		"team_id":     user.TeamID,
		"permissions": user.Permissions,
	}); err != nil {
		app.lo.Error("error setting values in session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}

	return r.SendEnvelope(user)
}

// handleLogout logs out the user.
func handleLogout(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	sess, err := app.sessManager.Acquire(r.RequestCtx, r, r)
	if err != nil {
		app.lo.Error("error acquiring session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	if err := sess.Destroy(); err != nil {
		app.lo.Error("error clearing session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.errorAcquiringSession"), nil))
	}
	return r.SendEnvelope(true)
}
