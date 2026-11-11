package inbox

import (
	"database/sql"
	"errors"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/jmoiron/sqlx"
)

// Outgoing templates are shared with all agents; notification templates are not
// selectable signatures. Keep the row locked until the inbox write commits.
func (m *Manager) validateDefaultSignature(tx *sqlx.Tx, inbox imodels.Inbox) error {
	if !inbox.DefaultSignatureID.Valid {
		return nil
	}
	invalid := func() error {
		return envelope.NewError(envelope.InputError, m.i18n.T("admin.inbox.defaultSignature.invalid"), nil)
	}
	if inbox.Channel != ChannelEmail || inbox.DefaultSignatureID.Int <= 0 {
		return invalid()
	}
	var typ string
	err := tx.Get(&typ, `SELECT type FROM templates WHERE id=$1 FOR SHARE`, inbox.DefaultSignatureID.Int)
	if errors.Is(err, sql.ErrNoRows) {
		return invalid()
	}
	if err != nil {
		return err
	}
	if typ != "email_outgoing" {
		return invalid()
	}
	return nil
}
