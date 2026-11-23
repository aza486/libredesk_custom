package main

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/inbox"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	emailnotifier "github.com/abhinavxd/libredesk/internal/notification/providers/email"
	"github.com/abhinavxd/libredesk/internal/setting"
	smodels "github.com/abhinavxd/libredesk/internal/setting/models"
	"github.com/google/uuid"
	"github.com/zerodha/logf"
)

func notificationInput(message string) error {
	return envelope.NewError(envelope.InputError, message, nil)
}
func mailboxAddress(value string) string {
	address, err := mail.ParseAddress(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return strings.ToLower(address.Address)
}
func loadNotificationEmail(settings *setting.Manager) (smodels.EmailNotification, error) {
	var cfg smodels.EmailNotification
	if settings == nil {
		return cfg, nil
	} // Existing isolated Create fixtures use only legacy configuration.
	raw, err := settings.GetByPrefix("notification.email")
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(raw, &cfg)
	return cfg, err
}
func resolveNotificationInbox(inboxes *inbox.Manager, identity string, byAddress bool) (imodels.Inbox, error) {
	if strings.TrimSpace(identity) == "" {
		return imodels.Inbox{}, notificationInput("Select a public email inbox in Admin > Notifications; legacy conversation.internal_inbox_uuid is not configured")
	}
	if !byAddress {
		id, err := uuid.Parse(identity)
		if err != nil {
			return imodels.Inbox{}, notificationInput("conversation.internal_inbox_uuid / notification inbox must be a valid inbox UUID")
		}
		identity = id.String()
	}
	records, err := inboxes.GetRoutingRecords()
	if err != nil {
		return imodels.Inbox{}, err
	}
	var found []imodels.Inbox
	for _, record := range records {
		if (!byAddress && record.UUID == identity) || (byAddress && record.Channel == inbox.ChannelEmail && record.AccessMode == "public" && mailboxAddress(record.From) == mailboxAddress(identity) && mailboxAddress(identity) != "") {
			found = append(found, record)
		}
	}
	if len(found) != 1 {
		return imodels.Inbox{}, notificationInput("The configured notification/internal inbox does not exist, was deleted, or is ambiguous; select it again in Admin > Notifications")
	}
	record := found[0]
	if !record.Enabled {
		return record, notificationInput("The configured notification/internal inbox is disabled")
	}
	if record.Channel != inbox.ChannelEmail || record.AccessMode != "public" {
		return record, notificationInput("The configured notification/internal inbox must be a public email inbox")
	}
	address := mailboxAddress(record.From)
	if address == "" {
		return record, notificationInput("The selected inbox has an invalid sender address")
	}
	for _, other := range records {
		if other.ID != record.ID && other.Enabled && other.Channel == inbox.ChannelEmail && other.AccessMode == "public" && mailboxAddress(other.From) == address {
			return record, notificationInput("Multiple email inboxes use this sender address; the mailbox mapping is ambiguous")
		}
	}
	return record, nil
}

func internalEmailConfiguration(app *App) (string, string, error) {
	cfg, err := loadNotificationEmail(app.setting)
	if err != nil {
		return "", "", err
	}
	identity := strings.TrimSpace(app.internalConversation.InboxUUID)
	contact := strings.TrimSpace(app.internalConversation.ContactEmail)
	if cfg.TeamContactEmail != nil {
		contact = strings.TrimSpace(*cfg.TeamContactEmail)
	}
	if cfg.InboxUUID != nil {
		identity = strings.TrimSpace(*cfg.InboxUUID)
		if identity == "" {
			return "", "", notificationInput("No internal inbox selected in Admin > Notifications; explicit admin configuration disables the legacy inbox fallback")
		}
	} else if identity == "" && mailboxAddress(cfg.EmailAddress) != "" {
		record, e := resolveNotificationInbox(app.inbox, cfg.EmailAddress, true)
		if e != nil {
			return "", "", e
		}
		identity = record.UUID
	}
	return identity, contact, nil
}

// One persisted settings snapshot per delivery. No stale process-wide SMTP or
// routing cache; mailbox runtime handles existing SMTP pools/OAuth refresh.
type notificationEmailProvider struct {
	settings *setting.Manager
	inboxes  *inbox.Manager
	lo       *logf.Logger
}

func (*notificationEmailProvider) Name() string { return notifier.ProviderEmail }
func (p *notificationEmailProvider) Send(msg notifier.Message) error {
	cfg, err := loadNotificationEmail(p.settings)
	if err != nil {
		return err
	}
	if msg.Optional && !cfg.Enabled {
		return nil
	}
	if cfg.InboxUUID != nil && strings.TrimSpace(*cfg.InboxUUID) != "" {
		record, err := resolveNotificationInbox(p.inboxes, strings.TrimSpace(*cfg.InboxUUID), false)
		if err != nil {
			return err
		}
		transport, err := p.inboxes.Get(record.ID)
		if err != nil {
			return fmt.Errorf("selected notification inbox is not available for sending: %w", err)
		}
		for _, recipient := range msg.RecipientEmails {
			if mailboxAddress(recipient) == mailboxAddress(record.From) {
				return notificationInput("Notification recipient matches its sending inbox; self-delivery is not allowed")
			}
		}
		return transport.Send(cmodels.OutboundMessage{From: record.From, To: msg.RecipientEmails, Subject: msg.Subject, Content: msg.Content, ContentType: msg.ContentType, AltContent: msg.AltContent, Attachments: msg.Attachments, Headers: msg.Headers})
	}
	// Unconverted installations keep their existing dedicated SMTP transport.
	sender, err := emailnotifier.New([]imodels.SMTPConfig{{Host: cfg.Host, Port: cfg.Port, Username: cfg.Username, Password: cfg.Password, MaxConns: cfg.MaxConns, IdleTimeout: cfg.IdleTimeout, PoolWaitTimeout: cfg.WaitTimeout, AuthProtocol: cfg.AuthProtocol, MaxMessageRetries: cfg.MaxMsgRetries, TLSType: cfg.TLSType, TLSSkipVerify: cfg.TLSSkipVerify, HelloHostname: cfg.HelloHostname}}, emailnotifier.Opts{Lo: p.lo, FromEmail: cfg.EmailAddress})
	if err != nil {
		return err
	}
	defer sender.Close()
	return sender.Send(msg)
}
