package main

import (
	"encoding/json"
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetConversations(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		c, err = app.conversationMgr.GetConversations()
	)

	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	// Strip html from the last message and truncate.
	for i := range c {
		c[i].LastMessage = app.msgMgr.TrimMsg(c[i].LastMessage)
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
		convUUID     = r.RequestCtx.UserValue("conversation_uuid").(string)
		assigneeType = r.RequestCtx.UserValue("assignee_type").(string)
		userUUID     = r.RequestCtx.UserValue("user_uuid").(string)
	)

	if err := app.conversationMgr.UpdateAssignee(convUUID, assigneeUUID, assigneeType); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	if assigneeType == "agent" {
		app.msgMgr.RecordAssigneeUserChange(string(assigneeUUID), convUUID, userUUID)
	}

	if assigneeType == "team" {
		app.msgMgr.RecordAssigneeTeamChange(string(assigneeUUID), convUUID, userUUID)
	}

	return r.SendEnvelope("ok")
}

func handleUpdatePriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		priority = p.Peek("priority")
		convUUID = r.RequestCtx.UserValue("conversation_uuid").(string)
		userUUID = r.RequestCtx.UserValue("user_uuid").(string)
	)
	if err := app.conversationMgr.UpdatePriority(convUUID, priority); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	app.msgMgr.RecordPriorityChange(string(priority), convUUID, userUUID)

	return r.SendEnvelope("ok")
}

func handleUpdateStatus(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		status   = p.Peek("status")
		convUUID = r.RequestCtx.UserValue("conversation_uuid").(string)
		userUUID = r.RequestCtx.UserValue("user_uuid").(string)
	)
	if err := app.conversationMgr.UpdateStatus(convUUID, status); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	app.msgMgr.RecordStatusChange(string(status), convUUID, userUUID)

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
