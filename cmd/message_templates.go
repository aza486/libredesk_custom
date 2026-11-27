package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/messagetemplate"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/google/uuid"
	"github.com/zerodha/fastglue"
)

// Every operation resolves the real conversation and fresh ACL. A caller cannot
// grant global access by sending a private inbox ID or cached role metadata.
func messageTemplateAccess(r *fastglue.Request) (messagetemplate.Access, error) {
	app := r.Context.(*App)
	actor := r.RequestCtx.UserValue("user").(amodels.User)
	a := messagetemplate.Access{UserID: actor.ID, ConversationUUID: string(r.RequestCtx.QueryArgs().Peek("conversation_uuid"))}
	r.RequestCtx.Response.Header.Set("Cache-Control", "no-store")
	if a.ConversationUUID != "" {
		if _, err := uuid.Parse(a.ConversationUUID); err != nil {
			return a, envelope.NewError(envelope.InputError, "Ungültige Konversation.", nil)
		}
		if _, err := enforceConversationAccess(app, a.ConversationUUID, umodels.User{ID: actor.ID}); err != nil {
			return a, err
		}
	}
	return a, nil
}
func decodeMessageTemplate(r *fastglue.Request, out any) error {
	if len(r.RequestCtx.PostBody()) > messagetemplate.MaxContentBytes+8192 {
		return envelope.NewError(envelope.InputError, "Die Anfrage ist zu groß.", nil)
	}
	d := json.NewDecoder(bytes.NewReader(r.RequestCtx.PostBody()))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return envelope.NewError(envelope.InputError, "Ungültige Vorlagendaten. Eigentümer und Systemfelder können nicht gesetzt werden.", nil)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return envelope.NewError(envelope.InputError, "Ungültige Vorlagendaten.", nil)
	}
	return nil
}
func handleListMessageTemplates(r *fastglue.Request) error {
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result, err := r.Context.(*App).messageTemplates.List(a)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(result)
}
func handleGetMessageTemplate(r *fastglue.Request) error {
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	result, err := r.Context.(*App).messageTemplates.Get(a, r.RequestCtx.UserValue("uuid").(string))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(result)
}
func handleCreateMessageTemplate(r *fastglue.Request) error {
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var in messagetemplate.Input
	if err := decodeMessageTemplate(r, &in); err != nil {
		return sendErrorEnvelope(r, err)
	}
	result, err := r.Context.(*App).messageTemplates.Create(a, in)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(result)
}
func handleUpdateMessageTemplate(r *fastglue.Request) error {
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	var in messagetemplate.Input
	if err := decodeMessageTemplate(r, &in); err != nil {
		return sendErrorEnvelope(r, err)
	}
	result, err := r.Context.(*App).messageTemplates.Update(a, r.RequestCtx.UserValue("uuid").(string), in)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(result)
}
func handleDeleteMessageTemplate(r *fastglue.Request) error {
	a, err := messageTemplateAccess(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	revision, err := strconv.ParseInt(string(r.RequestCtx.QueryArgs().Peek("revision")), 10, 64)
	if err != nil || revision < 1 {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, "Die Vorlagenversion fehlt.", nil))
	}
	if err := r.Context.(*App).messageTemplates.Delete(a, r.RequestCtx.UserValue("uuid").(string), revision); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
