package main

import (
	"context"
	"errors"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
	"time"
)

func handleGetReplySources(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	if _, err = enforceConversationAccess(app, cuuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	page, size := getPagination(r)
	rows, total, err := app.conversation.ReplySources(cuuid, size, (page-1)*size)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]any{"results": rows, "total": total, "has_more": page*size < total})
}
func handleAssignReplySource(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	if _, err = enforceConversationAccess(app, cuuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	var req struct {
		Source string `json:"reply_to_message_uuid"`
	}
	if r.Decode(&req, "json") != nil {
		return r.SendErrorEnvelope(400, "Invalid source request", nil, envelope.InputError)
	}
	if _, err = uuid.Parse(req.Source); err != nil {
		return r.SendErrorEnvelope(400, "Invalid source UUID", nil, envelope.InputError)
	}
	err = app.conversation.AssignReplySource(app.ctx, cuuid, r.RequestCtx.UserValue("uuid").(string), req.Source)
	if errors.Is(err, conversation.ErrReplySource) {
		return r.SendErrorEnvelope(409, "Source missing, foreign, newer than the send or already bound", nil, envelope.InputError)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(map[string]any{"assigned": true})
}
func handleResolveMailCompletion(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	before, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if _, err = uuid.Parse(r.RequestCtx.UserValue("uuid").(string)); err != nil {
		return r.SendErrorEnvelope(400, "Invalid outgoing UUID", nil, envelope.InputError)
	}
	var req struct {
		Expected string `json:"expected_incoming_version"`
	}
	if r.Decode(&req, "json") != nil || req.Expected == "" {
		return r.SendErrorEnvelope(400, "Expected incoming version required", nil, envelope.InputError)
	}
	ctx, cancel := context.WithTimeout(app.ctx, 8*time.Second)
	defer cancel()
	result, err := app.conversation.ResolveMailCompletion(ctx, cuuid, r.RequestCtx.UserValue("uuid").(string), req.Expected)
	if errors.Is(err, conversation.ErrCompletionConflict) {
		return r.SendJSON(409, map[string]any{"outcome": "superseded", "retryable": false, "replayed": result.Replayed})
	}
	if errors.Is(err, conversation.ErrReplySource) || errors.Is(err, conversation.ErrCompletionNotSent) {
		return r.SendJSON(409, map[string]any{"outcome": "invalid_completion", "retryable": false})
	}
	if err != nil {
		return r.SendJSON(503, map[string]any{"outcome": "temporary_failure", "retryable": true})
	}
	if !result.Replayed {
		// No automation/CSAT/mail side effects on this completion endpoint.
		// Status audit and SLA are best effort after the durable transition.
		if result.Changed {
			if e := app.sla.EvaluateConversationSLA(before.ID); e != nil {
				app.lo.Error("error evaluating completion SLA", "conversation_uuid", cuuid)
			}
			if e := app.conversation.RecordStatusChange("Resolved", cuuid, user); e != nil {
				app.lo.Error("error recording completion status activity", "conversation_uuid", cuuid)
			}
			app.webhook.TriggerEvent(wmodels.EventConversationStatusChanged, map[string]any{"conversation_uuid": cuuid, "new_status": "Resolved", "previous_status": result.PreviousStatus, "snooze_until": "", "actor_id": user.ID, "outgoing_message_uuid": r.RequestCtx.UserValue("uuid")})
		}
		// Broadcast current persisted status, never write an unguarded status afterwards.
		if fresh, e := app.conversation.GetConversation(0, cuuid, ""); e == nil {
			app.conversation.BroadcastConversationUpdate(cuuid, map[string]any{"status": fresh.Status.String})
		}
	}
	return r.SendJSON(200, map[string]any{"outcome": result.Outcome, "replayed": result.Replayed, "retryable": false})
}
