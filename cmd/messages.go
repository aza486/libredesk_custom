package main

import (
	"strconv"

	amodels "github.com/abhinavxd/artemis/internal/auth/models"
	"github.com/abhinavxd/artemis/internal/automation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	medModels "github.com/abhinavxd/artemis/internal/media/models"
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
		auser       = r.RequestCtx.UserValue("user").(amodels.User)
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
		total       = 0
	)

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, uuid, user)
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
		messages[i].HideCSAT()
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
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	message, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	message.HideCSAT()

	for j := range message.Attachments {
		message.Attachments[j].URL = app.media.GetURL(message.Attachments[j].UUID)
	}

	return r.SendEnvelope(message)
}

// handleRetryMessage changes message status so it can be retried for sending.
func handleRetryMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		auser  = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
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
		app   = r.Context.(*App)
		auser  = r.RequestCtx.UserValue("user").(amodels.User)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		req   = messageReq{}
		media = []medModels.Media{}
	)

	user, err := app.user.Get(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling media ids", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error decoding request", nil, "")
	}

	for _, id := range req.Attachments {
		m, err := app.media.Get(id)
		if err != nil {
			app.lo.Error("error fetching media", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error fetching media", nil, "")
		}
		media = append(media, m)
	}

	// Private note.
	if req.Private {
		if err := app.conversation.SendPrivateNote(media, user.ID, cuuid, req.Message); err != nil {
			return sendErrorEnvelope(r, err)
		}
		return r.SendEnvelope(true)
	}

	// Reply.
	if err := app.conversation.SendReply(media, user.ID, cuuid, req.Message, ""); err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Evaluate automation rules.
	app.automation.EvaluateConversationUpdateRules(cuuid, models.EventConversationMessageOutgoing)

	return r.SendEnvelope(true)
}