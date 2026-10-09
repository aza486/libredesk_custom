package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"unicode/utf8"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type aiAnalysisNoteRequest struct {
	Message    string                   `json:"message"`
	Private    bool                     `json:"private"`
	SenderType string                   `json:"sender_type"`
	AIAnalysis *conversation.AIAnalysis `json:"ai_analysis"`
}

func allowedAIAnalysisCaller(method string, id int, allowed []int) bool {
	return method == authMethodAPIKey && id > 0 && slices.Contains(allowed, id)
}

func decodeAIAnalysisRequest(body []byte) (aiAnalysisNoteRequest, error) {
	var req aiAnalysisNoteRequest
	if !utf8.Valid(body) {
		return req, errors.New("invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return req, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return req, errors.New("expected one JSON object")
	}
	if !req.Private || req.SenderType != "agent" || strings.TrimSpace(req.Message) == "" || req.AIAnalysis == nil {
		return req, errors.New("expected private agent note with ai_analysis")
	}
	return req, req.AIAnalysis.Validate()
}

func handleCreateAIAnalysisNote(r *fastglue.Request) error {
	app := r.Context.(*App)
	caller, ok := r.RequestCtx.UserValue("user").(amodels.User)
	method, _ := r.RequestCtx.UserValue("auth_method").(string)
	if !ok || !allowedAIAnalysisCaller(method, caller.ID, app.aiNoteAgentIDs) {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "AI analysis integration account required", nil, envelope.PermissionError)
	}
	user, err := app.user.GetAgentCachedOrLoad(caller.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !user.Enabled || user.Type != "agent" {
		return r.SendErrorEnvelope(fasthttp.StatusForbidden, "Enabled integration agent required", nil, envelope.PermissionError)
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	conv, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if conv.InboxChannel != "email" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "AI analysis notes require an email conversation", nil, envelope.InputError)
	}
	req, err := decodeAIAnalysisRequest(r.RequestCtx.PostBody())
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid AI analysis note request", nil, envelope.InputError)
	}
	message, err := app.conversation.SendAIAnalysisNote(user.ID, cuuid, req.Message, *req.AIAnalysis)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(message)
}
