package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/automation/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/csat"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
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
		auser       = r.RequestCtx.UserValue("user").(amodels.User)
		viewID, _   = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
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
	if view.UserID != auser.ID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "You don't have access to this view.", nil, envelope.PermissionError)
	}

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Prepare lists user has access to based on user permissions, internally this affects the SQL query.
	lists := []string{}
	for _, perm := range user.Permissions {
		if perm == authzModels.PermConversationsReadAll {
			// No further lists required as user has access to all conversations.
			lists = []string{cmodels.AllConversations}
			break
		}
		if perm == authzModels.PermConversationsReadUnassigned {
			lists = append(lists, cmodels.UnassignedConversations)
		}
		if perm == authzModels.PermConversationsReadAssigned {
			lists = append(lists, cmodels.AssignedConversations)
		}
		if perm == authzModels.PermConversationsReadTeamInbox {
			lists = append(lists, cmodels.TeamUnassignedConversations)
		}
	}

	// No lists found, user doesn't have access to any conversations.
	if len(lists) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Permission denied", nil, envelope.PermissionError)
	}

	conversations, err := app.conversation.GetViewConversationsList(user.ID, user.Teams.IDs(), lists, order, orderBy, string(view.Filters), page, pageSize)
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
		auser       = r.RequestCtx.UserValue("user").(amodels.User)
		teamIDStr   = r.RequestCtx.UserValue("id").(string)
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
	exists, err := app.team.UserBelongsToTeam(teamID, auser.ID)
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
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		auser    = r.RequestCtx.UserValue("user").(amodels.User)
		priority = string(r.RequestCtx.PostArgs().Peek("priority"))
	)
	if priority == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `priority`", nil, envelope.InputError)
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
	if err := app.conversation.UpdateConversationPriority(uuid, 0 /**priority_id**/, priority, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(uuid, models.EventConversationPriorityChange)
	return r.SendEnvelope("Priority updated successfully")
}

// handleUpdateConversationStatus updates the status of a conversation.
func handleUpdateConversationStatus(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		status       = string(r.RequestCtx.PostArgs().Peek("status"))
		snoozedUntil = string(r.RequestCtx.PostArgs().Peek("snoozed_until"))
		uuid         = r.RequestCtx.UserValue("uuid").(string)
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if status == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `status`", nil, envelope.InputError)
	}

	if snoozedUntil == "" && status == cmodels.StatusSnoozed {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `snoozed_until`", nil, envelope.InputError)
	}

	conversation, err := app.conversation.GetConversation(0, uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if status == cmodels.StatusResolved && conversation.AssignedUserID.Int == 0 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Cannot resolve the conversation without an assigned user, Please assign a user before attempting to resolve.", nil))
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

	if err := app.conversation.UpdateConversationStatus(uuid, 0 /**status_id**/, status, snoozedUntil, user); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(uuid, models.EventConversationStatusChange)

	// If status is `Resolved`, send CSAT survey if enabled on inbox.
	if status == cmodels.StatusResolved {
		if err := sendCSATSurvey(app, conversation, user); err != nil {
			return sendErrorEnvelope(r, err)
		}
	}
	return r.SendEnvelope("Status updated successfully")
}

// handleUpdateConversationtags updates conversation tags.
func handleUpdateConversationtags(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		tagNames = []string{}
		tagJSON  = r.RequestCtx.PostArgs().Peek("tags")
		auser    = r.RequestCtx.UserValue("user").(amodels.User)
		uuid     = r.RequestCtx.UserValue("uuid").(string)
	)

	if err := json.Unmarshal(tagJSON, &tagNames); err != nil {
		app.lo.Error("error unmarshalling tags JSON", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error unmarshalling tags JSON", nil, envelope.GeneralError)
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

	if err := app.conversation.UpsertConversationTags(uuid, tagNames); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope("Tags added successfully")
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

// sendCSATSurvey sends a CSAT survey if enabled on the inbox.
func sendCSATSurvey(app *App, conversation cmodels.Conversation, user umodels.User) error {
	inbox, err := app.inbox.GetDBRecord(conversation.InboxID)
	if err != nil {
		return err
	}

	if !inbox.CSATEnabled {
		return nil
	}

	csatR, err := app.csat.Create(conversation.ID, conversation.AssignedUserID.Int)
	if err != nil && err != csat.ErrCSATAlreadyExists {
		return err
	}

	csatURL := fmt.Sprintf("%s/csat/%s", app.consts.AppBaseURL, csatR.UUID)
	messageContent := fmt.Sprintf("Please rate your experience with us: <a href=\"%s\">Rate now</a>", csatURL)
	meta := map[string]interface{}{
		"is_csat": true,
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		app.lo.Error("error marshalling meta JSON for csat message", "error", err)
		return err
	}
	return app.conversation.SendReply(nil, user.ID, conversation.UUID, messageContent, string(metaJSON))
}
