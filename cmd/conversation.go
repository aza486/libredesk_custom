package main

import (
	"encoding/json"
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetAllConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filter      = string(r.RequestCtx.QueryArgs().Peek("filter"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	)
	c, err := app.conversation.GetAllConversations(order, orderBy, filter, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

func handleGetAssignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(umodels.User)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filter      = string(r.RequestCtx.QueryArgs().Peek("filter"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	)
	c, err := app.conversation.GetAssignedConversations(user.ID, order, orderBy, filter, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(c)
}

func handleGetTeamConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(umodels.User)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filter      = string(r.RequestCtx.QueryArgs().Peek("filter"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	)
	c, err := app.conversation.GetTeamConversations(user.ID, order, orderBy, filter, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(c)
}

func handleGetConversation(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	c, err := app.conversation.GetConversation(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

func handleUpdateAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	err := app.conversation.UpdateConversationAssigneeLastSeen(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	p, err := app.conversation.GetConversationParticipants(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

func handleUpdateUserAssignee(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	assigneeID, err := r.RequestCtx.PostArgs().GetUint("assignee_id")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid assignee `id`.", nil, envelope.InputError)
	}
	if err := app.conversation.UpdateConversationUserAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdateTeamAssignee(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	assigneeID, err := r.RequestCtx.PostArgs().GetUint("assignee_id")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid assignee `id`.", nil, envelope.InputError)
	}
	if err := app.conversation.UpdateConversationTeamAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdatePriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		priority = p.Peek("priority")
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		user     = r.RequestCtx.UserValue("user").(umodels.User)
	)
	if err := app.conversation.UpdateConversationPriority(uuid, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUpdateStatus(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		p      = r.RequestCtx.PostArgs()
		status = p.Peek("status")
		uuid   = r.RequestCtx.UserValue("uuid").(string)
		user   = r.RequestCtx.UserValue("user").(umodels.User)
	)
	if err := app.conversation.UpdateConversationStatus(uuid, status, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleAddConversationTags(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		p       = r.RequestCtx.PostArgs()
		tagIDs  = []int{}
		tagJSON = p.Peek("tag_ids")
		uuid    = r.RequestCtx.UserValue("uuid").(string)
	)
	err := json.Unmarshal(tagJSON, &tagIDs)
	if err != nil {
		app.lo.Error("unmarshalling tag ids", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error adding tags", nil, "")
	}
	if err := app.conversation.UpsertConversationTags(uuid, tagIDs); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleUserDashboardCounts(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	stats, err := app.conversation.GetConversationAssigneeStats(user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

func handleUserDashboardCharts(r *fastglue.Request) error {
	var app = r.Context.(*App)
	stats, err := app.conversation.GetNewConversationsStats()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}
