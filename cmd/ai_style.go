package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type styleRequest struct {
	Style        string `json:"style"`
	Instructions string `json:"instructions"`
	OriginalText string `json:"original_text"`
	CurrentText  string `json:"current_text"`
	NoteUUID     string `json:"note_uuid"`
}

func decodeStyle(body []byte) (styleRequest, error) {
	var req styleRequest
	if len(body) > 220000 || !utf8.Valid(body) {
		return req, io.ErrUnexpectedEOF
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&req); err != nil {
		return req, err
	}
	if d.Decode(new(any)) != io.EOF {
		return req, io.ErrUnexpectedEOF
	}
	return req, nil
}
func validatedNote(note cmodels.Message, cuuid string) (conversation.AIAnalysis, bool) {
	var meta struct {
		Analysis conversation.AIAnalysis `json:"ai_analysis"`
		Deleted  any                     `json:"deleted_at"`
	}
	err := json.Unmarshal(note.Meta, &meta)
	return meta.Analysis, err == nil && meta.Deleted == nil && note.ConversationUUID == cuuid && note.Private && note.Type == "outgoing" && meta.Analysis.Validate() == nil
}
func handleAIStyleAdjust(r *fastglue.Request) error {
	app := r.Context.(*App)
	if r.RequestCtx.UserValue("auth_method") != authMethodSession {
		return r.SendErrorEnvelope(403, "Employee session required", nil, envelope.PermissionError)
	}
	caller := r.RequestCtx.UserValue("user").(amodels.User)
	user, err := app.user.GetAgentCachedOrLoad(caller.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !user.Enabled || user.Type != "agent" {
		return r.SendErrorEnvelope(403, "Enabled employee required", nil, envelope.PermissionError)
	}
	cuuid := r.RequestCtx.UserValue("cuuid").(string)
	conv, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	id := uuid.NewString()
	fail := func(status int, code, message string) error {
		return r.SendJSON(status, map[string]any{"success": false, "request_id": id, "error": map[string]string{"code": code, "message": message}})
	}
	req, err := decodeStyle(r.RequestCtx.PostBody())
	payload := aiassist.StyleRequest{RequestID: id, AgentID: strconv.Itoa(user.ID), Style: req.Style, Instructions: req.Instructions, OriginalText: req.OriginalText, CurrentText: req.CurrentText}
	if err != nil || !payload.Validate() || conv.InboxChannel != "email" {
		return fail(400, "invalid_request", "Ungültige Stil-Anfrage.")
	}
	if _, err := uuid.Parse(req.NoteUUID); err != nil {
		return fail(400, "invalid_note", "Ungültige KI-Notiz.")
	}
	note, err := app.conversation.GetMessage(req.NoteUUID)
	if err != nil {
		return fail(400, "invalid_note", "KI-Notiz nicht verfügbar.")
	}
	analysis, ok := validatedNote(note, cuuid)
	if !ok || analysis.SuggestedMessage != req.OriginalText {
		return fail(400, "invalid_note", "KI-Notiz passt nicht zum Originaltext.")
	}
	original, err := app.conversation.GetMessage(analysis.OriginalMessageUUID)
	if err != nil || conversation.ValidateAIAnalysisOriginal(analysis, original, cuuid) != nil {
		return fail(400, "invalid_source", "Ursprüngliche Kundenmail nicht verfügbar.")
	}
	used, err := app.conversation.AIResponseUsed(cuuid, analysis.OriginalMessageUUID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if used {
		return fail(409, "already_used", "Für diesen Vorschlag wurde bereits eine Antwort eingereiht.")
	}
	if app.aiStyle == nil {
		return fail(503, "not_configured", "Stilanpassung ist noch nicht eingerichtet.")
	}
	ctx, cancel := context.WithTimeout(app.ctx, 45*time.Second)
	defer cancel()
	var result aiassist.StyleResponse
	status, err := app.aiStyle.Call(ctx, "X-Libredesk-Style-Token", id, payload, &result)
	if err != nil {
		code := "style_unavailable"
		if status == 422 {
			code = "output_rejected"
		}
		if status == 504 {
			code = "style_timeout"
		}
		return fail(status, code, "Stilanpassung fehlgeschlagen. Bitte erneut versuchen.")
	}
	if !result.Valid(id) {
		return fail(fasthttp.StatusBadGateway, "invalid_response", "Ungültige Antwort der Stilanpassung.")
	}
	return r.SendJSON(200, result)
}
