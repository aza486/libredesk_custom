package main

import (
	"fmt"
	"net/http"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func aauth(handler fastglue.FastRequestHandler, requiredPerms ...string) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		var app = r.Context.(*App)
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			return r.SendErrorEnvelope(http.StatusUnauthorized, "Invalid or expired session", nil, envelope.PermissionError)
		}

		fmt.Println("req ", requiredPerms)

		// User is loggedin, Set user in the request context.
		r.RequestCtx.SetUserValue("user", user)
		return handler(r)
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
		var app = r.Context.(*App)
		user, err := app.auth.ValidateSession(r)
		if err != nil {
			return handler(r)
		}
		// User is loggedin, Set user in the request context.
		r.RequestCtx.SetUserValue("user", user)
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
