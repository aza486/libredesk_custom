package main

import (
	"encoding/json"
	"net/http"

	"github.com/abhinavxd/artemis/internal/message"
	"github.com/zerodha/fastglue"
)

func handleGetConversations(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)

	c, err := app.conversationMgr.GetConversations()

	// Strip html from the last message and truncate.
	for i := range c {
		c[i].LastMessage = app.msgMgr.TrimMsg(c[i].LastMessage)
	}
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	return r.SendEnvelope(c)
}

func handleGetConversation(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("conversation_uuid").(string)
	)
	c, err := app.conversationMgr.Get(uuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(c)
}

func handleUpdateAssigneeLastSeen(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("conversation_uuid").(string)
	)
	err := app.conversationMgr.UpdateAssigneeLastSeen(uuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope("ok")
}

func handleGetConversationParticipants(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("conversation_uuid").(string)
	)
	p, err := app.conversationMgr.GetParticipants(uuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope(p)
}

func handleUpdateAssignee(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		p            = r.RequestCtx.PostArgs()
		assigneeUUID = p.Peek("assignee_uuid")
		uuid         = r.RequestCtx.UserValue("conversation_uuid").(string)
		assigneeType = r.RequestCtx.UserValue("assignee_type").(string)
		userUUID     = r.RequestCtx.UserValue("user_uuid").(string)
		userID       = r.RequestCtx.UserValue("user_id").(int)
	)

	if err := app.conversationMgr.UpdateAssignee(uuid, assigneeUUID, assigneeType); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	// Insert the activity message.
	actorAgent, err := app.userMgr.GetUser(userUUID)
	if err != nil {
		app.lo.Warn("fetching agent details from uuid", "uuid", userUUID)
		return r.SendEnvelope("ok")
	}

	if assigneeType == "agent" {
		assigneeAgent, err := app.userMgr.GetUser(userUUID)
		if err != nil {
			app.lo.Warn("fetching agent details from uuid", "uuid", string(assigneeUUID))
			return r.SendEnvelope("ok")
		}
		activityType := message.ActivityAssignedAgentChange
		if string(assigneeUUID) == userUUID {
			activityType = message.ActivitySelfAssign
		}
		app.msgMgr.RecordActivity(activityType, assigneeAgent.FullName(), uuid, actorAgent.FullName(), userID)

	} else if assigneeType == "team" {
		team, err := app.teamMgr.GetTeam(string(assigneeUUID))
		if err != nil {
			app.lo.Warn("fetching team details from uuid", "uuid", string(assigneeUUID))
			return r.SendEnvelope("ok")
		}
		app.msgMgr.RecordActivity(message.ActivityAssignedTeamChange, team.Name, uuid, actorAgent.FullName(), userID)
	}

	return r.SendEnvelope("ok")
}

func handleUpdatePriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		priority = p.Peek("priority")
		uuid     = r.RequestCtx.UserValue("conversation_uuid").(string)
		userUUID = r.RequestCtx.UserValue("user_uuid").(string)
		userID   = r.RequestCtx.UserValue("user_id").(int)
	)
	if err := app.conversationMgr.UpdatePriority(uuid, priority); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	actorAgent, err := app.userMgr.GetUser(userUUID)
	if err != nil {
		app.lo.Warn("fetching agent details from uuid", "uuid", string(userUUID))
		return r.SendEnvelope("ok")
	}

	app.msgMgr.RecordActivity(message.ActivityPriorityChange, string(priority), uuid, actorAgent.FullName(), userID)

	return r.SendEnvelope("ok")
}

func handleUpdateStatus(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		status   = p.Peek("status")
		uuid     = r.RequestCtx.UserValue("conversation_uuid").(string)
		userUUID = r.RequestCtx.UserValue("user_uuid").(string)
		userID   = r.RequestCtx.UserValue("user_id").(int)
	)
	if err := app.conversationMgr.UpdateStatus(uuid, status); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	actorAgent, err := app.userMgr.GetUser(userUUID)
	if err != nil {
		app.lo.Warn("fetching agent details from uuid", "uuid", string(userUUID))
		return r.SendEnvelope("ok")
	}

	app.msgMgr.RecordActivity(message.ActivityStatusChange, string(status), uuid, actorAgent.FullName(), userID)

	return r.SendEnvelope("ok")
}

func handlAddConversationTags(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		tagIDs  = []int{}
		p       = r.RequestCtx.PostArgs()
		tagJSON = p.Peek("tag_ids")
		uuid    = r.RequestCtx.UserValue("conversation_uuid").(string)
	)
	err := json.Unmarshal(tagJSON, &tagIDs)
	if err != nil {
		app.lo.Error("unmarshalling tag ids", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "error adding tags", nil, "")
	}

	if err := app.conversationTagsMgr.AddTags(uuid, tagIDs); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope("ok")
}
