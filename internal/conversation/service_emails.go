package conversation

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"slices"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/jmoiron/sqlx"
)

// Shared by configuration writes and automatic publication. A registry update
// cannot commit between the last classification and a guarded message insert.
const serviceEmailLock = 2147483014

// The exact Unicode whitespace set used by strings.TrimSpace.
const serviceEmailWhitespaceSQL = `U&' \0009\000A\000B\000C\000D\0085\00A0\1680\2000\2001\2002\2003\2004\2005\2006\2007\2008\2009\200A\2028\2029\202F\205F\3000'`

func normalizeServiceEmail(address string) (string, error) {
	address = strings.TrimSpace(address)
	parsed, err := mail.ParseAddress(address)
	if err != nil || parsed.Address != address {
		return "", envelope.NewError(envelope.InputError, "Invalid email address", nil)
	}
	return strings.ToLower(address), nil
}

func (c *Manager) GetServiceEmailAddresses() ([]string, error) {
	addresses := []string{}
	err := c.db.Select(&addresses, `SELECT address FROM service_email_addresses ORDER BY address`)
	return addresses, err
}
func (c *Manager) AddServiceEmailAddress(address string) error {
	return c.mutateServiceEmailAddress(address, true)
}
func (c *Manager) RemoveServiceEmailAddress(address string) error {
	return c.mutateServiceEmailAddress(address, false)
}

func (c *Manager) mutateServiceEmailAddress(address string, add bool) error {
	address, err := normalizeServiceEmail(address)
	if err != nil {
		return err
	}
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock($1)`, serviceEmailLock); err != nil {
		return err
	}
	if add {
		var exists bool
		if err = tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM service_email_addresses WHERE LOWER(BTRIM(address,`+serviceEmailWhitespaceSQL+`))=$1)`, address); err != nil {
			return err
		}
		if exists {
			return envelope.NewError(envelope.InputError, "Service email address already exists", nil)
		}
		_, err = tx.Exec(`INSERT INTO service_email_addresses(address) VALUES($1)`, address)
	} else {
		_, err = tx.Exec(`DELETE FROM service_email_addresses WHERE LOWER(BTRIM(address,`+serviceEmailWhitespaceSQL+`))=$1`, address)
	}
	if err != nil {
		return err
	}
	// Only the system classification changes. Keep routing, assignments, status,
	// visibility, messages and notes. Personal and internal tickets are excluded.
	rows := []struct {
		ID   int    `db:"id"`
		UUID string `db:"uuid"`
	}{}
	err = tx.Select(&rows, `SELECT conversations.id,conversations.uuid FROM conversations JOIN users ON users.id=conversations.contact_id JOIN inboxes ON inboxes.id=conversations.inbox_id
 WHERE LOWER(BTRIM(users.email,`+serviceEmailWhitespaceSQL+`))=$1 AND COALESCE(inboxes.access_mode,'public')<>'personal'
 AND COALESCE(conversations.custom_attributes->>'access_mode','public')<>'personal'
 AND COALESCE((conversations.custom_attributes->>'private')::boolean,false)=false ORDER BY conversations.id FOR UPDATE OF conversations`, address)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if add {
			_, err = tx.Exec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,14) ON CONFLICT DO NOTHING`, row.ID)
		} else {
			_, err = tx.Exec(`DELETE FROM conversation_tags WHERE conversation_id=$1 AND tag_id=14`, row.ID)
		}
		if err != nil {
			return err
		}
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if c.wsHub != nil {
		for _, row := range rows {
			tags := []string{}
			if err := c.q.GetConversationTags.Select(&tags, row.UUID); err != nil {
				c.lo.Error("service mail refresh failed", "error", err)
				continue
			}
			c.broadcastTagMembershipUpdate(row.UUID, map[string]any{"tags": tags})
		}
		// No addresses or conversation data in this management-only invalidation.
		ids := []int{}
		for _, id := range c.wsHub.ConnectedUserIDs() {
			user, err := c.userStore.GetAgent(id, "")
			if err == nil && user.Enabled && (user.HasAdminRole() || slices.Contains(user.Permissions, "service_emails:manage") || slices.Contains(user.Permissions, "general_settings:manage")) {
				ids = append(ids, id)
			}
		}
		if len(ids) > 0 {
			c.broadcastToUsers(ids, wsmodels.Message{Type: wsmodels.MessageTypeConversationUpdate, Data: map[string]any{"service_emails_changed": true}})
		}
	}
	return nil
}

func (c *Manager) IsServiceEmailAddress(address string) (bool, error) {
	address, err := normalizeServiceEmail(address)
	if err != nil {
		return false, err
	}
	var exists bool
	err = c.db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM service_email_addresses WHERE LOWER(BTRIM(address,`+serviceEmailWhitespaceSQL+`))=$1)`, address)
	return exists, err
}

