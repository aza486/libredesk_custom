package main

import (
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

func personalContactAccess(next fastglue.FastRequestHandler) fastglue.FastRequestHandler {
	return func(r *fastglue.Request) error {
		app := r.Context.(*App)
		viewer, err := app.user.GetAgentCachedOrLoad(r.RequestCtx.UserValue("user").(amodels.User).ID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		id, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		allowed, err := app.user.CanAccessContact(id, viewer)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if !allowed {
			return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, app.i18n.T("status.deniedPermission"), nil))
		}
		return next(r)
	}
}
