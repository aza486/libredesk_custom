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

// handleGetAllConversations retrieves all conversations with pagination, sorting, and filtering.
func handleGetAllConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filter      = string(r.RequestCtx.QueryArgs().Peek("filter"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	)
	// Fetch all conversations
	c, err := app.conversation.GetAllConversations(order, orderBy, filter, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(c)
}

// handleGetAssignedConversations retrieves conversations assigned to the current user.
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
	// Fetch conversations assigned to the user
	c, err := app.conversation.GetAssignedConversations(user.ID, order, orderBy, filter, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(c)
}

// handleGetTeamConversations retrieves team-specific conversations.
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
	// Fetch conversations assigned to the user's team
	c, err := app.conversation.GetTeamConversations(user.ID, order, orderBy, filter, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(c)
}

// handleGetConversation retrieves a single conversation by UUID with permission checks.
func handleGetConversation(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Check if the user has access to the conversation
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

	// Check permission
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update last seen
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

	// Check permission
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Fetch participants
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

	// Check permission
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update the user assignee
	if err := app.conversation.UpdateConversationUserAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
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

	// Check permission
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update the team assignee
	if err := app.conversation.UpdateConversationTeamAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
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

	// Check permission
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update priority
	if err := app.conversation.UpdateConversationPriority(uuid, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
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

	// Check permission
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Update status
	if err := app.conversation.UpdateConversationStatus(uuid, status, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
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

	// Check permission
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Upsert tags
	if err := app.conversation.UpsertConversationTags(uuid, tagIDs); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleUserDashboardCounts retrieves dashboard statistics for the current user.
func handleUserDashboardCounts(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Fetch dashboard counts
	stats, err := app.conversation.GetDashboardCounts(user.ID, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

// handleUserDashboardCharts retrieves chart data for the current user's dashboard.
func handleUserDashboardCharts(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		user = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Fetch dashboard chart data
	stats, err := app.conversation.GetDashboardChartData(user.ID, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

// handleDashboardCounts retrieves general dashboard counts for all users.
func handleDashboardCounts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	// Fetch dashboard counts
	stats, err := app.conversation.GetDashboardCounts(0, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

// handleDashboardCharts retrieves general dashboard chart data.
func handleDashboardCharts(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	// Fetch dashboard chart data
	stats, err := app.conversation.GetDashboardChartData(0, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
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
