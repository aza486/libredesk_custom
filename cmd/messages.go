package main

import (
	"encoding/json"
	"net/http"

	"github.com/abhinavxd/artemis/internal/attachment/models"
	"github.com/abhinavxd/artemis/internal/message"
	mmodels "github.com/abhinavxd/artemis/internal/message/models"
	"github.com/zerodha/fastglue"
)

func handleGetMessages(r *fastglue.Request) error {
	var (
		app  = r.Context.(*App)
		uuid = r.RequestCtx.UserValue("conversation_uuid").(string)
	)
	msgs, err := app.msgMgr.GetConvMessages(uuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	// Generate URLs for each of the attachments.
	for i := range msgs {
		for j := range msgs[i].Attachments {
			msgs[i].Attachments[j].URL = app.attachmentMgr.Store.GetURL(msgs[i].Attachments[j].UUID)
		}
	}

	return r.SendEnvelope(msgs)
}

func handleGetMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		muuid = r.RequestCtx.UserValue("message_uuid").(string)
	)
	msgs, err := app.msgMgr.GetMessage(muuid)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	// Generate URLs for each of the attachments.
	for i := range msgs {
		for j := range msgs[i].Attachments {
			msgs[i].Attachments[j].URL = app.attachmentMgr.Store.GetURL(msgs[i].Attachments[j].UUID)
		}
	}

	return r.SendEnvelope(msgs)
}

func handleRetryMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		muuid = r.RequestCtx.UserValue("message_uuid").(string)
	)
	// Change status to pending so this message can be retried again.
	err := app.msgMgr.UpdateMessageStatus(muuid, message.StatusPending)
	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}
	return r.SendEnvelope("ok")
}

func handleSendMessage(r *fastglue.Request) error {
	var (
		app              = r.Context.(*App)
		userID           = r.RequestCtx.UserValue("user_id").(int)
		p                = r.RequestCtx.PostArgs()
		msgContent       = p.Peek("message")
		private          = p.GetBool("private")
		conversationUUID = r.RequestCtx.UserValue("conversation_uuid").(string)
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
		ConversationUUID: conversationUUID,
		SenderID:         userID,
		Type:             message.TypeOutgoing,
		SenderType:       message.SenderTypeUser,
		Status:           message.StatusPending,
		Content:          string(msgContent),
		ContentType:      message.ContentTypeHTML,
		Private:          private,
		Meta:             "{}",
		Attachments:      attachments,
	}

	if err := app.msgMgr.RecordMessage(&msg); err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	app.conversationMgr.AddParticipant(userID, conversationUUID)

	// Update conversation meta with the last message details.
	trimmedMessage := app.msgMgr.TrimMsg(msg.Content)
	app.conversationMgr.UpdateLastMessage(0, conversationUUID, trimmedMessage, msg.CreatedAt)

	// Send WS update.
	app.msgMgr.BroadcastNewConversationMessage(msg, trimmedMessage)

	return r.SendEnvelope("Message sent")
}
