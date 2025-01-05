package main

import (
	"strconv"

	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleShowCSAT renders the CSAT page for a given csat.
func handleShowCSAT(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)

	if uuid == "" {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"error_message": "Page not found",
		})
	}

	csat, err := app.csat.Get(uuid)
	if err != nil {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"error_message": "CSAT not found",
		})
	}

	if csat.ResponseTimestamp.Valid {
		return app.tmpl.RenderWebPage(r.RequestCtx, "info", map[string]interface{}{
			"message": "You've already submitted your feedback",
		})
	}

	conversation, err := app.conversation.GetConversation(csat.ConversationID, "")
	if err != nil {
		return app.tmpl.RenderWebPage(r.RequestCtx, "error", map[string]interface{}{
			"error_message": "Conversation not found",
		})
	}

	return app.tmpl.RenderWebPage(r.RequestCtx, "csat", map[string]interface{}{
		"csat": map[string]interface{}{
			"uuid": csat.UUID,
		},
		"conversation": map[string]interface{}{
			"subject": conversation.Subject.String,
		},
	})
}

// handleUpdateCSATResponse updates the CSAT response for a given csat.
func handleUpdateCSATResponse(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		uuid     = r.RequestCtx.UserValue("uuid").(string)
		rating   = r.RequestCtx.FormValue("rating")
		feedback = string(r.RequestCtx.FormValue("feedback"))
	)

	ratingI, err := strconv.Atoi(string(rating))
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid `rating`", nil, envelope.InputError)
	}

	if ratingI < 1 || ratingI > 5 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "`rating` should be between 1 and 5", nil, envelope.InputError)
	}

	if uuid == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Empty `uuid`", nil, envelope.InputError)
	}

	if err := app.csat.UpdateResponse(uuid, ratingI, feedback); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope("CSAT response updated")
}
