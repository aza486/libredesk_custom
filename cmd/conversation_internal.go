package main

import (
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
)

// Optional deployment configuration. Resolve only when internal creation is used,
// so installation and unrelated ticket types never require an email inbox.
type internalConversationConfig struct {
	InboxUUID    string
	ContactEmail string
}

func prepareInternalConversation(req *createConversationRequest, app *App, actorID int) error {
	configuredUUID, legacyContact, err := internalEmailConfiguration(app)
	if err != nil {
		return err
	}
	record, err := resolveNotificationInbox(app.inbox, configuredUUID, false)
	if err != nil {
		return err
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

	// Team-only tickets have no external contact or recipient. The creator is
	// the message author, never a synthetic contact.
	req.Email, req.FirstName, req.LastName = "", "", ""
	if req.AssignedAgentID > 0 {
		agent, err := app.user.GetAgent(req.AssignedAgentID, "")
		if err != nil {
			return err
		}
		req.Email = strings.ToLower(strings.TrimSpace(agent.Email.String))
		req.FirstName, req.LastName = agent.FirstName, agent.LastName
	}
	// Preserve the legacy explicit recipient for unassigned internal email
	// requests. A team-only assignment never consumes this setting.
	if req.AssignedAgentID == 0 && req.AssignedTeamID == 0 {
		req.Email = strings.ToLower(strings.TrimSpace(legacyContact))
		req.FirstName, req.LastName = "Internal", "Ticket"
	}
	if mailboxAddress(req.Email) != "" && mailboxAddress(req.Email) == mailboxAddress(record.From) {
		return notificationInput("The internal ticket recipient matches the sending inbox; use a different employee address")
	}
	// In particular, a contact selected earlier in external mode cannot replace
	// the intended internal recipient via its external ID or overwrite its data.
	req.ExternalUserID = ""
	req.ReuseContact = true
	return nil
}
