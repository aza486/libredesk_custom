package conversation

import (
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func initializePersonalVisibility(attrs map[string]any, ownerID int) {
	attrs["access_mode"] = "personal"
	attrs["owner_user_id"] = ownerID
	delete(attrs, "creator_id")
	attrs["customer_visibility"] = false
	attrs["visibility_managers"] = []int{ownerID}
	attrs["visible_users"] = []int{ownerID}
}

func isPersonalConversation(raw json.RawMessage) bool {
	var attrs struct {
		AccessMode string `json:"access_mode"`
	}
	_ = json.Unmarshal(raw, &attrs)
	return attrs.AccessMode == "personal"
}

// Shared by initial personal assignment and the existing multi-assignment service.
// The caller owns the transaction and visibility updates.
func replaceUserAssignees(tx *sqlx.Tx, conversationID int, ids []int, actorID int) error {
	if _, err := tx.Exec(`DELETE FROM conversation_assignees WHERE conversation_id=$1 AND NOT (user_id=ANY($2::bigint[]))`, conversationID, pq.Array(ids)); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO conversation_assignees (conversation_id, user_id, assigned_by_user_id)
 SELECT $1, unnest($2::bigint[]), $3 ON CONFLICT (conversation_id, user_id) DO NOTHING`, conversationID, pq.Array(ids), actorID); err != nil {
		return err
	}
	var primary any
	if len(ids) > 0 {
		primary = ids[0]
	}
	_, err := tx.Exec(`UPDATE conversations SET assigned_user_id=$2 WHERE id=$1`, conversationID, primary)
	return err
}

func (m *Manager) canThreadIntoInbox(conversationID, inboxID int) (bool, error) {
	var allowed bool
	err := m.db.Get(&allowed, `SELECT c.inbox_id=$2 OR (COALESCE(c.custom_attributes->>'access_mode','public') <> 'personal' AND i.access_mode <> 'personal') FROM conversations c CROSS JOIN inboxes i WHERE c.id=$1 AND i.id=$2`, conversationID, inboxID)
	return allowed, err
}

// GetPersonalConversationsList restricts the mailbox route to its current owner,
// including administrators. The list query repeats this check to avoid a TOCTOU leak.
func (c *Manager) GetPersonalConversationsList(viewerID, inboxID int, highPriority bool, order, orderBy, filters string, page, pageSize int) ([]models.ConversationListItem, error) {
	var owned bool
	if err := c.db.Get(&owned, `SELECT EXISTS(SELECT 1 FROM inboxes WHERE id=$1 AND access_mode='personal' AND owner_user_id=$2)`, inboxID, viewerID); err != nil {
		return nil, err
	}
	if !owned {
		return nil, envelope.NewError(envelope.PermissionError, c.i18n.T("conversation.personalInboxOwnerOnly"), nil)
	}
	listType := models.PersonalConversations
	if highPriority {
		listType = models.PersonalHighPriorityConversations
	}
	return c.GetConversations(viewerID, viewerID, nil, []string{listType}, order, orderBy, filters, page, pageSize, inboxID)
}

// ValidateVisibleUserRemoval is also called before the handler's assignment guard,
// so an owner receives the same actionable error while assigned or unassigned.
func (c *Manager) ValidateVisibleUserRemoval(conversation *models.Conversation, userID int) error {
	var attrs struct {
		AccessMode  string `json:"access_mode"`
		OwnerUserID int    `json:"owner_user_id"`
	}
	_ = json.Unmarshal(conversation.CustomAttributes, &attrs)
	if (conversation.InboxAccessMode == "personal" || attrs.AccessMode == "personal") &&
		(conversation.InboxOwnerUserID.Int == userID || attrs.OwnerUserID == userID) {
		return envelope.NewError(envelope.InputError, c.i18n.T("conversation.personalOwnerVisibilityProtected"), nil)
	}
	return nil
}
