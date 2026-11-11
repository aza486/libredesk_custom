package main

import (
	"slices"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/fastglue"
)

func canManageServiceEmails(user umodels.User) bool {
	return user.HasAdminRole() || slices.Contains(user.Permissions, "general_settings:manage") || slices.Contains(user.Permissions, "service_emails:manage")
}

// Authentication and CSRF remain identical to the existing administration API.
func serviceEmailManagement(handler fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return auth(func(r *fastglue.Request) error {
		app := r.Context.(*App)
		caller := r.RequestCtx.UserValue("user").(amodels.User)
		user, err := app.user.GetAgent(caller.ID, "")
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if !canManageServiceEmails(user) {
			return r.SendErrorEnvelope(403, "Service email management permission required", nil, envelope.PermissionError)
		}
		return handler(r)
	})
}