// Recheck under the registry lock, rather than persisting a stale pre-removal
// result from incoming-message hooks. Use the same conversation lock as Spam.
func (c *Manager) syncServiceEmailTag(uuid string) error {
	tx, err := c.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock_shared($1)`, serviceEmailLock); err != nil {
		return err
	}
	var id int
	if err = tx.Get(&id, `SELECT id FROM conversations WHERE uuid=$1 FOR UPDATE`, uuid); err != nil {
		return err
	}
	var service bool
	err = tx.Get(&service, `SELECT EXISTS(SELECT 1 FROM service_email_addresses s WHERE LOWER(BTRIM(s.address,`+serviceEmailWhitespaceSQL+`))=LOWER(BTRIM(u.email,`+serviceEmailWhitespaceSQL+`))) FROM conversations c JOIN users u ON u.id=c.contact_id JOIN inboxes i ON i.id=c.inbox_id WHERE c.id=$1 AND COALESCE(i.access_mode,'public')<>'personal' AND COALESCE(c.custom_attributes->>'access_mode','public')<>'personal' AND COALESCE((c.custom_attributes->>'private')::boolean,false)=false`, id)
	if err != nil {
		return err
	}
	if service {
		_, err = tx.Exec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,14) ON CONFLICT DO NOTHING`, id)
	} else {
		_, err = tx.Exec(`DELETE FROM conversation_tags WHERE conversation_id=$1 AND tag_id=14`, id)
	}
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if c.wsHub != nil {
		tags := []string{}
		if err = c.q.GetConversationTags.Select(&tags, uuid); err != nil {
			return err
		}
		c.broadcastTagMembershipUpdate(uuid, map[string]any{"tags": tags})
	}
	return nil
}

// Deliberately checks current configuration, not historical tags or folders.
// Removing an address permits future processing without replaying old work.
func checkAutomaticAIAllowed(q sqlx.Queryer, uuid string) error {
	var state struct {
		Email   string `db:"email"`
		Service bool   `db:"service"`
		Spam    bool   `db:"spam"`
	}
	err := sqlx.Get(q, &state, `SELECT COALESCE(users.email,'') AS email,
 EXISTS(SELECT 1 FROM service_email_addresses s WHERE LOWER(BTRIM(s.address,`+serviceEmailWhitespaceSQL+`))=LOWER(BTRIM(users.email,`+serviceEmailWhitespaceSQL+`))) AS service,
 `+spamExistsSQL+` AS spam FROM conversations JOIN users ON users.id=conversations.contact_id WHERE conversations.uuid=$1`, uuid)
	if err != nil {
		return fmt.Errorf("automatic AI classification unavailable: %w", err)
	}
	if _, err := normalizeServiceEmail(state.Email); err != nil {
		return fmt.Errorf("automatic AI classification unavailable: invalid sender")
	}
	if state.Service || state.Spam {
		return envelope.NewError(envelope.PermissionError, "Automatic AI processing blocked for service mail or spam", nil)
	}
	return nil
}

func automaticAIMeta(raw json.RawMessage) bool {
	var meta map[string]json.RawMessage
	if json.Unmarshal(raw, &meta) != nil {
		return false
	}
	return string(meta["automatic_email_ai"]) == "true" || meta["ai_analysis"] != nil
}

// Use on the existing insert transaction; DB errors fail closed without losing
// the incoming message or modifying manual employee actions.
func lockAutomaticAI(tx *sqlx.Tx, uuid string) error {
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock_shared($1)`, serviceEmailLock); err != nil {
		return err
	}
	return checkAutomaticAIAllowed(tx, uuid)
}

func (c *Manager) CheckAutomaticAIAllowed(uuid string) error {
	return c.automaticAIPublicationError(checkAutomaticAIAllowed(c.db, uuid))
}

func (c *Manager) automaticAIPublicationError(err error) error {
	if err == nil {
		return nil
	}
	c.lo.Error("automatic AI publication rejected", "error", err)
	if _, ok := err.(envelope.Error); ok {
		return err
	}
	return envelope.NewError(envelope.GeneralError, "Automatic AI classification unavailable; retry later", nil)
}
