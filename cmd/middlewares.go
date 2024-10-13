package main

import (
	"net/http"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// authMiddleware does session validation, CSRF checking, and permission enforcement.
func authMiddleware(handler fastglue.FastRequestHandler, object, action string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)

		// Validate session and fetch user.
		userSession, err := app.auth.ValidateSession(r)
		if err != nil {
			app.lo.Error("error validating session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}

		user, err := app.user.Get(userSession.ID)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong", nil, envelope.GeneralError)
		}

		// CSRF check.
		cookieToken := string(r.RequestCtx.Request.Header.Cookie("csrf_token"))
		hdrToken := string(r.RequestCtx.Request.Header.Peek("X-CSRFTOKEN"))
		if cookieToken == "" || hdrToken == "" || cookieToken != hdrToken {
			return r.SendErrorEnvelope(http.StatusForbidden, "Invalid CSRF token", nil, envelope.PermissionError)
		}

		// Permission enforcement.
		if object != "" && action != "" {
			ok, err := app.authz.Enforce(user, object, action)
			if err != nil {
				return r.SendErrorEnvelope(http.StatusInternalServerError, "Error checking permissions", nil, envelope.GeneralError)
			}
			if !ok {
				return r.SendErrorEnvelope(http.StatusForbidden, "Permission denied", nil, envelope.PermissionError)
			}
		}

		// Set user in the request context.
		r.RequestCtx.SetUserValue("user", user)

		// Proceed to the next handler.
		return handler(r)
	}
}

// getUserFromContext retrieves the authenticated user from the request context.
func getUserFromContext(r *fastglue.Request) (umodels.User, bool) {
	user, ok := r.RequestCtx.UserValue("user").(umodels.User)
	return user, ok
}

// authenticatedPage ensures the user is logged in; otherwise, redirects to the login page.
func authenticatedPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		user, ok := getUserFromContext(r)
		if ok && user.ID > 0 {
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

// notAuthenticatedPage allows access only if the user is not authenticated; otherwise, redirects to the dashboard.
func notAuthenticatedPage(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		user, _ := getUserFromContext(r)
		if user.ID != 0 {
			nextURI := string(r.RequestCtx.QueryArgs().Peek("next"))
			if nextURI == "" {
				nextURI = "/dashboard"
			}
			return r.RedirectURI(nextURI, fasthttp.StatusFound, nil, "")
		}
		return handler(r)
	}
}
