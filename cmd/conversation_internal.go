package main

import (
	"errors"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/google/uuid"
)

// Optional deployment configuration. Resolve only when internal creation is used,
// so installation and unrelated ticket types never require an email inbox.
type internalConversationConfig struct {
	InboxUUID    string
	ContactEmail string
}

func prepareInternalConversation(req *createConversationRequest, app *App, actorID int) error {
	configuredUUID := strings.TrimSpace(app.internalConversation.InboxUUID)
	if configuredUUID == "" {
		return envelope.NewError(envelope.InputError, "Internal ticket inbox is not configured; set conversation.internal_inbox_uuid", nil)
	}
	id, err := uuid.Parse(configuredUUID)
	if err != nil {
		return envelope.NewError(envelope.InputError, "conversation.internal_inbox_uuid must be a valid inbox UUID", nil)
	}
	record, err := app.inbox.GetDBRecord(id.String())
	if err != nil {
		var apiErr envelope.Error
		if errors.As(err, &apiErr) && apiErr.ErrorType == envelope.InputError {
			return envelope.NewError(envelope.InputError, "The configured internal ticket inbox does not exist or was deleted", nil)
		}
		return err
	}
	if !record.Enabled {
		return envelope.NewError(envelope.InputError, "The configured internal ticket inbox is disabled", nil)
	}
	if record.Channel != inbox.ChannelEmail || record.AccessMode != "public" {
		return envelope.NewError(envelope.InputError, "The configured internal ticket inbox must be a public email inbox", nil)
	}
	// Existing API clients may still send the actual ID. Reject contradictory
	// routing; the UI omits the field and never needs to know deployment IDs.
	if req.InboxID != 0 && req.InboxID != record.ID {
		return envelope.NewError(envelope.InputError, "Internal tickets must use the configured inbox; omit inbox_id or use its current ID", nil)
	}
	req.InboxID = record.ID
	if err := validatePrivateConversationCreate(*req, app, actorID, record.AccessMode); err != nil {
		return err
	}

	// Preserve the established email-message/recipient contract. Internal tickets
	// still use a contact, but its identity cannot come from stale client fields.
	req.Email = strings.ToLower(strings.TrimSpace(app.internalConversation.ContactEmail))
	req.FirstName, req.LastName = "Internal", "Ticket"
	if req.AssignedAgentID > 0 {
		agent, err := app.user.GetAgent(req.AssignedAgentID, "")
		if err != nil {
			return err
		}
		req.Email = strings.ToLower(strings.TrimSpace(agent.Email.String))
		req.FirstName, req.LastName = agent.FirstName, agent.LastName
	} else if req.Email == "" {
		return envelope.NewError(envelope.InputError, "Internal tickets without an assigned agent require conversation.internal_contact_email", nil)
	}
	// In particular, a contact selected earlier in external mode cannot replace
	// the intended internal recipient via its external ID or overwrite its data.
	req.ExternalUserID = ""
	req.ReuseContact = true
	return nil
}
