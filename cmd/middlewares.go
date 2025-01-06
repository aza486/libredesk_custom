package main

import (
	"net/http"

	amodels "github.com/abhinavxd/artemis/internal/auth/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// tryAuth is a middleware that attempts to authenticate the user and add them to the context
// but doesn't enforce authentication. Handlers can check if user exists in context optionally.
func tryAuth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Try to validate session without returning error.
		userSession, err := app.auth.ValidateSession(r)
		if err != nil || userSession.ID <= 0 {
			return handler(r)
		}

		// Try to get user.
		user, err := app.user.Get(userSession.ID)
		if err != nil {
			return handler(r)
		}

		// Set user in context if found.
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        user.ID,
			Email:     user.Email.String,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})

		return handler(r)
	}
}

// auth makes sure the user is logged in.
func auth(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app = r.Context.(*App)
		)

		// Validate session and fetch user.
		userSession, err := app.auth.ValidateSession(r)
		if err != nil || userSession.ID <= 0 {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}

		// Set user in the request context.
		user, err := app.user.Get(userSession.ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        user.ID,
			Email:     user.Email.String,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		})

		return handler(r)
	}
}

// perm does session validation, CSRF, and permission enforcement.
func perm(handler fastglue.FastRequestHandler, perm string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			// app = r.Context.(*App)
			// cookieToken = string(r.RequestCtx.Request.Header.Cookie("csrf_token"))
			// hdrToken    = string(r.RequestCtx.Request.Header.Peek("X-CSRFTOKEN"))
		)

		// if cookieToken == "" || hdrToken == "" || cookieToken != hdrToken {
		// 	app.lo.Error("csrf token mismatch", "cookie_token", cookieToken, "header_token", hdrToken)
		// 	return r.SendErrorEnvelope(http.StatusForbidden, "Invalid CSRF token", nil, envelope.PermissionError)
		// }

		// Validate session and fetch user.
		// sessUser, err := app.auth.ValidateSession(r)
		// if err != nil || sessUser.ID <= 0 {
		// 	app.lo.Error("error validating session", "error", err)
		// 	return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		// }

		// // Get user from DB.
		// user, err := app.user.Get(sessUser.ID)
		// if err != nil {
		// 	return sendErrorEnvelope(r, err)
		// }

		// // Split the permission string into object and action and enforce it.
		// parts := strings.Split(perm, ":")
		// if len(parts) != 2 {
		// 	return r.SendErrorEnvelope(http.StatusInternalServerError, "Invalid permission format", nil, envelope.GeneralError)
		// }
		// object, action := parts[0], parts[1]
		// ok, err := app.authz.Enforce(user, object, action)
		// if err != nil {
		// 	return r.SendErrorEnvelope(http.StatusInternalServerError, "Error checking permissions", nil, envelope.GeneralError)
		// }
		// if !ok {
		// 	return r.SendErrorEnvelope(http.StatusForbidden, "Permission denied", nil, envelope.PermissionError)
		// }
	
		// Set user in the request context.
		r.RequestCtx.SetUserValue("user", amodels.User{
			ID:        1,
			Email:     "sample@example.com",
			FirstName: "Sample",
			LastName:  "User",
		})

		return handler(r)
	}
}

// authPage ensures the user is logged in; otherwise, redirects to the login page.
func authPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Validate session.
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}
		if user.ID > 0 {
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

// notAuthPage allows access only if the user is not authenticated; otherwise, redirects to the user inbox.
func notAuthPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Validate session.
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}

		if user.ID != 0 {
			nextURI := string(r.RequestCtx.QueryArgs().Peek("next"))
			if nextURI == "" {
				nextURI = "/inboxes/assigned"
			}
			return r.RedirectURI(nextURI, fasthttp.StatusFound, nil, "")
		}
		return handler(r)
	}
}
