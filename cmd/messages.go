package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/google/uuid"

	"github.com/abhinavxd/libredesk/internal/aifeedback"
	inboxmodels "github.com/abhinavxd/libredesk/internal/inbox"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	authzModels "github.com/abhinavxd/libredesk/internal/authz/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type messageReq struct {
	ReplyToMessageUUID            string                 `json:"reply_to_message_uuid"`
	AIResponse                    *responseProvenance    `json:"ai_response,omitempty"`
	AIFeedbackOriginalMessageUUID string                 `json:"ai_feedback_original_message_uuid"`
	Attachments                   []int                  `json:"attachments"`
	Message                       string                 `json:"message"`
	Private                       bool                   `json:"private"`
	To                            []string               `json:"to"`
	CC                            []string               `json:"cc"`
	BCC                           []string               `json:"bcc"`
	SenderType                    string                 `json:"sender_type"`
	Mentions                      []cmodels.MentionInput `json:"mentions"`
	EchoID                        string                 `json:"echo_id"`
	TemplateID                    int                    `json:"template_id"`
}

// handleGetMessages returns messages for a conversation.
func handleGetMessages(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		uuid    = r.RequestCtx.UserValue("uuid").(string)
		auser   = r.RequestCtx.UserValue("user").(amodels.User)
		total   = 0
		private *bool
	)
	page, pageSize := getPagination(r)

	// Parse optional private filter (null = no filter)
	if r.RequestCtx.QueryArgs().Has("private") {
		p := r.RequestCtx.QueryArgs().GetBool("private")
		private = &p
	}

	// Parse repeated type params: ?type=incoming&type=outgoing
	var msgTypes []string
	for _, v := range r.RequestCtx.QueryArgs().PeekMulti("type") {
		msgTypes = append(msgTypes, string(v))
	}

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, uuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	messages, pageSize, err := app.conversation.GetConversationMessages(uuid, page, pageSize, private, msgTypes)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	rootURL, _ := app.setting.GetAppRootURL()
	for i := range messages {
		total = messages[i].Total
		app.conversation.SignAttachmentURLs(messages[i].Attachments)
		resolveQuotedCIDs(app, &messages[i])
		resolveAttachmentCIDs(&messages[i], rootURL)
	}

	// Process CSAT status for all messages (will only affect CSAT messages)
	app.conversation.ProcessCSATStatus(messages)

	// Strip CSAT UUID from agent sessions to prevent self-rating.
	if r.RequestCtx.UserValue("auth_method") != authMethodAPIKey {
		for i := range messages {
			if messages[i].HasCSAT() {
				messages[i].StripCSATUUID()
			}
		}
	}

	return r.SendEnvelope(envelope.PageResults{
		Total:      total,
		Results:    messages,
		Page:       page,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
	})
}

// handleGetMessage fetches a single from DB using the uuid.
func handleGetMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)
	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	message, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if message.ConversationUUID != cuuid {
		return sendErrorEnvelope(r, envelope.NewError(envelope.PermissionError, "Permission denied", nil))
	}

	// Process CSAT status for the message (will only affect CSAT messages)
	messages := []cmodels.Message{message}
	app.conversation.ProcessCSATStatus(messages)
	message = messages[0]

	// Strip CSAT UUID from agent sessions to prevent self-rating.
	if r.RequestCtx.UserValue("auth_method") != authMethodAPIKey && message.HasCSAT() {
		message.StripCSATUUID()
	}

	rootURL, _ := app.setting.GetAppRootURL()
	app.conversation.SignAttachmentURLs(message.Attachments)
	resolveQuotedCIDs(app, &message)
	resolveAttachmentCIDs(&message, rootURL)

	return r.SendEnvelope(message)
}

// handleRetryMessage changes message status to `pending`, so it's enqueued for sending.
func handleRetryMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check permission
	_, err = enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Only outgoing agent messages that have failed can be retried.
	msg, err := app.conversation.GetMessage(uuid)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if msg.SenderType != cmodels.SenderTypeAgent || msg.Status != cmodels.MessageStatusFailed || msg.SenderID != user.ID || msg.ConversationUUID != cuuid {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	if err = app.conversation.MarkMessageAsPending(uuid); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleDeletePrivateNote deletes a private note and its mentions.
func handleDeletePrivateNote(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		uuid  = r.RequestCtx.UserValue("uuid").(string)
	)

	app.lo.Info(
		"DELETE NOTE",
		"user_type", fmt.Sprintf("%T", r.RequestCtx.UserValue("user")),
		"uuid", r.RequestCtx.UserValue("uuid"),
	)

	if err := app.conversation.DeletePrivateNote(
		uuid,
		auser.ID,
	); err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(true)
}

