package main

import (
	"slices"

	authzmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

// The ordinary Create API may request private/internal flags, but cannot supply
// ACLs or ownership, even when an administrator defined attributes with those keys.
func validateCreateSystemAttributes(req createConversationRequest) error {
	for key, value := range req.CustomAttributes {
		switch key {
		case "private", "internal":
			if _, ok := value.(bool); !ok {
				return envelope.NewError(envelope.InputError, key+" must be a boolean", nil)
			}
		default:
			if conversation.IsSystemAttribute(key) {
				return envelope.NewError(envelope.PermissionError, key+" is server managed", nil)
			}
		}
	}
	private, _ := req.CustomAttributes["private"].(bool)
	internal, _ := req.CustomAttributes["internal"].(bool)
	if internal && !private {
		return envelope.NewError(envelope.InputError, "Internal tickets must be private", nil)
	}
	if private && req.Initiator != umodels.UserTypeAgent {
		return envelope.NewError(envelope.InputError, "Private tickets require an agent initiator", nil)
	}
	return nil
}

func validatePrivateConversationCreate(req createConversationRequest, app *App, actorID int, inboxAccessMode string) error {
	// Personal inboxes have their own server-managed owner/visibility contract.
	if inboxAccessMode == "personal" {
		return envelope.NewError(envelope.InputError, "Private tickets require a public inbox", nil)
	}
	user, err := app.user.GetAgent(actorID, "")
	if err != nil {
		return err
	}
	if !user.Enabled || user.Type != umodels.UserTypeAgent || !slices.Contains(user.Permissions, authzmodels.PermConversationWrite) {
		return envelope.NewError(envelope.PermissionError, "Permission denied", nil)
	}
	if req.AssignedAgentID < 0 || req.AssignedTeamID < 0 {
		return envelope.NewError(envelope.InputError, "Invalid private ticket assignee", nil)
	}
	if req.AssignedAgentID > 0 {
		if !slices.Contains(user.Permissions, authzmodels.PermConversationsUpdateUserAssignee) {
			return envelope.NewError(envelope.PermissionError, "Permission denied", nil)
		}
		assignee, err := app.user.GetAgent(req.AssignedAgentID, "")
		if err != nil {
			return err
		}
		if !assignee.Enabled || assignee.Type != umodels.UserTypeAgent {
			return envelope.NewError(envelope.InputError, "Private ticket assignees must be active agents", nil)
		}
	}
	if req.AssignedTeamID > 0 {
		if !slices.Contains(user.Permissions, authzmodels.PermConversationsUpdateTeamAssignee) {
			return envelope.NewError(envelope.PermissionError, "Permission denied", nil)
		}
		if _, err := app.team.Get(req.AssignedTeamID); err != nil {
			return err
		}
	}
	return nil
}
