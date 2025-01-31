package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleOIDCLogin redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error parsing provider id.", nil, envelope.GeneralError)
	}

	// Set a state and save it in the session, to prevent CSRF attacks.
	state, err := stringutil.RandomAlNumString(32)
	if err != nil {
		app.lo.Error("error generating state", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error generating state.", nil, envelope.GeneralError)
	}
	if err = app.auth.SetSessionValues(r, map[string]interface{}{
		"oidc_state": state,
	}); err != nil {
		app.lo.Error("error saving state in session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error saving state in session.", nil, envelope.GeneralError)
	}

	authURL, err := app.auth.LoginURL(providerID, state)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.Redirect(authURL, fasthttp.StatusFound, nil, "")
}

// handleOIDCCallback receives the redirect callback from the OIDC provider and completes the handshake.
func handleOIDCCallback(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		code            = string(r.RequestCtx.QueryArgs().Peek("code"))
		state           = string(r.RequestCtx.QueryArgs().Peek("state"))
		providerID, err = strconv.Atoi(string(r.RequestCtx.UserValue("id").(string)))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error parsing provider id.", nil, envelope.GeneralError)
	}

	// Compare the state from the session with the state from the query.
	sessionState, err := app.auth.GetSessionValue(r, "oidc_state")
	if err != nil {
		app.lo.Error("error getting state from session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error getting state from session.", nil, envelope.GeneralError)
	}
	if state != sessionState {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Invalid state.", nil, envelope.GeneralError)
	}

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code)
	if err != nil {
		app.lo.Error("error exchanging oidc token", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error exchanging OIDC token.", nil, envelope.GeneralError)
	}

	// Lookup the user by email and set the session.
	user, err := app.user.GetByEmail(claims.Email)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error saving session.", nil, envelope.GeneralError)
	}

	return r.Redirect("/", fasthttp.StatusFound, nil, "")
}
