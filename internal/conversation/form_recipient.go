package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/mail"
	"slices"
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
	managed, originalUUID, err := m.formRecipientOrigin(uuid)
	if err != nil {
		return models.ReplyRecipient{Managed: true, Blocked: true}, err
	}
	result.Managed = managed
	if !managed {
		return result, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
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
 WHERE conversation_uuid=$1 AND message_uuid=$2 AND is_form=true AND direction='inbound'`, uuid, originalUUID)
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

// Classification uses only LibreDesk; sending never depends on pg-n8n.
func (m *Manager) formRecipientOrigin(uuid string) (managed bool, originalUUID string, err error) {
	if m.formRecipients == nil {
		return false, "", nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var original struct {
		UUID    string
		Sender  string
		Contact string
	}
	err = m.db.GetContext(ctx, &original, `SELECT COALESCE(cm.uuid::text,'') AS uuid, COALESCE(cm.meta->'from'->>0,u.email,'') AS sender, COALESCE(u.email,'') AS contact
 FROM conversations c JOIN users u ON u.id=c.contact_id
 LEFT JOIN LATERAL (SELECT uuid,meta FROM conversation_messages WHERE conversation_id=c.id AND type='incoming' AND NOT private ORDER BY created_at,id LIMIT 1) cm ON true
 WHERE c.uuid=$1`, uuid)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", errFormRecipient
	}
	for _, sender := range m.formRecipients.TechnicalSenders {
		parsed, e := mail.ParseAddress(original.Sender)
		if (e == nil && strings.EqualFold(parsed.Address, strings.TrimSpace(sender))) || strings.EqualFold(original.Contact, strings.TrimSpace(sender)) {
			managed = true
			break
		}
	}
	return managed, original.UUID, nil
}

const RecipientConfirmationKey = "recipient_confirmation"

// Created only for an explicit authenticated employee send. Bind the decision to
// the actual recipients so a later metadata/recipient change cannot reuse it.
type RecipientConfirmation struct {
	ActorID int      `json:"actor_id"`
	To      []string `json:"to"`
	CC      []string `json:"cc"`
	BCC     []string `json:"bcc"`
}

func ConfirmReplyRecipients(actorID int, to, cc, bcc []string) RecipientConfirmation {
	return RecipientConfirmation{ActorID: actorID, To: stringutil.RemoveEmpty(to), CC: stringutil.RemoveEmpty(cc), BCC: stringutil.RemoveEmpty(bcc)}
}

var errRecipientReview = envelope.NewError(envelope.ConflictError, "Empfänger dieser Formularantwort bitte im Antworteditor prüfen und erneut senden.", nil)

func (m *Manager) checkFormRecipientDecision(uuid string, senderID int, to, cc, bcc []string, meta json.RawMessage) error {
	if m.formRecipients == nil || len(to) == 0 {
		return nil
	}
	// Only the ambiguous legacy default needs review. Other queued addresses are
	// never changed or compared with the form suggestion.
	for _, recipient := range to {
		parsed, e := mail.ParseAddress(recipient)
		if e != nil {
			return nil
		} // General address validation handles malformed input.
		technical := false
		for _, sender := range m.formRecipients.TechnicalSenders {
			if strings.EqualFold(parsed.Address, strings.TrimSpace(sender)) {
				technical = true
				break
			}
		}
		if !technical {
			return nil
		}
	}
	var metadata struct {
		Confirmation RecipientConfirmation `json:"recipient_confirmation"`
	}
	if json.Unmarshal(meta, &metadata) == nil {
		c := metadata.Confirmation
		if c.ActorID == senderID && senderID > 0 && slices.Equal(c.To, to) && slices.Equal(c.CC, cc) && slices.Equal(c.BCC, bcc) {
			return nil
		}
	}
	managed, _, err := m.formRecipientOrigin(uuid)
	if err != nil {
		return err
	}
	if managed {
		return errRecipientReview
	}
	return nil
}

// Use the normal RFC mailbox syntax for all reply recipients, independently of
// form suggestions. Display names remain supported by the mail transport.
func validateReplyRecipients(to, cc, bcc []string) error {
	if len(to) == 0 {
		return envelope.NewError(envelope.InputError, "Mindestens ein Empfänger ist erforderlich.", nil)
	}
	for _, list := range [][]string{to, cc, bcc} {
		for _, value := range list {
			if _, err := mail.ParseAddress(value); err != nil || strings.ContainsAny(value, "\r\n") {
				return envelope.NewError(envelope.InputError, "Ungültige E-Mail-Adresse in TO, CC oder BCC.", nil)
			}
		}
	}
	return nil
}
