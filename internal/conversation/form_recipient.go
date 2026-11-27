package conversation

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
)

// FormRecipients reads mail_events and cases, independently of Learning permissions.
type FormRecipients struct {
	DB               *sqlx.DB
	TechnicalSenders []string
}

var errFormRecipient = envelope.NewError(envelope.ConflictError, "Formular-Empfänger nicht sicher ermittelbar. Bitte später erneut versuchen oder die Formulardaten prüfen.", nil)

func validFormAddress(raw string, technical []string) (string, error) {
	address := strings.TrimSpace(raw)
	if !stringutil.ValidEmail(address) || strings.ContainsAny(raw, "\r\n") {
		return "", errFormRecipient
	}
	for _, sender := range technical {
		if strings.EqualFold(address, strings.TrimSpace(sender)) {
			return "", errFormRecipient
		}
	}
	return address, nil
}

// ResolveReplyRecipient must be called only after API conversation authorization.
// The first public inbound message anchors the lookup, even after customer follow-ups.
func (m *Manager) ResolveReplyRecipient(uuid string) (models.ReplyRecipient, error) {
	result := models.ReplyRecipient{}
	if m.formRecipients == nil {
		return result, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var original struct {
		UUID    string
		Sender  string
		Contact string
	}
	err := m.db.GetContext(ctx, &original, `SELECT COALESCE(cm.uuid::text,'') AS uuid, COALESCE(cm.meta->'from'->>0,u.email,'') AS sender, COALESCE(u.email,'') AS contact
 FROM conversations c JOIN users u ON u.id=c.contact_id
 LEFT JOIN LATERAL (SELECT uuid,meta FROM conversation_messages WHERE conversation_id=c.id AND type='incoming' AND NOT private ORDER BY created_at,id LIMIT 1) cm ON true
 WHERE c.uuid=$1`, uuid)
	if errors.Is(err, sql.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return models.ReplyRecipient{Managed: true, Blocked: true}, errFormRecipient
	}
	for _, sender := range m.formRecipients.TechnicalSenders {
		parsed, e := mail.ParseAddress(original.Sender)
		if (e == nil && strings.EqualFold(parsed.Address, strings.TrimSpace(sender))) || strings.EqualFold(original.Contact, strings.TrimSpace(sender)) {
			result.Managed = true
			break
		}
	}
	if !result.Managed {
		return result, nil
	}
	result.Blocked = true
	if m.formRecipients.DB == nil {
		return result, errFormRecipient
	}
	tx, err := m.formRecipients.DB.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return result, errFormRecipient
	}
	defer tx.Rollback()
	var addresses []sql.NullString
	// Both identifiers are bound to local data; no browser-provided message IDs.
	err = tx.SelectContext(ctx, &addresses, `SELECT sender_email FROM public.mail_events
 WHERE conversation_uuid=$1 AND message_uuid=$2 AND is_form=true AND direction='inbound'`, uuid, original.UUID)
	if err != nil || len(addresses) == 0 {
		return result, errFormRecipient
	}
	for _, raw := range addresses {
		address, e := validFormAddress(raw.String, m.formRecipients.TechnicalSenders)
		if !raw.Valid || e != nil || (result.Email != "" && !strings.EqualFold(result.Email, address)) {
			result.Email = ""
			return result, errFormRecipient
		}
		result.Email = address
	}
	// cases is corroboration only; never a fallback source.
	var cases []sql.NullString
	if err := tx.SelectContext(ctx, &cases, `SELECT customer_email FROM public.cases WHERE conversation_uuid=$1`, uuid); err != nil {
		result.Email = ""
		return result, errFormRecipient
	}
	for _, customer := range cases {
		if !customer.Valid || strings.TrimSpace(customer.String) == "" {
			continue
		}
		address, e := validFormAddress(customer.String, m.formRecipients.TechnicalSenders)
		if e != nil || !strings.EqualFold(address, result.Email) {
			result.Email = ""
			return result, errFormRecipient
		}
	}
	result.Blocked = false
	return result, nil
}
