package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

func decodeManualReason(body []byte) (string, error) {
	var req struct {
		Reason *string `json:"feedback_reason"`
	}
	if len(body) > 20000 || !utf8.Valid(body) || !bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		return "", conversation.ErrManualReasonInput
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		return "", conversation.ErrManualReasonInput
	}
	if d.Decode(new(any)) != io.EOF {
		return "", conversation.ErrManualReasonInput
	}
	if req.Reason == nil {
		return "", nil
	}
	return conversation.NormalizeManualReason(*req.Reason)
}

func handleManualReviewReason(r *fastglue.Request) error {
	app := r.Context.(*App)
	caller, ok := r.RequestCtx.UserValue("user").(amodels.User)
	if !ok || r.RequestCtx.UserValue("auth_method") != authMethodSession {
		return r.SendErrorEnvelope(403, "Mitarbeiter-Sitzung erforderlich.", nil, envelope.PermissionError)
	}
	user, err := app.user.GetAgentCachedOrLoad(caller.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !user.Enabled || user.Type != "agent" || user.IsSystemUser() {
		return r.SendErrorEnvelope(403, "Aktiver Mitarbeiter erforderlich.", nil, envelope.PermissionError)
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	if _, err = enforceConversationAccess(app, cuuid, user); err != nil {
		return sendErrorEnvelope(r, err)
	}
	var reason *string
	switch string(r.RequestCtx.Method()) {
	case "GET":
	case "DELETE":
		empty := ""
		reason = &empty
	case "PUT":
		value, e := decodeManualReason(r.RequestCtx.PostBody())
		if e != nil {
			return r.SendErrorEnvelope(400, "Bitte höchstens 2500 Zeichen als Feedbackgrund eingeben.", nil, envelope.InputError)
		}
		reason = &value
	default:
		return r.SendErrorEnvelope(405, "Methode nicht erlaubt.", nil, envelope.InputError)
	}
	result, err := app.conversation.ManualReviewFeedback(app.ctx, cuuid, r.RequestCtx.UserValue("uuid").(string), user.ID, reason)
	switch {
	case err == nil:
		return r.SendEnvelope(result)
	case errors.Is(err, conversation.ErrManualReviewNote):
		return r.SendErrorEnvelope(404, "Die manuelle Prüfnotiz ist nicht verfügbar.", nil, envelope.InputError)
	case errors.Is(err, conversation.ErrManualReasonLocked):
		return r.SendErrorEnvelope(409, "Der Feedbackgrund kann nicht mehr geändert werden: Die Anfrage wurde bereits beantwortet oder eine neuere Kundenmail ist eingegangen.", nil, envelope.InputError)
	case errors.Is(err, conversation.ErrManualReasonAgent):
		return r.SendErrorEnvelope(403, "Aktiver Mitarbeiter erforderlich.", nil, envelope.PermissionError)
	case errors.Is(err, conversation.ErrManualReasonInput):
		return r.SendErrorEnvelope(400, "Ungültiger Feedbackgrund.", nil, envelope.InputError)
	default:
		return sendErrorEnvelope(r, err)
	}
}
