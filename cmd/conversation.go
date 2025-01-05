package main

import (
	"encoding/json"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/artemis/internal/auth/models"
	"github.com/abhinavxd/artemis/internal/automation/models"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
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

	conversations, err := app.conversation.GetAllConversationsList(order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	// Calculate SLA deadlines if conversation has an SLA policy.
	for i := range conversations {
		if conversations[i].SLAPolicyID.Int != 0 {
			calculateSLA(app, &conversations[i])
		}
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
		user        = r.RequestCtx.UserValue("user").(amodels.User)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	conversations, err := app.conversation.GetAssignedConversationsList(user.ID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	// Calculate SLA deadlines if conversation has an SLA policy.
	for i := range conversations {
		if conversations[i].SLAPolicyID.Int != 0 {
			calculateSLA(app, &conversations[i])
		}
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
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)

	conversations, err := app.conversation.GetUnassignedConversationsList(order, orderBy, filters, page, pageSize)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, err.Error(), nil, "")
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	// Calculate SLA deadlines if conversation has an SLA policy.
	for i := range conversations {
		if conversations[i].SLAPolicyID.Int != 0 {
			calculateSLA(app, &conversations[i])
		}
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetViewConversations retrieves conversations for a view.
func handleGetViewConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(amodels.User)
		viewID, _   = strconv.Atoi(r.RequestCtx.UserValue("view_id").(string))
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	if viewID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `view_id`", nil, envelope.InputError)
	}

	// Check if user has access to the view.
	view, err := app.view.Get(viewID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if view.UserID != user.ID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You don't have access to this view.", nil, envelope.PermissionError)
	}

	conversations, err := app.conversation.GetViewConversationsList(user.ID, view.InboxType, order, orderBy, string(view.Filters), page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	// Calculate SLA deadlines if conversation has an SLA policy.
	for i := range conversations {
		if conversations[i].SLAPolicyID.Int != 0 {
			calculateSLA(app, &conversations[i])
		}
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}

// handleGetTeamUnassignedConversations returns conversations assigned to a team but not to any user.
func handleGetTeamUnassignedConversations(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		user        = r.RequestCtx.UserValue("user").(amodels.User)
		teamIDStr   = r.RequestCtx.UserValue("team_id").(string)
		order       = string(r.RequestCtx.QueryArgs().Peek("order"))
		orderBy     = string(r.RequestCtx.QueryArgs().Peek("order_by"))
		filters     = string(r.RequestCtx.QueryArgs().Peek("filters"))
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)
	teamID, _ := strconv.Atoi(teamIDStr)
	if teamID < 1 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `team_id`", nil, envelope.InputError)
	}

	// Check if user belongs to the team.
	exists, err := app.team.UserBelongsToTeam(teamID, user.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if !exists {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "You're not a member of this team, Please refresh the page and try again.", nil))
	}

	conversations, err := app.conversation.GetTeamUnassignedConversationsList(teamID, order, orderBy, filters, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	// Calculate SLA deadlines if conversation has an SLA policy.
	for i := range conversations {
		if conversations[i].SLAPolicyID.Int != 0 {
			calculateSLA(app, &conversations[i])
		}
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
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}

	// Calculate SLA deadlines if conversation has an SLA policy.
	if conversation.SLAPolicyID.Int != 0 {
		calculateSLA(app, &conversation)
	}
	return r.SendEnvelope(conversation)
}

// handleUpdateConversationAssigneeLastSeen updates the assignee's last seen timestamp for a conversation.
func handleUpdateConversationAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}
	if err = app.conversation.UpdateConversationAssigneeLastSeen(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleGetConversationParticipants retrieves participants of a conversation.
func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
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
		app        = r.Context.(*App)
		uuid       = r.RequestCtx.UserValue("uuid").(string)
		auser      = r.RequestCtx.UserValue("user").(amodels.User)
		assigneeID = r.RequestCtx.PostArgs().GetUintOrZero("assignee_id")
	)
	if assigneeID == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `assignee_id`", nil, envelope.InputError)
	}

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}

	if err := app.conversation.UpdateConversationUserAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(uuid, models.EventConversationUserAssigned)

	return r.SendEnvelope(true)
}

// handleUpdateTeamAssignee updates the team assigned to a conversation.
func handleUpdateTeamAssignee(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	assigneeID, err := r.RequestCtx.PostArgs().GetUint("assignee_id")
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid assignee `id`.", nil, envelope.InputError)
	}

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}
	if err := app.conversation.UpdateConversationTeamAssignee(uuid, assigneeID, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(uuid, models.EventConversationTeamAssigned)
	return r.SendEnvelope(true)
}

// handleUpdateConversationPriority updates the priority of a conversation.
func handleUpdateConversationPriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		priority = p.Peek("priority")
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		auser    = r.RequestCtx.UserValue("user").(amodels.User)
	)
	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}
	if err := app.conversation.UpdateConversationPriority(uuid, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(uuid, models.EventConversationPriorityChange)

	return r.SendEnvelope(true)
}

// handleUpdateConversationStatus updates the status of a conversation.
func handleUpdateConversationStatus(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		p            = r.RequestCtx.PostArgs()
		status       = p.Peek("status")
		snoozedUntil = p.Peek("snoozed_until")
		uuid         = r.RequestCtx.UserValue("uuid").(string)
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
	)
	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}
	if err := app.conversation.UpdateConversationStatus(uuid, status, snoozedUntil, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(uuid, models.EventConversationStatusChange)
	return r.SendEnvelope(true)
}

// handleAddConversationTags adds tags to a conversation.
func handleAddConversationTags(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		p       = r.RequestCtx.PostArgs()
		tagIDs  = []int{}
		tagJSON = p.Peek("tag_ids")
		auser   = r.RequestCtx.UserValue("user").(amodels.User)
		uuid    = r.RequestCtx.UserValue("uuid").(string)
	)

	// Parse tag IDs from JSON
	err := json.Unmarshal(tagJSON, &tagIDs)
	if err != nil {
		app.lo.Error("unmarshalling tag ids", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error adding tags", nil, "")
	}

	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	allowed, err := app.authz.EnforceConversationAccess(user, conversation)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !allowed {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}

	if err := app.conversation.UpsertConversationTags(uuid, tagIDs); err != nil {
		return sendErrorEnvelope(r, err)
	}
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
	conversation, err := app.conversation.GetConversation(0, uuid)
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

// calculateSLA calculates the SLA deadlines and sets them on the conversation.
func calculateSLA(app *App, conversation *cmodels.Conversation) error {
	firstRespAt, resolutionDueAt, err := app.sla.CalculateConversationDeadlines(conversation.CreatedAt, conversation.AssignedTeamID.Int, conversation.SLAPolicyID.Int)
	if err != nil {
		app.lo.Error("error calculating SLA deadlines for conversation", "id", conversation.ID, "error", err)
		return err
	}
	conversation.FirstReplyDueAt = null.NewTime(firstRespAt, firstRespAt != time.Time{})
	conversation.ResolutionDueAt = null.NewTime(resolutionDueAt, resolutionDueAt != time.Time{})
	return nil
}
