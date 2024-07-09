package main

import (
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/vividvilla/simplesessions"
	"github.com/zerodha/fastglue"
)

func auth(handler fastglue.FastRequestHandler, perms ...string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app       = r.Context.(*App)
			sess, err = app.sessManager.Acquire(r, r, nil)
		)

		if err != nil {
			app.lo.Error("error acquiring session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		// User email in session?
		email, err := sess.String(sess.Get("user_email"))
		if err != nil && (err != simplesessions.ErrInvalidSession && err != simplesessions.ErrFieldNotFound) {
			app.lo.Error("error fetching session session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		// User ID in session?
		userID, err := sess.Int(sess.Get("user_id"))
		if err != nil && (err != simplesessions.ErrInvalidSession && err != simplesessions.ErrFieldNotFound) {
			app.lo.Error("error fetching session session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		userUUID, err := sess.String(sess.Get("user_uuid"))
		if err != nil && (err != simplesessions.ErrInvalidSession && err != simplesessions.ErrFieldNotFound) {
			app.lo.Error("error fetching session session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		if email != "" && userID > 0 {
			// Set both in request context so they can be accessed in the handlers.
			r.RequestCtx.SetUserValue("user_email", email)
			r.RequestCtx.SetUserValue("user_id", userID)
			r.RequestCtx.SetUserValue("user_uuid", userUUID)

			// Check permission.
			for _, perm := range perms {
				hasPerm, err := app.rbac.HasPermission(userID, perm)
				if err != nil || !hasPerm {
					return r.SendErrorEnvelope(http.StatusUnauthorized, "You don't have permissions to access this page.", nil, "PermissionException")
				}
			}

			return handler(r)
		}

		if err := sess.Clear(); err != nil {
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}
		return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
	}
}

// authPage middleware makes sure user is logged in to access the page
// else redirects to login page.
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
	userID, ok := r.RequestCtx.UserValue("user_id").(int)
	if userID == 0 || !ok {
		return userID, false
	}
	return userID, true
}

func sess(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app       = r.Context.(*App)
			sess, err = app.sessManager.Acquire(r, r, nil)
		)

		if err != nil {
			app.lo.Error("error acquiring session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		// User email in session?
		email, err := sess.String(sess.Get("user_email"))
		if err != nil && (err != simplesessions.ErrInvalidSession && err != simplesessions.ErrFieldNotFound) {
			app.lo.Error("error fetching session session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		// User ID in session?
		userID, err := sess.Int(sess.Get("user_id"))
		if err != nil && (err != simplesessions.ErrInvalidSession && err != simplesessions.ErrFieldNotFound) {
			app.lo.Error("error fetching session session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		userUUID, err := sess.String(sess.Get("user_uuid"))
		if err != nil && (err != simplesessions.ErrInvalidSession && err != simplesessions.ErrFieldNotFound) {
			app.lo.Error("error fetching session session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		if email != "" && userID > 0 {
			// Set both in request context so they can be accessed in the handlers.
			r.RequestCtx.SetUserValue("user_email", email)
			r.RequestCtx.SetUserValue("user_id", userID)
			r.RequestCtx.SetUserValue("user_uuid", userUUID)
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

		// User is logged in direct if `next` is available else redirect to the dashboard.
		nextURI := string(r.RequestCtx.QueryArgs().Peek("next"))
		if len(nextURI) == 0 {
			nextURI = "/dashboard"
		}

		return r.RedirectURI(nextURI, fasthttp.StatusFound, nil, "")
	}
}
