package main

import (
	"net/http"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func perm(handler fastglue.FastRequestHandler, obj, act string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var app = r.Context.(*App)

		user, err := app.auth.ValidateSession(r)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}

		// Fetch user and permissions from DB.
		user, err = app.user.Get(user.ID)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong", nil, envelope.GeneralError)
		}

		// Set user in the request context.
		r.RequestCtx.SetUserValue("user", user)

		// Enforce the permissions with the user, object, and action.
		ok, err := app.authz.Enforce(user, obj, act)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong", nil, envelope.GeneralError)
		}
		if !ok {
			return r.SendErrorEnvelope(http.StatusForbidden, "Permission denied", nil, envelope.PermissionError)
		}

		// Return handler.
		return handler(r)
	}
}

// authPage middleware makes sure user is logged in to access the page else redirects to login page.
func authPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		// Check if user is logged in. If logged in return next handler.
		userID, ok := getAuthUserFromSess(r)
		if ok && userID > 0 {
			return handler(r)
		}
		nextURI := r.RequestCtx.QueryArgs().Peek("next")
		if len(nextURI) == 0 {
			nextURI = r.RequestCtx.RequestURI()
		}
		return r.RedirectURI("/", fasthttp.StatusFound, map[string]interface{}{
			"next": string(nextURI),
		}, "")
	}
}

// getAuthUserFromSess retrives authUser from request context set by the sess() middleware.
func getAuthUserFromSess(r *fastglue.Request) (int, bool) {
	user, ok := r.RequestCtx.UserValue("user").(umodels.User)
	if user.ID == 0 || !ok {
		return user.ID, false
	}
	return user.ID, true
}

func sess(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var app = r.Context.(*App)
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}
		if user.ID >= 0 {
			r.RequestCtx.SetUserValue("user", user)
		}
		return handler(r)
	}
}

func authSess(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			userID, ok = getAuthUserFromSess(r)
		)
		if !ok || userID <= 0 {
			return sendErrorEnvelope(r,
				envelope.NewError(envelope.GeneralError, "Invalid or expired session.", nil))
		}
		return handler(r)
	}
}

func noAuthPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		_, ok := getAuthUserFromSess(r)
		if !ok {
			return handler(r)
		}
		// User is logged in direct if `next` is available else redirect.
		nextURI := string(r.RequestCtx.QueryArgs().Peek("next"))
		if len(nextURI) == 0 {
			nextURI = "/dashboard"
		}
		return r.RedirectURI(nextURI, fasthttp.StatusFound, nil, "")
	}
}
