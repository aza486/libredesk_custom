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
		p                 = r.RequestCtx.PostArgs()
		app               = r.Context.(*App)
		msgContent        = p.Peek("message")
		private           = p.GetBool("private")
		attachmentsJSON   = p.Peek("attachments")
		attachmentdsUUIDs = []string{}
		userID            = r.RequestCtx.UserValue("user_id").(int)
		conversationUUID  = r.RequestCtx.UserValue("conversation_uuid").(string)
	)

	if err := json.Unmarshal(attachmentsJSON, &attachmentdsUUIDs); err != nil {
		app.lo.Error("error unmarshalling attachments uuids", "error", err)
		return r.SendErrorEnvelope(http.StatusInternalServerError, "error parsing attachments", nil, "")
	}

	var attachments = make(models.Attachments, 0, len(attachmentdsUUIDs))
	for _, attUUID := range attachmentdsUUIDs {
		attachments = append(attachments, models.Attachment{
			UUID: attUUID,
		})
	}

	var status = message.StatusPending
	if private {
		status = message.StatusSent
	}

	_, _, err := app.msgMgr.RecordMessage(
		mmodels.Message{
			ConversationUUID: conversationUUID,
			SenderID:         int64(userID),
			Type:             message.TypeOutgoing,
			SenderType:       "user",
			Status:           status,
			Content:          string(msgContent),
			ContentType:      message.ContentTypeHTML,
			Private:          private,
			Meta:             "{}",
			Attachments:      attachments,
		},
	)

	if err != nil {
		return r.SendErrorEnvelope(http.StatusInternalServerError, err.Error(), nil, "")
	}

	// Add this user as a participant to the conversation.
	app.conversationMgr.AddParticipant(userID, conversationUUID)

	return r.SendEnvelope("Message sent")
}
