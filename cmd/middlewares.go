package main

import (
	"net/http"

	"github.com/vividvilla/simplesessions"
	"github.com/zerodha/fastglue"
)

func auth(handler fastglue.FastRequestHandler, perms ...string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app       = r.Context.(*App)
			sess, err = app.sessMgr.Acquire(r, r, nil)
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
