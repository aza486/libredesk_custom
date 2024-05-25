package main

import (
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleLogin(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		email    = string(p.Peek("email"))
		password = p.Peek("password")
	)
	user, err := app.userDB.Login(email, password)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "GeneralException")
	}

	sess, err := app.sess.Acquire(r, r, nil)
	if err != nil {
		app.lo.Error("error acquiring session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error acquiring session.", nil, "GeneralException")
	}

	// Set email in the session.
	if err := sess.Set("user_email", email); err != nil {
		app.lo.Error("error setting session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error setting session.", nil, "GeneralException")
	}

	// Set user DB ID in the session.
	if err := sess.Set("user_id", user.ID); err != nil {
		app.lo.Error("error setting session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error setting session.", nil, "GeneralException")
	}

	if err := sess.Commit(); err != nil {
		app.lo.Error("error comitting session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error commiting session.", nil, "GeneralException")
	}

	agent, err := app.userDB.GetAgent(email)
	if err != nil {
		app.lo.Error("fetching agent", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error fetching agent.", nil, "GeneralException")
	}

	return r.SendEnvelope(agent)
}

func handleLogout(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	sess, err := app.sess.Acquire(r, r, nil)
	if err != nil {
		app.lo.Error("error acquiring session", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			"Error acquiring session.", nil, "GeneralException")
	}

	if err := sess.Clear(); err != nil {
		app.lo.Error("error clearing session", "error", err)
	}
	return r.SendEnvelope("ok")
}
