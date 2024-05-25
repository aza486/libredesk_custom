package main

import (
	"net/http"

	"github.com/zerodha/fastglue"
)

func handleGetConversations(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
	)
	conversations, err := app.conversations.GetConversations()
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope(conversations)
}

func handleGetConversation(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	conversation, err := app.conversations.GetConversation(uuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope(conversation)
}

func handleUpdateAssignee(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		p            = r.RequestCtx.PostArgs()
		uuid         = r.RequestCtx.UserValue("uuid").(string)
		assigneeType = r.RequestCtx.UserValue("assignee_type").(string)
		assigneeUUID = p.Peek("assignee_uuid")
	)
	if err := app.conversations.UpdateAssignee(uuid, assigneeUUID, assigneeType); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope("ok")
}

func handleUpdatePriority(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		p        = r.RequestCtx.PostArgs()
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		priority = p.Peek("priority")
	)
	if err := app.conversations.UpdatePriority(uuid, priority); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope("ok")
}

func handleUpdateStatus(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		p      = r.RequestCtx.PostArgs()
		uuid   = r.RequestCtx.UserValue("uuid").(string)
		status = p.Peek("status")
	)
	if err := app.conversations.UpdateStatus(uuid, status); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, "Something went wrong, try again later.", nil, "")
	}

	return r.SendEnvelope("ok")
}
