package inbox

import (
	"errors"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/lib/pq"
)

func validateAccessFields(in imodels.Inbox) error {
	if in.AccessMode != "public" && in.AccessMode != "personal" {
		return envelope.NewError(envelope.InputError, "access_mode must be public or personal", nil)
	}
	if in.AccessMode == "public" && in.OwnerUserID.Valid {
		return envelope.NewError(envelope.InputError, "Public inboxes cannot have an owner", nil)
	}
	if in.AccessMode == "personal" && (!in.OwnerUserID.Valid || in.OwnerUserID.Int <= 0 || in.Channel != "email") {
		return envelope.NewError(envelope.InputError, "Personal email inboxes require an active owner", nil)
	}
	return nil
}

// ValidateAccess checks inbox access fields and resolves an active employee owner.
func (m *Manager) ValidateAccess(in imodels.Inbox) error {
	if err := validateAccessFields(in); err != nil {
		return err
	}
	if in.AccessMode == "personal" {
		var active bool
		if err := m.db.Get(&active, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND type='agent' AND email IS DISTINCT FROM 'System' AND enabled AND deleted_at IS NULL)`, in.OwnerUserID.Int); err != nil {
			return err
		}
		if !active {
			return envelope.NewError(envelope.InputError, "Personal inbox owner must be an active employee", nil)
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
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
}

func (m *Manager) GetOwnPersonalInboxes(userID int) ([]PersonalInboxSummary, error) {
	inboxes := make([]PersonalInboxSummary, 0)
	err := m.db.Select(&inboxes, `SELECT id,name FROM inboxes WHERE access_mode='personal' AND owner_user_id=$1 AND deleted_at IS NULL ORDER BY name,id`, userID)
	return inboxes, err
}
