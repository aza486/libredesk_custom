package conversation

import (
	"encoding/json"
	"fmt"
	"slices"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	tagmanager "github.com/abhinavxd/libredesk/internal/tag"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Existing installation contract, also used by the sidebar's Spam view.
const spamExistsSQL = `EXISTS (SELECT 1 FROM conversation_tags spam_ct WHERE spam_ct.conversation_id=conversations.id AND spam_ct.tag_id=12)`

// Only the dedicated Spam filter selects the Spam view. In particular, 112,
// negative tag filters and optional OR branches must not disable exclusion.
func isSpamViewFilter(raw string) bool {
	var filters []struct {
		Model    string `json:"model"`
		Field    string `json:"field"`
		Operator string `json:"operator"`
		Value    string `json:"value"`
	}
	if json.Unmarshal([]byte(raw), &filters) != nil || len(filters) != 1 {
		return false
	}
	f := filters[0]
	if f.Model != "conversations" || f.Field != "tags" || f.Operator != "contains" {
		return false
	}
	var ids []int
	return json.Unmarshal([]byte(f.Value), &ids) == nil && len(ids) == 1 && ids[0] == 12
}

func conversationHasSpam(db sqlx.Queryer, uuid string) (bool, error) {
	var spam bool
	err := sqlx.Get(db, &spam, `SELECT `+spamExistsSQL+` FROM conversations WHERE uuid=$1`, uuid)
	return spam, err
}

// Every tag/status transition takes the same row lock. Inbox, routing and
// assignment fields are deliberately absent from these writes.
func (c *Manager) lockSpamConversation(uuid string) (*sqlx.Tx, error) {
	tx, err := c.db.Beginx()
	if err != nil {
		return nil, err
	}
	var id int
	if err = tx.Get(&id, `SELECT id FROM conversations WHERE uuid=$1 FOR UPDATE`, uuid); err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}

func (c *Manager) mutateTagsWithSpam(uuid, action string, names []string, actor umodels.User) ([]string, []string, bool, error) {
	tx, err := c.lockSpamConversation(uuid)
	if err != nil {
		return nil, nil, false, err
	}
	defer tx.Rollback()
	prev := []string{}
	if err = tx.Stmtx(c.q.GetConversationTags).Select(&prev, uuid); err != nil {
		return nil, nil, false, err
	}
	// Retain existing protection for unrelated system tags.
	for _, name := range names {
		if action == amodels.ActionAddTags && tagmanager.IsSystemTagName(name) && actor.ID != 41 {
			return nil, nil, false, envelope.NewError(envelope.InputError, "System tags cannot be added manually.", nil)
		}
		if action == amodels.ActionRemoveTags && tagmanager.IsSystemTagName(name) {
			return nil, nil, false, envelope.NewError(envelope.InputError, "System tags cannot be removed.", nil)
		}
	}
	if action == amodels.ActionSetTags {
		for _, name := range prev {
			if tagmanager.IsSystemTagName(name) && !slices.Contains(names, name) {
				return nil, nil, false, envelope.NewError(envelope.InputError, "System tags cannot be removed.", nil)
			}
		}
	}
	wasSpam, err := conversationHasSpam(tx, uuid)
	if err != nil {
		return nil, nil, false, err
	}
	var stmt *sqlx.Stmt
	switch action {
	case amodels.ActionAddTags:
		stmt = c.q.AddConversationTags
	case amodels.ActionSetTags:
		stmt = c.q.SetConversationTags
	case amodels.ActionRemoveTags:
		stmt = c.q.RemoveConversationTags
	default:
		return nil, nil, false, fmt.Errorf("invalid tag action: %s", action)
	}
	if _, err = tx.Stmtx(stmt).Exec(uuid, pq.Array(names)); err != nil {
		return nil, nil, false, err
	}
	isSpam, err := conversationHasSpam(tx, uuid)
	if err != nil {
		return nil, nil, false, err
	}
	changed := false
	if isSpam {
		// Missing canonical statuses cause rollback of the tag operation as well.
		var closed int
		if err = tx.Get(&closed, `SELECT id FROM conversation_statuses WHERE name=$1`, models.StatusClosed); err != nil {
			return nil, nil, false, err
		}
		var old int
		if err = tx.Get(&old, `SELECT status_id FROM conversations WHERE uuid=$1`, uuid); err != nil {
			return nil, nil, false, err
		}
		if !wasSpam {
			_, err = tx.Exec(`UPDATE conversations SET custom_attributes=COALESCE(custom_attributes,'{}'::jsonb) || jsonb_build_object('_spam_previous_state',jsonb_build_object('status_id',status_id,'closed_at',closed_at,'resolved_at',resolved_at)) WHERE uuid=$1`, uuid)
			if err != nil {
				return nil, nil, false, err
			}
		}
		if old != closed {
			_, err = tx.Stmtx(c.q.UpdateConversationStatus).Exec(uuid, models.StatusClosed, nil)
			if err != nil {
				return nil, nil, false, err
			}
			changed = true
		}
	} else if wasSpam {
		// Legacy Spam without a snapshot stays closed. Snoozed resumes as Open.
		var before int
		if err = tx.Get(&before, `SELECT status_id FROM conversations WHERE uuid=$1`, uuid); err != nil {
			return nil, nil, false, err
		}
		_, err = tx.Exec(`UPDATE conversations c SET
   status_id=COALESCE((SELECT CASE WHEN s.name='Snoozed' THEN (SELECT id FROM conversation_statuses WHERE name='Open') ELSE s.id END FROM conversation_statuses s WHERE to_jsonb(s.id)=c.custom_attributes->'_spam_previous_state'->'status_id'),c.status_id),
   closed_at=CASE WHEN c.custom_attributes ? '_spam_previous_state' THEN (c.custom_attributes->'_spam_previous_state'->>'closed_at')::timestamptz ELSE c.closed_at END,
   resolved_at=CASE WHEN c.custom_attributes ? '_spam_previous_state' THEN (c.custom_attributes->'_spam_previous_state'->>'resolved_at')::timestamptz ELSE c.resolved_at END,
   snoozed_until=NULL,
   custom_attributes=c.custom_attributes-'_spam_previous_state',updated_at=NOW()
   WHERE uuid=$1`, uuid)
		if err != nil {
			return nil, nil, false, err
		}
		var after int
		if err = tx.Get(&after, `SELECT status_id FROM conversations WHERE uuid=$1`, uuid); err != nil {
			return nil, nil, false, err
		}
		changed = before != after
	}
	next := []string{}
	if err = tx.Stmtx(c.q.GetConversationTags).Select(&next, uuid); err != nil {
		return nil, nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, nil, false, err
	}
	return prev, next, changed, nil
}
