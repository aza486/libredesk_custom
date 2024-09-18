package authz

import (
	"fmt"
	"strconv"
	"strings"

	cmodels "github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

// TODO: pass logger.
// Enforcer is a struct that holds the Casbin enforcer
type Enforcer struct {
	enforcer *casbin.Enforcer
}

const casbinModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj && r.act == p.act
`

// NewEnforcer initializes a new Enforcer with the hardcoded model
func NewEnforcer() (*Enforcer, error) {
	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin model: %v", err)
	}

	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %v", err)
	}

	return &Enforcer{enforcer: e}, nil
}

// LoadPermissions adds the user's permissions to the Casbin enforcer if not already present
func (e *Enforcer) LoadPermissions(user umodels.User) error {
	for _, perm := range user.Permissions {
		parts := strings.Split(perm, ":")
		if len(parts) != 2 {
			return fmt.Errorf("invalid permission format: %s", perm)
		}

		permObj, permAct := parts[0], parts[1]

		ok, err := e.enforcer.HasPolicy(strconv.Itoa(user.ID), permObj, permAct)
		if err != nil || !ok {
			if _, err := e.enforcer.AddPolicy(strconv.Itoa(user.ID), permObj, permAct); err != nil {
				return fmt.Errorf("failed to add policy: %v", err)
			}
		}
	}
	return nil
}

// Enforce checks if a user has permission to perform an action on an object
func (e *Enforcer) Enforce(user umodels.User, obj, act string) (bool, error) {
	// Load permissions before enforcing
	err := e.LoadPermissions(user)
	if err != nil {
		return false, err
	}

	// Check if the user has the required permission
	allowed, err := e.enforcer.Enforce(strconv.Itoa(user.ID), obj, act)
	if err != nil {
		return false, fmt.Errorf("error checking permission: %v", err)
	}
	return allowed, nil
}

// EnforceConversationAccess checks if a user has access to a conversation based on their permissions.
// It returns true if the user has read_all permission, or read_team permission and is in the assigned team,
// or read_assigned permission and is the assigned user. Returns false otherwise.
func (e *Enforcer) EnforceConversationAccess(user umodels.User, conversation cmodels.Conversation) (bool, error) {
	// Check for `read_all` permission
	allowed, err := e.enforcer.Enforce(strconv.Itoa(user.ID), "conversations", "read_all")
	if err != nil {
		return false, envelope.NewError(envelope.GeneralError, "Error checking permissions", nil)
	}
	if allowed {
		return true, nil
	}

	// Check for `read_team` permission
	allowed, err = e.enforcer.Enforce(strconv.Itoa(user.ID), "conversations", "read_team")
	if err != nil {
		return false, envelope.NewError(envelope.GeneralError, "Error checking permissions", nil)
	}
	if allowed {
		for _, teamID := range user.Teams.IDs() {
			if conversation.AssignedTeamID.Int == teamID {
				return true, nil
			}
		}
	}

	// Check for `read_assigned` permission
	allowed, err = e.enforcer.Enforce(strconv.Itoa(user.ID), "conversations", "read_assigned")
	if err != nil {
		return false, envelope.NewError(envelope.GeneralError, "Error checking permissions", nil)
	}
	if allowed && conversation.AssignedUserID.Int == user.ID {
		return true, nil
	}

	return false, nil
}
