package conversation

import (
	"encoding/json"
	"fmt"
	"slices"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	tagmanager "github.com/abhinavxd/libredesk/internal/tag"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// The protected name is already used by incoming-message classification. Do not
// assume a numeric ID: older installations and existing fixtures differ.
const customerTagName = tagmanager.SystemTagCustomerName

const customerTagSQL = `EXISTS (SELECT 1 FROM conversation_tags customer_ct JOIN tags customer_tag ON customer_tag.id=customer_ct.tag_id WHERE customer_ct.conversation_id=conversations.id AND customer_tag.name='` + customerTagName + `')`
const supportTeamSQL = `(SELECT min(id) FROM teams WHERE name='Kundensupport' HAVING count(*)=1)`

// Employee-only support context requires the existing support role or a sole
// support-team membership, and no conflicting external team memberships.
const supportOnlyUserSQL = `((EXISTS (SELECT 1 FROM user_roles route_ur JOIN roles route_role ON route_role.id=route_ur.role_id WHERE route_ur.user_id=route_user.id AND route_role.name='Kundensupport') OR EXISTS (SELECT 1 FROM team_members support_tm WHERE support_tm.user_id=route_user.id AND support_tm.team_id=` + supportTeamSQL + `)) AND NOT EXISTS (SELECT 1 FROM team_members other_tm WHERE other_tm.user_id=route_user.id AND other_tm.team_id IS DISTINCT FROM ` + supportTeamSQL + `))`

const externalResponsibilitySQL = `(CASE WHEN conversations.assigned_team_id IS NOT NULL THEN COALESCE(conversations.assigned_team_id<>` + supportTeamSQL + `,TRUE) ELSE EXISTS (SELECT 1 FROM conversation_assignees route_ca JOIN users route_user ON route_user.id=route_ca.user_id WHERE route_ca.conversation_id=conversations.id AND route_user.enabled AND route_user.deleted_at IS NULL AND route_user.type='agent' AND NOT (` + supportOnlyUserSQL + `)) END)`
const redistributedSQL = `(` + customerTagSQL + ` AND ` + externalResponsibilitySQL + `)`
const activeRedistributedSQL = `(` + redistributedSQL + ` AND COALESCE(conversation_statuses.category IN ('open','waiting'),FALSE))`

type customerRoute struct {
	Active        bool   `db:"customer_redistributed_active"`
	UUID          string `db:"uuid"`
	Customer      bool   `db:"customer_ticket"`
	Redistributed bool   `db:"customer_redistributed"`
}

// One batch per list page, inside the list's repeatable-read transaction. No
// per-ticket requests and no writes caused by a read or WebSocket refresh.
func loadCustomerRoutes(q sqlx.Queryer, uuids []string) (map[string]customerRoute, error) {
	result := map[string]customerRoute{}
	if len(uuids) == 0 {
		return result, nil
	}
	var rows []customerRoute
	err := sqlx.Select(q, &rows, `SELECT conversations.uuid, `+customerTagSQL+` AS customer_ticket, `+redistributedSQL+` AS customer_redistributed, `+activeRedistributedSQL+` AS customer_redistributed_active FROM conversations LEFT JOIN conversation_statuses ON conversation_statuses.id=conversations.status_id WHERE conversations.uuid=ANY($1::uuid[])`, pq.Array(uuids))
	for _, r := range rows {
		result[r.UUID] = r
	}
	return result, err
}
func populateCustomerRoutes(q sqlx.Queryer, items []models.ConversationListItem) error {
	uuids := make([]string, len(items))
	for i := range items {
		uuids[i] = items[i].UUID
	}
	routes, err := loadCustomerRoutes(q, uuids)
	if err != nil {
		return err
	}
	for i := range items {
		r := routes[items[i].UUID]
		items[i].CustomerTicket = r.Customer
		items[i].CustomerRedistributed = r.Redistributed
		items[i].CustomerRedistributedActive = r.Active
	}
	return nil
}

type customerAssignment struct {
	kind            string
	users           []int
	team            int
	expectedTeam    *int
	authorizeReturn CustomerReturnAuthorization
}

// mutateCustomerAssignment handles only existing customer-tagged conversations.
// All assignment, visibility and return-status writes share the row lock used by
// Spam and normal status transitions. Noncustomer paths retain their contracts.
func (c *Manager) mutateCustomerAssignment(uuid string, change customerAssignment, actor umodels.User) (bool, error) {
	tx, err := c.lockSpamConversation(uuid)
	if err != nil {
		return true, err
	}
	defer tx.Rollback()
	var customer bool
	if err = tx.Get(&customer, `SELECT `+customerTagSQL+` FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return true, err
	}
	if !customer {
		return false, nil
	}
	var before models.Conversation
	if err = tx.Stmtx(c.q.GetConversation).Get(&before, 0, uuid, ""); err != nil {
		return true, err
	}
	var wasExternal bool
	if err = tx.Get(&wasExternal, `SELECT `+externalResponsibilitySQL+` FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return true, err
	}
	spam, err := conversationHasSpam(tx, uuid)
	if err != nil {
		return true, err
	}
	if change.kind == "return" && (!wasExternal || spam) {
		return true, envelope.NewError(envelope.InputError, c.i18n.T("conversation.routing.returnUnavailable"), nil)
	}
	if change.kind == "return" && change.authorizeReturn != nil {
		before.CustomerTicket = true
		before.CustomerRedistributed = wasExternal
		allowed, e := customerReturnAllowed(tx, actor.ID, before, change.authorizeReturn)
		if e != nil {
			return true, e
		}
		if !allowed {
			return true, envelope.NewError(envelope.PermissionError, "Permission denied", nil)
		}
	}
	if change.expectedTeam != nil && (len(before.AssignedUserIDs) > 0 || before.AssignedUserID.Valid || before.AssignedTeamID.Int != *change.expectedTeam) {
		return true, ErrConversationAlreadyAssigned
	}
	users := make([]int, 0, len(before.AssignedUserIDs))
	for _, id := range before.AssignedUserIDs {
		users = append(users, int(id))
	}
	team := before.AssignedTeamID.Int
	switch change.kind {
	case "users":
		users = change.users
	case "team":
		team = change.team
		if team > 0 && before.AssignedTeamID.Int != team {
			users = nil
		}
	case "return":
		users = nil
		team = 0
	default:
		return true, fmt.Errorf("invalid customer assignment")
	}
	// Validate direct/internal callers too, before changing any stored state.
	if len(users) > 0 {
		var valid int
		if err = tx.Get(&valid, `SELECT count(*) FROM users WHERE id=ANY($1::bigint[]) AND type='agent' AND enabled AND deleted_at IS NULL`, pq.Array(users)); err != nil {
			return true, err
		}
		if valid != len(users) {
			return true, envelope.NewError(envelope.InputError, "Invalid customer assignee", nil)
		}
	}
	var supportIDs []int
	if err = tx.Select(&supportIDs, `SELECT id FROM teams WHERE name='Kundensupport' ORDER BY id`); err != nil {
		return true, err
	}
	support := 0
	if len(supportIDs) == 1 {
		support = supportIDs[0]
	}
	if team > 0 {
		var exists bool
		if err = tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM teams WHERE id=$1)`, team); err != nil {
			return true, err
		}
		if !exists {
			return true, envelope.NewError(envelope.InputError, "Invalid team", nil)
		}
	}
	// Explicit employee assignment can leave a support team when the selected
	// employees cannot unambiguously work in that context. External team context
	// always wins, including for employees who also belong to support.
	if change.kind == "users" && team == support && support > 0 && len(users) > 0 {
		var supportMembers int
		if err = tx.Get(&supportMembers, `SELECT count(*) FROM team_members WHERE team_id=$1 AND user_id=ANY($2::bigint[])`, support, pq.Array(users)); err != nil {
			return true, err
		}
		if supportMembers != len(users) {
			team = 0
		}
	}
	if err = replaceUserAssignees(tx, before.ID, users, actor.ID); err != nil {
		return true, err
	}
	var teamValue any
	if team > 0 {
		teamValue = team
	}
	if _, err = tx.Exec(`UPDATE conversations SET assigned_team_id=$2,updated_at=NOW() WHERE id=$1`, before.ID, teamValue); err != nil {
		return true, err
	}
	var external bool
	if err = tx.Get(&external, `SELECT `+externalResponsibilitySQL+` FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return true, err
	}
	returning := change.kind == "return" || (wasExternal && !external)
	if returning {
		// Explicit return and loss of the last external responsibility belong to
		// the organizational support area, which needs no physical team.
		if change.kind == "return" || team == 0 {
			team = 0
		} else {
			team = support
		}
		// A direct support employee assignment is retained; team assignment / explicit
		// return / loss of the last external responsibility clears external users.
		if change.kind != "users" || len(users) == 0 {
			users = nil
		} else {
			var onlySupport bool
			if err = tx.Get(&onlySupport, `SELECT NOT EXISTS(SELECT 1 FROM users route_user WHERE route_user.id=ANY($1::bigint[]) AND NOT (`+supportOnlyUserSQL+`))`, pq.Array(users)); err != nil {
				return true, err
			}
			if !onlySupport {
				users = nil
			}
		}
		if err = replaceUserAssignees(tx, before.ID, users, actor.ID); err != nil {
			return true, err
		}
		teamValue = nil
		if team > 0 {
			teamValue = team
		}
		if _, err = tx.Exec(`UPDATE conversations SET assigned_team_id=$2 WHERE id=$1`, before.ID, teamValue); err != nil {
			return true, err
		}
		if !spam {
			if _, err = tx.Stmtx(c.q.UpdateConversationStatus).Exec(uuid, models.StatusOpen, time.Time{}); err != nil {
				return true, err
			}
		}
	}
	attrs := map[string]any{}
	if err = json.Unmarshal(before.CustomAttributes, &attrs); err != nil {
		return true, err
	}
	// Extend the existing durable visibility behavior to customer team assignment.
	// Only a real assignment adds visibility; identical requests never re-add it.
	visible, _ := attrs["visible_users"].([]any)
	add := []int{}
	for _, id := range users {
		if !slices.Contains(before.AssignedUserIDs, int64(id)) {
			add = append(add, id)
		}
	}
	if team > 0 && team != before.AssignedTeamID.Int {
		var members []int
		if err = tx.Select(&members, `SELECT u.id FROM users u JOIN team_members tm ON tm.user_id=u.id WHERE tm.team_id=$1 AND u.type='agent' AND u.enabled AND u.deleted_at IS NULL`, team); err != nil {
			return true, err
		}
		add = append(add, members...)
	}
	for _, id := range add {
		if !slices.ContainsFunc(visible, func(v any) bool { return v == float64(id) || v == id }) {
			visible = append(visible, id)
		}
	}
	attrs["visible_users"] = visible
	raw, err := json.Marshal(attrs)
	if err != nil {
		return true, err
	}
	if _, err = tx.Exec(`UPDATE conversations SET custom_attributes=$2 WHERE id=$1`, before.ID, raw); err != nil {
		return true, err
	}
	if err = tx.Commit(); err != nil {
		return true, err
	}
	// Existing hooks run only after the atomic state is visible. Delivery failures
	// are logged and cannot turn a committed return into a reported failure.
	if returning && !spam && before.Status.String != models.StatusOpen {
		if err = c.afterStatusChange(uuid, models.StatusOpen, time.Time{}, before, actor); err != nil {
			c.lo.Error("customer return status hooks", "error", err)
		}
	}
	if team != before.AssignedTeamID.Int {
		if team > 0 {
			if err = c.RecordAssigneeTeamChange(uuid, team, actor); err != nil {
				c.lo.Error("customer team activity", "error", err)
			}
		}
		if current, e := c.GetConversation(0, uuid, ""); e == nil {
			if team > 0 {
				c.applyTeamSLA(current, team)
			}
			c.automation.EvaluateConversationUpdateRules(current, amodels.EventConversationTeamAssigned, amodels.PreviousValues(before), actor)
			c.BroadcastConversationToWidget(uuid, current.ContactID, current.InboxID, map[string]any{"assigned_team_id": current.AssignedTeamID})
		}
	}
	if (change.kind == "team" || change.kind == "return") && len(before.AssignedUserIDs) > 0 && len(users) == 0 {
		if current, e := c.GetConversation(0, uuid, ""); e == nil {
			c.webhookStore.TriggerEvent(wmodels.EventConversationUnassigned, map[string]any{"conversation_uuid": uuid, "actor_id": actor.ID, "conversation": current})
			c.BroadcastConversationToWidget(uuid, current.ContactID, current.InboxID, map[string]any{"assignee": nil, "assigned_team_id": current.AssignedTeamID})
		}
	}
	for _, id := range users {
		if !slices.Contains(before.AssignedUserIDs, int64(id)) {
			if err = c.afterUserAssignedHooks(uuid, id, actor, amodels.PreviousValues(before)); err != nil {
				c.lo.Error("customer assignment hooks", "error", err)
			}
		}
	}
	if next, e := c.GetConversationListItem(uuid); e == nil {
		if current, err := c.GetConversation(0, uuid, ""); err == nil {
			_ = json.Unmarshal(current.CustomAttributes, &attrs)
		}
		// Existing conversation_update reaches newly authorized destination viewers,
		// including those who never subscribed to the previous source list.
		c.broadcastTagMembershipUpdate(uuid, map[string]any{"assigned_team_id": next.AssignedTeamID, "assigned_user_id": next.AssignedUserID, "assigned_user_ids": next.AssignedUserIDs, "customer_ticket": next.CustomerTicket, "customer_redistributed": next.CustomerRedistributed, "customer_redistributed_active": next.CustomerRedistributedActive, "status": next.Status, "custom_attributes": attrs, "routing_changed": true})
	}
	return true, nil
}

