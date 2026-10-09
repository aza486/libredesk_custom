package main

import (
	"encoding/json"
	"errors"

	"github.com/abhinavxd/libredesk/internal/aifeedback"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleAIFeedback never queues or retries a customer message.
func handleAIFeedback(r *fastglue.Request) error {
	app := r.Context.(*App)
	if r.RequestCtx.UserValue("auth_method") != authMethodSession {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Employee session required", nil, envelope.PermissionError)
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	conversation := r.RequestCtx.UserValue("cuuid").(string)
	if _, err := enforceConversationAccess(app, conversation, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if app.aiFeedback == nil {
		return r.SendErrorEnvelope(fasthttp.StatusServiceUnavailable, "AI feedback receiver is not configured; retry feedback only", nil, envelope.GeneralError)
	}
	var req struct {
		FeedbackReason string `json:"feedback_reason"`
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid feedback request", nil, envelope.InputError)
	}
	reply, err := app.conversation.GetMessage(r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Reject foreign replies before reading their original-message association.
	if reply.ConversationUUID != conversation || reply.SenderID != user.ID {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Feedback requires your own reply in this conversation", nil, envelope.PermissionError)
	}
	var meta map[string]any
	if json.Unmarshal(reply.Meta, &meta) != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing original message association", nil, envelope.InputError)
	}
	originalUUID, ok := meta[aifeedback.OriginalKey].(string)
	if !ok || originalUUID == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Missing original message association", nil, envelope.InputError)
	}
	original, err := app.conversation.GetMessage(originalUUID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	submission, err := aifeedback.Build(reply, original, conversation, user.ID, req.FeedbackReason)
	if err != nil {
		status := fasthttp.StatusBadRequest
		if errors.Is(err, aifeedback.ErrNotSent) {
			status = fasthttp.StatusConflict
		}
		return r.SendErrorEnvelope(status, err.Error(), nil, envelope.InputError)
	}
	if err := app.aiFeedback.Submit(app.ctx, submission); err != nil {
		app.lo.Error("AI feedback commit not confirmed", "message_uuid", reply.UUID, "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusBadGateway, "Feedback commit not confirmed; retry feedback only with the same reply UUID", nil, envelope.GeneralError)
	}
	return r.SendEnvelope(map[string]any{"id": submission.ID, "committed": true})
}
