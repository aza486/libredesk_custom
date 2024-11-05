package main

import (
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	medModels "github.com/abhinavxd/artemis/internal/media/models"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type messageReq struct {
	Attachments []int  `json:"attachments"`
	Message     string `json:"message"`
	Private     bool   `json:"private"`
}

// handleGetMessages returns messages for a conversation.
func handleGetMessages(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		uuid        = r.RequestCtx.UserValue("uuid").(string)
		user        = r.RequestCtx.UserValue("user").(umodels.User)
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)

	// Check permission
	_, err := enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	messages, pageSize, err := app.conversation.GetConversationMessages(uuid, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	for i := range messages {
		total = messages[i].Total
		for j := range messages[i].Attachments {
			messages[i].Attachments[j].URL = app.media.GetURL(messages[i].Attachments[j].UUID)
		}
	}
	return r.SendEnvelope(envelope.PageResults{
		Total:      total,
		Results:    messages,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

// handleGetMessage fetches a single from DB using the uuid.
func handleGetMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		user  = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Check permission
	_, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	messages, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	for j := range messages.Attachments {
		messages.Attachments[j].URL = app.media.GetURL(messages.Attachments[j].UUID)
	}

	return r.SendEnvelope(messages)
}

// handleRetryMessage changes message status so it can be retried for sending.
func handleRetryMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		user  = r.RequestCtx.UserValue("user").(umodels.User)
	)

	// Check permission
	_, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	err = app.conversation.MarkMessageAsPending(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleSendMessage sends a message in a conversation.
func handleSendMessage(r *fastglue.Request) error {
	var (
		app    = r.Context.(*App)
		user   = r.RequestCtx.UserValue("user").(umodels.User)
		cuuid  = r.RequestCtx.UserValue("cuuid").(string)
		req    = messageReq{}
		medias = []medModels.Media{}
	)

	// Check permission
	_, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling media ids", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error decoding request", nil, "")
	}

	for _, id := range req.Attachments {
		media, err := app.media.Get(id)
		if err != nil {
			app.lo.Error("error fetching media", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error fetching media", nil, "")
		}
		medias = append(medias, media)
	}

	// Private note.
	if req.Private {
		if err := app.conversation.SendPrivateNote(medias, user.ID, cuuid, req.Message); err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(true)
	}

	// Reply.
	if err := app.conversation.SendReply(medias, user.ID, cuuid, req.Message); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}
