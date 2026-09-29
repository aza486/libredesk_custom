package conversation

import (
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func initializePersonalVisibility(attrs map[string]any, ownerID int) {
	attrs["access_mode"] = "personal"
	attrs["owner_user_id"] = ownerID
	attrs["creator_id"] = ownerID
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
