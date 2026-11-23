package main

import (
	"context"
	"errors"
	"time"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

func handleAITranscriptionConfig(r *fastglue.Request) error {
	return r.SendEnvelope(map[string]bool{"enabled": r.Context.(*App).aiTranscription != nil})
}

// Raw audio avoids multipart parsing/temp files on the receiving server. Normal
// authenticated session, CSRF and messages:write middleware protect this route.
func handleAITranscribe(r *fastglue.Request) error {
	app := r.Context.(*App)
	if r.RequestCtx.UserValue("auth_method") != authMethodSession {
		return r.SendErrorEnvelope(403, "Employee session required", nil, envelope.PermissionError)
	}
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	agent, err := app.user.GetAgent(auser.ID, "")
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !agent.Enabled || agent.Type != "agent" {
		return r.SendErrorEnvelope(403, "Transcription unavailable", nil, envelope.PermissionError)
	}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	data := r.RequestCtx.PostBody()
	if len(data) > app.aiTranscription.MaxAudioBytes() {
		return r.SendErrorEnvelope(413, "Audio too large", nil, envelope.InputError)
	}
	typ := string(r.RequestCtx.Request.Header.ContentType())
	if !app.aiTranscription.ValidAudio(data, typ) {
		return r.SendErrorEnvelope(400, "Invalid audio", nil, envelope.InputError)
	}
	if app.aiTranscription == nil {
		return r.SendErrorEnvelope(503, "Transcription unavailable", nil, envelope.GeneralError)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	text, status, err := app.aiTranscription.Transcribe(ctx, data, typ)
	if err != nil {
		message := "Transcription failed. Please try again."
		switch {
		case status == 413:
			message = "Audio too large"
		case status == 504:
			message = "Transcription timed out. Please try again."
		case status == 503:
			message = "Transcription temporarily unavailable. Please try again later."
		case errors.Is(err, aiassist.ErrInvalidResponse):
			message = "No usable transcript received. Please record again."
		}
		return r.SendErrorEnvelope(status, message, nil, envelope.GeneralError)
	}
	return r.SendEnvelope(map[string]string{"text": text})
}