// handleSendMessage sends a message in a conversation.
func handleSendMessage(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		cuuid = r.RequestCtx.UserValue("cuuid").(string)
		req   = messageReq{}
	)

	user, err := app.user.GetAgentCachedOrLoad(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check access to conversation.
	conv, err := enforceConversationAccess(app, cuuid, user)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	if err := r.Decode(&req, "json"); err != nil {
		app.lo.Error("error unmarshalling message request", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	// Make sure the inbox is enabled.
	inbox, err := app.inbox.GetDBRecord(conv.InboxID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if !inbox.Enabled {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("status.disabledInbox"), nil, envelope.InputError)
	}

	if req.SenderType != umodels.UserTypeAgent && req.SenderType != umodels.UserTypeContact {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}

	// Contacts cannot send private messages
	if req.SenderType == umodels.UserTypeContact && req.Private {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Check if user has permission to send messages as contact
	if req.SenderType == umodels.UserTypeContact {
		parts := strings.Split(authzModels.PermMessagesWriteAsContact, ":")
		if len(parts) != 2 {
			app.lo.Error("error parsing permission string", "permission", authzModels.PermMessagesWriteAsContact)
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
		}
		ok, err := app.authz.Enforce(user, parts[0], parts[1])
		if err != nil {
			app.lo.Error("error checking permission", "error", err)
			return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
		}
		if !ok {
			return r.SendErrorEnvelope(fasthttp.StatusForbidden, app.i18n.T("status.deniedPermission"), nil, envelope.PermissionError)
		}
	}

	if req.AIFeedbackOriginalMessageUUID != "" {
		if r.RequestCtx.UserValue("auth_method") != authMethodSession || req.Private || req.SenderType != umodels.UserTypeAgent || inbox.Channel != inboxmodels.ChannelEmail {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "AI feedback requires a public agent email reply from a session", nil, envelope.InputError)
		}
		original, err := app.conversation.GetMessage(req.AIFeedbackOriginalMessageUUID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if err := aifeedback.ValidateOriginal(original, cuuid); err != nil {
			return r.SendErrorEnvelope(fasthttp.StatusBadRequest, "Invalid original customer message", nil, envelope.InputError)
		}
	}

	if p := req.AIResponse; p != nil {
		if !validProvenance(*p, req.Private, req.SenderType, inbox.Channel, fmt.Sprint(r.RequestCtx.UserValue("auth_method")), req.AIFeedbackOriginalMessageUUID) {
			return r.SendErrorEnvelope(400, "Invalid response provenance", nil, envelope.InputError)
		}
		original, err := app.conversation.GetMessage(p.SourceUUID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if aifeedback.ValidateOriginal(original, cuuid) != nil {
			return r.SendErrorEnvelope(400, "Invalid provenance source", nil, envelope.InputError)
		}
		if p.Source != "human" {
			note, err := app.conversation.GetMessage(p.NoteUUID)
			if err != nil {
				return sendErrorEnvelope(r, err)
			}
			analysis, ok := validatedNote(note, cuuid)
			if !ok || analysis.OriginalMessageUUID != p.SourceUUID {
				return r.SendErrorEnvelope(400, "Invalid provenance note", nil, envelope.InputError)
			}
			used, err := app.conversation.AIResponseUsed(cuuid, p.SourceUUID)
			if err != nil {
				return sendErrorEnvelope(r, err)
			}
			if used {
				return r.SendErrorEnvelope(409, "AI suggestion already answered", nil, envelope.InputError)
			}
		}
	}
	if req.ReplyToMessageUUID != "" {
		if _, err := uuid.Parse(req.ReplyToMessageUUID); err != nil {
			return r.SendErrorEnvelope(400, "Invalid reply source UUID", nil, envelope.InputError)
		}
		if req.Private || req.SenderType != "agent" || inbox.Channel != "email" {
			return r.SendErrorEnvelope(400, "Reply source requires a public employee email", nil, envelope.InputError)
		}
		original, err := app.conversation.GetMessage(req.ReplyToMessageUUID)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if aifeedback.ValidateOriginal(original, cuuid) != nil {
			return r.SendErrorEnvelope(400, "Reply source must be an incoming customer message in this conversation", nil, envelope.InputError)
		}
		if (req.AIResponse != nil && req.AIResponse.SourceUUID != req.ReplyToMessageUUID) || (req.AIFeedbackOriginalMessageUUID != "" && req.AIFeedbackOriginalMessageUUID != req.ReplyToMessageUUID) {
			return r.SendErrorEnvelope(400, "Conflicting reply sources", nil, envelope.InputError)
		}
	}
	// Get media for all attachments, skip any already associated with a model.
	media, err := getUnassociatedMedia(app, req.Attachments)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	rootURL, _ := app.setting.GetAppRootURL()

	// Create contact message.
	if req.SenderType == umodels.UserTypeContact {
		message, err := app.conversation.CreateContactMessage(media, int(conv.ContactID), cuuid, req.Message, cmodels.ContentTypeHTML, false)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		resolveQuotedCIDs(app, &message)
		resolveAttachmentCIDs(&message, rootURL)
		return r.SendEnvelope(message)
	}

	// Send private note.
	if req.Private {
		message, err := app.conversation.SendPrivateNote(media, user.ID, cuuid, req.Message, req.Mentions)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		resolveAttachmentCIDs(&message, rootURL)
		return r.SendEnvelope(message)
	}

	// Queue outgoing reply.
	meta := map[string]any{}
	if req.ReplyToMessageUUID != "" {
		meta["reply_to_message_uuid"] = req.ReplyToMessageUUID
	}
	if req.AIResponse != nil {
		meta["ai_response"] = req.AIResponse
	}
	if req.AIFeedbackOriginalMessageUUID != "" {
		meta[aifeedback.OriginalKey] = req.AIFeedbackOriginalMessageUUID
	}
	if req.EchoID != "" {
		meta["echo_id"] = req.EchoID
	}

	// Store the selected outgoing email template in message meta.
	// If no template is selected, the default outgoing template is used.
	if req.TemplateID > 0 {
		meta["template_id"] = req.TemplateID
	}

	message, err := app.conversation.QueueReply(media, conv.InboxID, user.ID, conv.ContactID, cuuid, req.Message, req.To, req.CC, req.BCC, meta)
	if errors.Is(err, conversation.ErrReplySource) {
		return r.SendErrorEnvelope(400, "Invalid reply source", nil, envelope.InputError)
	}
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	markAssignmentNotificationRead(app, conv, user)
	resolveQuotedCIDs(app, &message)
	resolveAttachmentCIDs(&message, rootURL)
	return r.SendEnvelope(message)
}

// resolveAttachmentCIDs replaces inline image cid: references in email message content
// with actual attachment URLs and resolves relative /uploads/ paths to absolute URLs.
func resolveAttachmentCIDs(msg *cmodels.Message, rootURL string) {
	for _, att := range msg.Attachments {
		if att.ContentID != "" && att.URL != "" {
			msg.Content = strings.ReplaceAll(msg.Content, "cid:"+att.ContentID, att.URL)
		}
	}
	if rootURL != "" {
		msg.Content = strings.ReplaceAll(msg.Content, `src="/uploads/`, `src="`+rootURL+`/uploads/`)
		msg.Content = strings.ReplaceAll(msg.Content, `src='/uploads/`, `src='`+rootURL+`/uploads/`)
	}
}

// resolveQuotedCIDs replaces cid: refs to media on other messages with signed URLs.
func resolveQuotedCIDs(app *App, msg *cmodels.Message) {
	refs, err := app.conversation.GetInlineMediaRefs(msg)
	if err != nil {
		app.lo.Error("error fetching inline media refs", "conversation_uuid", msg.ConversationUUID, "error", err)
		return
	}
	for _, ref := range refs {
		url := app.media.GetURL(ref.UUID, ref.ContentType, ref.Filename)
		msg.Content = strings.ReplaceAll(msg.Content, "cid:"+ref.ContentID, url)
	}
}
