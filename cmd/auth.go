package main

import (
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleOIDCLogin redirects to the OIDC provider for login.
func handleOIDCLogin(r *fastglue.Request) error {
	var (
		app             = r.Context.(*App)
		providerID, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		csrfToken       = string(r.RequestCtx.Request.Header.Cookie("csrf_token"))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error parsing provider id.", nil, envelope.GeneralError)
	}
	authURL, err := app.auth.LoginURL(providerID, csrfToken)
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
		providerID, err = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("id")))
		csrfToken       = string(r.RequestCtx.Request.Header.Cookie("csrf_token"))
	)
	if err != nil {
		app.lo.Error("error parsing provider id", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error parsing provider id.", nil, envelope.GeneralError)
	}

	_, claims, err := app.auth.ExchangeOIDCToken(r.RequestCtx, providerID, code, csrfToken)
	if err != nil {
		app.lo.Error("error exchanging oidc token", "error", err)
		return err
	}

	// Get user by e-mail received.
	user, err := app.user.GetByEmail(claims.Email)
	if err != nil {
		return err
	}

	// Set the session.
	if err := app.auth.SaveSession(user, r); err != nil {
		return err
	}

	return r.Redirect(state, fasthttp.StatusFound, nil, "")
}
