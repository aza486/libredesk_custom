package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
	"strconv"
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
