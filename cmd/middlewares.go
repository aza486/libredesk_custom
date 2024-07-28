package main

import (
	"net/http"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/simplesessions/v3"
)

func auth(handler fastglue.FastRequestHandler, requiredPerms ...string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app       = r.Context.(*App)
			sess, err = app.sess.Acquire(r.RequestCtx, r, r)
		)

		if err != nil {
			app.lo.Error("error acquiring session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		// User details in session?
		sessVals, err := sess.GetMulti("id", "email", "first_name", "last_name", "team_id")
		if err != nil && (err != simplesessions.ErrInvalidSession) {
			app.lo.Error("error fetching session", "error", err)
			return sendErrorEnvelope(r,
				envelope.NewError(envelope.GeneralError, "Error fetching session.", nil))
		}

		var (
			userID, _    = sess.Int(sessVals["id"], nil)
			email, _     = sess.String(sessVals["email"], nil)
			firstName, _ = sess.String(sessVals["first_name"], nil)
			lastName, _  = sess.String(sessVals["last_name"], nil)
			teamID, _    = sess.Int(sessVals["team_id"], nil)
		)

		if userID > 0 {
			// Fetch user perms.
			userPerms, err := app.user.GetPermissions(userID)
			if err != nil {
				return sendErrorEnvelope(r, err)
			}

			if !hasPerms(userPerms, requiredPerms) {
				return r.SendErrorEnvelope(http.StatusUnauthorized, "You don't have permissions to access this page.", nil, envelope.PermissionError)
			}

			// User is loggedin, Set user in the request context.
			r.RequestCtx.SetUserValue("user", umodels.User{
				ID:        userID,
				Email:     email,
				FirstName: firstName,
				LastName:  lastName,
				TeamID:    teamID,
			})

			return handler(r)
		}

		if err := sess.Destroy(); err != nil {
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, envelope.PermissionError)
		}
		return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, envelope.PermissionError)
	}
}

// hasPerms checks if all requiredPerms exist in userPerms.
func hasPerms(userPerms []string, requiredPerms []string) bool {
	userPermMap := make(map[string]bool)

	// make map for user's permissions for quick look up
	for _, perm := range userPerms {
		userPermMap[perm] = true
	}

	// iterate through required perms and if not found in userPermMap return false
	for _, requiredPerm := range requiredPerms {
		if _, ok := userPermMap[requiredPerm]; !ok {
			return false
		}
	}

	return true
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
	user, ok := r.RequestCtx.UserValue("user").(umodels.User)
	if user.ID == 0 || !ok {
		return user.ID, false
	}
	return user.ID, true
}

func sess(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var (
			app       = r.Context.(*App)
			sess, err = app.sess.Acquire(r.RequestCtx, r, r)
		)

		if err != nil {
			app.lo.Error("error acquiring session", "error", err)
			return r.SendErrorEnvelope(http.StatusUnauthorized, "invalid or expired session", nil, "PermissionException")
		}

		// User details in session?
		sessVals, err := sess.GetMulti("id", "email", "first_name", "last_name", "team_id")
		if err != nil && (err != simplesessions.ErrInvalidSession) {
			app.lo.Error("error fetching session", "error", err)
			return sendErrorEnvelope(r,
				envelope.NewError(envelope.GeneralError, "Error fetching session.", nil))
		}

		var (
			userID, _    = sess.Int(sessVals["id"], nil)
			email, _     = sess.String(sessVals["email"], nil)
			firstName, _ = sess.String(sessVals["first_name"], nil)
			lastName, _  = sess.String(sessVals["last_name"], nil)
			teamID, _    = sess.Int(sessVals["team_id"], nil)
		)

		if userID > 0 {
			// Set both in request context so they can be accessed in the handlers.
			// Set user in the request context.
			r.RequestCtx.SetUserValue("user", umodels.User{
				ID:        userID,
				Email:     email,
				FirstName: firstName,
				LastName:  lastName,
				TeamID:    teamID,
			})

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
