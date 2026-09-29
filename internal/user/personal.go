package user

import (
	"errors"
	"fmt"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
)

func personalOwnerError(err error) error {
	var pg *pq.Error
	if errors.As(err, &pg) && pg.Constraint == "personal_inbox_owner_active" {
		return envelope.NewError(envelope.ConflictError, "Reassign personal inboxes before deleting or disabling their owner", nil)
	}
	return nil
}

func personalContactCondition(viewer models.User) string {
	return fmt.Sprintf(`(NOT EXISTS (SELECT 1 FROM conversations pc WHERE pc.contact_id=users.id)
 OR EXISTS (SELECT 1 FROM conversations pc WHERE pc.contact_id=users.id AND %s))`, authz.PersonalAccessSQL("pc", viewer.ID, viewer.HasAdminRole()))
}

func (u *Manager) CanAccessContact(id int, viewer models.User) (bool, error) {
	var allowed bool
	err := u.db.Get(&allowed, `SELECT EXISTS(SELECT 1 FROM users WHERE users.id=$1 AND `+personalContactCondition(viewer)+`)`, id)
	return allowed, err
}
