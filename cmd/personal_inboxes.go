package main

import (
	"slices"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

func handleGetOwnPersonalInboxes(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	inboxes, err := app.inbox.GetOwnPersonalInboxes(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(inboxes)
}

func handleUpdatePersonalInboxOwners(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	inboxID, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || inboxID <= 0 {
		return r.SendErrorEnvelope(400, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	var req struct {
		OwnerUserIDs []int64 `json:"owner_user_ids"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(400, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}
	actor, err := app.user.GetAgentCachedOrLoad(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	canManageAll := slices.Contains(actor.Permissions, "inboxes:manage")
	if err := app.inbox.UpdatePersonalInboxOwners(inboxID, user.ID, req.OwnerUserIDs, canManageAll); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleGetPersonalConversations(r *fastglue.Request) error {
	app := r.Context.(*App)
	user := r.RequestCtx.UserValue("user").(amodels.User)
	inboxID, _ := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	page, pageSize := getPagination(r)
	args := r.RequestCtx.QueryArgs()
	rows, err := app.conversation.GetPersonalConversationsList(user.ID, inboxID, string(args.Peek("priority")) == "high", string(args.Peek("order")), string(args.Peek("order_by")), string(args.Peek("filters")), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{Results: rows, Total: total, PerPage: pageSize, TotalPages: (total + pageSize - 1) / pageSize, Page: page})
}
