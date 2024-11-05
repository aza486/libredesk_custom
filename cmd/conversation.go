package main

import (
	"encoding/json"
	"strconv"

	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleGetAllConversations retrieves all conversations.
func handleGetAllConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total       = 0
	)
	conversations, pageSize, err := app.conversation.GetAllConversationsList(order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetAssignedConversations retrieves conversations assigned to the current user.
func handleGetAssignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(umodels.User)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total       = 0
	)
	conversations, pageSize, err := app.conversation.GetAssignedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetUnassignedConversations retrieves unassigned conversations.
func handleGetUnassignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(umodels.User)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		total       = 0
	)
	conversations, pageSize, err := app.conversation.GetUnassignedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}
	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetConversation retrieves a single conversation by UUID with permission checks.
func handleGetConversation(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	conversation, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(conversation)
}

// handleUpdateConversationAssigneeLastSeen updates the assignee's last seen timestamp for a conversation.
func handleUpdateConversationAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err = app.conversation.UpdateConversationAssigneeLastSeen(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetConversationParticipants retrieves participants of a conversation.
func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	p, err := app.conversation.GetConversationParticipants(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(p)
}

// handleUpdateConversationUserAssignee updates the user assigned to a conversation.
func handleUpdateConversationUserAssignee(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)
	assigneeID, err := r.RequestCtx.PostArgs().GetUint("assignee_id")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid assignee `id`.", nil, envelope.InputError)
	}

	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.UpdateConversationUserAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.automation.EvaluateConversationUpdateRules(uuid)
	return r.SendEnvelope(true)
}

// handleUpdateTeamAssignee updates the team assigned to a conversation.
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
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.UpdateConversationTeamAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.automation.EvaluateConversationUpdateRules(uuid)
	return r.SendEnvelope(true)
}

// handleUpdateConversationPriority updates the priority of a conversation.
func handleUpdateConversationPriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		priority = p.Peek("priority")
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		user     = r.RequestCtx.UserValue("user").(umodels.User)
	)
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.UpdateConversationPriority(uuid, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.automation.EvaluateConversationUpdateRules(uuid)
	return r.SendEnvelope(true)
}

// handleUpdateConversationStatus updates the status of a conversation.
func handleUpdateConversationStatus(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		p      = r.RequestCtx.PostArgs()
		status = p.Peek("status")
		uuid   = r.RequestCtx.UserValue("uuid").(string)
		user   = r.RequestCtx.UserValue("user").(umodels.User)
	)
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.conversation.UpdateConversationStatus(uuid, status, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.automation.EvaluateConversationUpdateRules(uuid)
	return r.SendEnvelope(true)
}

// handleAddConversationTags adds tags to a conversation.
func handleAddConversationTags(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		p       = r.RequestCtx.PostArgs()
		tagIDs  = []int{}
		tagJSON = p.Peek("tag_ids")
		user    = r.RequestCtx.UserValue("user").(umodels.User)
		uuid    = r.RequestCtx.UserValue("uuid").(string)
	)

	// Parse tag IDs from JSON
	err := json.Unmarshal(tagJSON, &tagIDs)
	if err != nil {
		app.lo.Error("unmarshalling tag ids", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error adding tags", nil, "")
	}

	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := app.conversation.UpsertConversationTags(uuid, tagIDs); err != nil {
		return sendErrorEnvelope(r, err)
	}
	app.automation.EvaluateConversationUpdateRules(uuid)
	return r.SendEnvelope(true)
}

// handleDashboardCounts retrieves general dashboard counts for all users.
func handleDashboardCounts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	counts, err := app.conversation.GetDashboardCounts(0, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(counts)
}

// handleDashboardCharts retrieves general dashboard chart data.
func handleDashboardCharts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	charts, err := app.conversation.GetDashboardChart(0, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(charts)
}

// enforceConversationAccess fetches the conversation and checks if the user has access to it.
func enforceConversationAccess(app *App, uuid string, user umodels.User) (*cmodels.Conversation, error) {
	conversation, err := app.conversation.GetConversation(uuid)
	if err != nil {
		return nil, err
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return nil, envelope.NewError(envelope.GeneralError, "Error checking permissions", nil)
	}
	if !allowed {
		return nil, envelope.NewError(envelope.PermissionError, "Permission denied", nil)
	}
	return &conversation, nil
}