func (c *Manager) ReturnCustomerToSupport(uuid string, actor umodels.User, authorization ...CustomerReturnAuthorization) error {
	change := customerAssignment{kind: "return"}
	if len(authorization) > 0 {
		change.authorizeReturn = authorization[0]
	}
	handled, err := c.mutateCustomerAssignment(uuid, change, actor)
	if !handled {
		return envelope.NewError(envelope.InputError, c.i18n.T("conversation.routing.returnUnavailable"), nil)
	}
	return err
}

// Customer visibility mutations use the assignment/Spam lock, preventing a
// concurrent forwarding from restoring a removed user or losing new members.
func (c *Manager) mutateCustomerVisibility(uuid string, userID int, remove bool) (bool, error) {
	tx, err := c.lockSpamConversation(uuid)
	if err != nil {
		return true, err
	}
	defer tx.Rollback()
	var customer bool
	if err = tx.Get(&customer, `SELECT `+customerTagSQL+` FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return true, err
	}
	if !customer {
		return false, nil
	}
	var conversation models.Conversation
	if err = tx.Stmtx(c.q.GetConversation).Get(&conversation, 0, uuid, ""); err != nil {
		return true, err
	}
	if remove {
		if err = c.ValidateVisibleUserRemoval(&conversation, userID); err != nil {
			return true, err
		}
	}
	attrs := map[string]any{}
	if err = json.Unmarshal(conversation.CustomAttributes, &attrs); err != nil {
		return true, err
	}
	if remove && attrs["creator_id"] == float64(userID) {
		return true, nil
	}
	visible, _ := attrs["visible_users"].([]any)
	if remove {
		visible = slices.DeleteFunc(visible, func(v any) bool { return v == float64(userID) || v == userID })
	} else if !slices.ContainsFunc(visible, func(v any) bool { return v == float64(userID) || v == userID }) {
		visible = append(visible, userID)
	}
	if visible == nil {
		visible = []any{}
	}
	attrs["visible_users"] = visible
	raw, err := json.Marshal(attrs)
	if err != nil {
		return true, err
	}
	if _, err = tx.Exec(`UPDATE conversations SET custom_attributes=$2,updated_at=NOW() WHERE id=$1`, conversation.ID, raw); err != nil {
		return true, err
	}
	if err = tx.Commit(); err != nil {
		return true, err
	}
	c.BroadcastConversationUpdate(uuid, map[string]any{"custom_attributes": attrs})
	return true, nil
}

// Incoming reopen keeps every valid external context. If the sole remaining
// individual responsibility was deactivated/deleted, restore support under the
// same row lock before the normal reopen SQL runs.
func (c *Manager) repairCustomerResponsibilityTx(tx *sqlx.Tx, uuid string, actorID int) (bool, error) {
	var invalid bool
	err := tx.Get(&invalid, `SELECT `+customerTagSQL+` AND assigned_team_id IS NULL
 AND EXISTS(SELECT 1 FROM conversation_assignees ca WHERE ca.conversation_id=conversations.id)
 AND NOT EXISTS(SELECT 1 FROM conversation_assignees ca JOIN users u ON u.id=ca.user_id WHERE ca.conversation_id=conversations.id AND u.type='agent' AND u.enabled AND u.deleted_at IS NULL)
 FROM conversations WHERE uuid=$1`, uuid)
	if err != nil || !invalid {
		return false, err
	}
	var id int
	if err = tx.Get(&id, `SELECT id FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return false, err
	}
	if err = replaceUserAssignees(tx, id, nil, actorID); err != nil {
		return false, err
	}
	_, err = tx.Exec(`UPDATE conversations SET assigned_team_id=NULL WHERE id=$1`, id)
	return err == nil, err
}

