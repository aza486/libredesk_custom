package main

import (
	"encoding/json"
	"net/http"

	"github.com/abhinavxd/artemis/internal/attachment/models"
	"github.com/abhinavxd/artemis/internal/message"
	mmodels "github.com/abhinavxd/artemis/internal/message/models"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/zerodha/fastglue"
)

func handleGetMessages(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	msgs, err := app.messageManager.GetConversationMessages(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Generate URLs for all attachments.
	for i := range msgs {
		for j := range msgs[i].Attachments {
			msgs[i].Attachments[j].URL = app.attachmentManager.Store.GetURL(msgs[i].Attachments[j].UUID)
		}
	}
	return r.SendEnvelope(msgs)
}

func handleGetMessage(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("uuid").(string)
	)
	msgs, err := app.messageManager.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Generate URLs for each of the attachments.
	for i := range msgs {
		for j := range msgs[i].Attachments {
			msgs[i].Attachments[j].URL = app.attachmentManager.Store.GetURL(msgs[i].Attachments[j].UUID)
		}
	}

	return r.SendEnvelope(msgs)
}

func handleRetryMessage(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("message_uuid").(string)
	)
	err := app.messageManager.RetryMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

func handleSendMessage(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		user             = r.RequestCtx.UserValue("user").(umodels.User)
		p                = r.RequestCtx.PostArgs()
		content          = p.Peek("message")
		private          = p.GetBool("private")
		uuid             = r.RequestCtx.UserValue("uuid").(string)
		attachmentsUUIDs = []string{}
		attachmentsJSON  = p.Peek("attachments")
		attachments      = make(models.Attachments, 0, len(attachmentsUUIDs))
	)

	if err := json.Unmarshal(attachmentsJSON, &attachmentsUUIDs); err != nil {
		app.lo.Error("error unmarshalling attachments uuids", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "error parsing attachments", nil, "")
	}
	for _, attUUID := range attachmentsUUIDs {
		attachments = append(attachments, models.Attachment{
			UUID: attUUID,
		})
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
		Meta:             "{}",
		Attachments:      attachments,
	}

	if err := app.messageManager.RecordMessage(&msg); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.conversationManager.AddParticipant(user.ID, uuid)

	// Update conversation meta with the last message details.
	trimmedMessage := app.messageManager.TrimMsg(msg.Content)
	app.conversationManager.UpdateLastMessage(0, uuid, trimmedMessage, msg.CreatedAt)

	// Send WS update.
	app.messageManager.BroadcastNewConversationMessage(msg, trimmedMessage)

	return r.SendEnvelope("Message sent")
}
