package main

import (
	"encoding/json"
	"strconv"

	"github.com/abhinavxd/artemis/internal/conversation"
	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	medModels "github.com/abhinavxd/artemis/internal/media/models"
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

	messages, err := app.conversation.GetConversationMessages(uuid, page, pageSize)
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
	messages, err := app.conversation.GetMessage(uuid)
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
	err := app.conversation.MarkMessageAsPending(uuid)
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

	msg := cmodels.Message{
		ConversationUUID: uuid,
		SenderID:         user.ID,
		Type:             conversation.MessageOutgoing,
		SenderType:       conversation.SenderTypeUser,
		Status:           conversation.MessageStatusPending,
		Content:          string(content),
		ContentType:      conversation.ContentTypeHTML,
		Private:          private,
		Media:            medias,
	}

	if err := app.conversation.InsertMessage(&msg); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.conversation.AddConversationParticipant(user.ID, uuid)

	// Update conversation meta with the last message details.
	trimmedMessage := stringutil.Trim(msg.Content, 45)
	app.conversation.UpdateConversationLastMessage(0, uuid, trimmedMessage, msg.CreatedAt)
	return r.SendEnvelope(true)
}