// General attribute saves must not restore a stale visibility list after a
// forwarding/removal or change the existing Creator/manager authority. Explicit
// visibility endpoints remain the only way to change those customer fields.
func (c *Manager) mutateCustomerAttributes(uuid string, attributes map[string]any) (bool, error) {
	tx, err := c.lockSpamConversation(uuid)
	if err != nil {
		return true, err
	}
	defer tx.Rollback()
	var customer bool
	if err = tx.Get(&customer, `SELECT `+customerTagSQL+` FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return true, err
	}
	if !customer {
		return false, nil
	}
	var raw []byte
	if err = tx.Get(&raw, `SELECT custom_attributes FROM conversations WHERE uuid=$1`, uuid); err != nil {
		return true, err
	}
	current := map[string]any{}
	if err = json.Unmarshal(raw, &current); err != nil {
		return true, err
	}
	next := map[string]any{}
	for key, value := range attributes {
		next[key] = value
	}
	for _, key := range []string{"visible_users", "visibility_managers", "customer_visibility", "creator_id", "private", "internal", "access_mode", "owner_user_id", "_spam_previous_state"} {
		if value, ok := current[key]; ok {
			next[key] = value
		} else {
			delete(next, key)
		}
	}
	raw, err = json.Marshal(next)
	if err != nil {
		return true, err
	}
	if _, err = tx.Stmtx(c.q.UpdateConversationCustomAttributes).Exec(uuid, raw); err != nil {
		return true, err
	}
	if err = tx.Commit(); err != nil {
		return true, err
	}
	c.BroadcastConversationUpdate(uuid, map[string]any{"custom_attributes": next})
	return true, nil
}

// The application supplies its existing Creator/manager policy. The current
// team membership is read from the same transaction as the return mutation.
type CustomerReturnAuthorization func(models.Conversation, bool) bool

func customerReturnAllowed(q sqlx.Queryer, viewerID int, conversation models.Conversation, authorize CustomerReturnAuthorization) (bool, error) {
	member := false
	if conversation.AssignedTeamID.Valid {
		if err := sqlx.Get(q, &member, `SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id=$1 AND user_id=$2)`, conversation.AssignedTeamID.Int, viewerID); err != nil {
			return false, err
		}
	}
	return authorize(conversation, member), nil
}
func (c *Manager) CustomerReturnAllowed(viewerID int, conversation models.Conversation, authorize CustomerReturnAuthorization) (bool, error) {
	return customerReturnAllowed(c.db, viewerID, conversation, authorize)
}
