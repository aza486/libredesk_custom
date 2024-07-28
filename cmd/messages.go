package main

import (
	"encoding/json"
	"strconv"

	medModels "github.com/abhinavxd/artemis/internal/media/models"
	"github.com/abhinavxd/artemis/internal/message"
	mmodels "github.com/abhinavxd/artemis/internal/message/models"
	"github.com/abhinavxd/artemis/internal/stringutil"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func handleGetMessages(r *fastglue.Request) error {
	var (
		app         = r.Context.(*App)
		uuid        = r.RequestCtx.UserValue("uuid").(string)
		page, _     = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page")))
		pageSize, _ = strconv.Atoi(string(r.RequestCtx.QueryArgs().Peek("page_size")))
	)

	messages, err := app.message.GetConversationMessages(uuid, page, pageSize)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	for i := range messages {
		for j := range messages[i].Attachments {
			messages[i].Attachments[j].URL = app.media.Store.GetURL(messages[i].Attachments[j].Name)
		}
	}
	return r.SendEnvelope(messages)
}

func handleGetMessage(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	messages, err := app.message.Get(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	for j := range messages.Attachments {
		messages.Attachments[j].URL = app.media.Store.GetURL("")
	}

	return r.SendEnvelope(messages)
}

func handleRetryMessage(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("message_uuid").(string)
	)
	err := app.message.MarkAsPending(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleSendMessage(r *fastglue.Request) error {
	var (
		app        = r.Context.(*App)
		user       = r.RequestCtx.UserValue("user").(umodels.User)
		p          = r.RequestCtx.PostArgs()
		content    = p.Peek("message")
		private    = p.GetBool("private")
		uuid       = r.RequestCtx.UserValue("uuid").(string)
		mediaIDs   = []int{}
		mediasJSON = p.Peek("medias")
	)

	if err := json.Unmarshal(mediasJSON, &mediaIDs); err != nil {
		app.lo.Error("error unmarshalling media ids", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "error parsing attachments", nil, "")
	}

	var medias = []medModels.Media{}
	for _, id := range mediaIDs {
		media, err := app.media.Get(id)
		if err != nil {
			app.lo.Error("error fetching media", "error", err)
			return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, "Error sending message.", nil, "")
		}
		medias = append(medias, media)
	}

	msg := mmodels.Message{
		ConversationUUID: uuid,
		SenderID:         user.ID,
		Type:             message.TypeOutgoing,
		SenderType:       message.SenderTypeUser,
		Status:           message.StatusPending,
		Content:          string(content),
		ContentType:      message.ContentTypeHTML,
		Private:          private,
		Media:            medias,
	}

	if err := app.message.Insert(&msg); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.conversation.AddParticipant(user.ID, uuid)

	// Update conversation meta with the last message details.
	trimmedMessage := stringutil.Trim(msg.Content, 45)
	app.conversation.UpdateLastMessage(0, uuid, trimmedMessage, msg.CreatedAt)

	// Send WS update.
	app.message.BroadcastNewConversationMessage(msg, trimmedMessage)

	return r.SendEnvelope(true)
}
