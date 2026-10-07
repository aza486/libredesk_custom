package inbox

import (
	"errors"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/lib/pq"
)

func validateAccessFields(in imodels.Inbox) error {
	ownerIDs := personalOwnerIDs(in)
	if in.AccessMode != "public" && in.AccessMode != "personal" {
		return envelope.NewError(envelope.InputError, "access_mode must be public or personal", nil)
	}
	if in.AccessMode == "public" && len(ownerIDs) > 0 {
		return envelope.NewError(envelope.InputError, "Public inboxes cannot have an owner", nil)
	}
	if in.AccessMode == "personal" && (len(ownerIDs) == 0 || in.Channel != "email") {
		return envelope.NewError(envelope.InputError, "Personal email inboxes require an active owner", nil)
	}
	return nil
}

func personalOwnerIDs(in imodels.Inbox) []int64 {
	ids := in.OwnerUserIDs
	if len(ids) == 0 && in.OwnerUserID.Valid {
		ids = []int64{int64(in.OwnerUserID.Int)}
	}
	seen := make(map[int64]bool, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	return unique
}

// ValidateAccess checks inbox access fields and resolves an active employee owner.
func (m *Manager) ValidateAccess(in imodels.Inbox) error {
	if err := validateAccessFields(in); err != nil {
		return err
	}
	if in.AccessMode == "personal" {
		var active bool
		ownerIDs := personalOwnerIDs(in)
		var validCount int
		if err := m.db.Get(&validCount, `SELECT COUNT(DISTINCT id) FROM users WHERE id = ANY($1::bigint[]) AND type='agent' AND email IS DISTINCT FROM 'System' AND enabled AND deleted_at IS NULL`, pq.Array(ownerIDs)); err != nil {
			return err
		}
		active = validCount == len(ownerIDs)
		if !active {
			return envelope.NewError(envelope.InputError, "Personal inbox owners must be active employees", nil)
		}
	}
	return nil
}

func accessConstraintError(err error) error {
	var pg *pq.Error
	if errors.As(err, &pg) && pg.Constraint == "inbox_access_mode_locked" {
		return envelope.NewError(envelope.ConflictError, "Cannot change access mode: this inbox already contains conversations", nil)
	}
	return nil
}

// PersonalInboxSummary intentionally excludes configuration and credentials.
type PersonalInboxSummary struct {
	ID           int           `db:"id" json:"id"`
	Name         string        `db:"name" json:"name"`
	OwnerUserIDs pq.Int64Array `db:"owner_user_ids" json:"owner_user_ids"`
}

func (m *Manager) GetOwnPersonalInboxes(userID int) ([]PersonalInboxSummary, error) {
	inboxes := make([]PersonalInboxSummary, 0)
	err := m.db.Select(&inboxes, `SELECT i.id,i.name,COALESCE(ARRAY(SELECT pio.user_id FROM personal_inbox_owners pio WHERE pio.inbox_id=i.id ORDER BY pio.created_at,pio.user_id),ARRAY[]::BIGINT[]) AS owner_user_ids FROM inboxes i WHERE i.access_mode='personal' AND i.enabled IS TRUE AND i.deleted_at IS NULL AND EXISTS (SELECT 1 FROM personal_inbox_owners pio WHERE pio.inbox_id=i.id AND pio.user_id=$1) ORDER BY i.name,i.id`, userID)
	return inboxes, err
}
