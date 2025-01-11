package email

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/user"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime"
	"github.com/volatiletech/null/v9"
)

const (
	DefaultReadInterval = time.Duration(5 * time.Minute)
)

// ReadIncomingMessages reads and processes incoming messages from an IMAP server based on the provided configuration.
func (e *Email) ReadIncomingMessages(ctx context.Context, cfg IMAPConfig) error {
	readInterval, err := time.ParseDuration(cfg.ReadInterval)
	if err != nil {
		e.lo.Warn("could not parse IMAP read interval, using the default read interval", "interval", cfg.ReadInterval, "inbox_id", e.Identifier(), "error", err)
		readInterval = DefaultReadInterval
	}

	readTicker := time.NewTicker(readInterval)
	defer readTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-readTicker.C:
			if err := e.processMailbox(cfg); err != nil {
				e.lo.Error("error processing mailbox", "error", err)
			}
		}
	}
}

// processMailbox processes emails in the specified mailbox.
func (e *Email) processMailbox(cfg IMAPConfig) error {
	e.lo.Debug("processing emails from mailbox", "mailbox", cfg.Mailbox, "inbox_id", e.Identifier())
	client, err := imapclient.DialTLS(cfg.Host+":"+fmt.Sprint(cfg.Port), &imapclient.Options{})
	if err != nil {
		return fmt.Errorf("error connecting to IMAP server: %w", err)
	}
	defer client.Logout()

	if err := client.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		return fmt.Errorf("error logging in to the IMAP server: %w", err)
	}

	if _, err := client.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return fmt.Errorf("error selecting mailbox: %w", err)
	}

	// TODO: Set value from config.
	since := time.Now().Add(-12 * time.Hour)

	searchData, err := e.searchMessages(client, since)
	if err != nil {
		return fmt.Errorf("error searching messages: %w", err)
	}

	return e.fetchAndProcessMessages(client, searchData, e.Identifier())
}

// searchMessages searches for messages in the specified time range.
func (e *Email) searchMessages(client *imapclient.Client, since time.Time) (*imap.SearchData, error) {
	searchCMD := client.Search(&imap.SearchCriteria{
		Since: since,
	},
		&imap.SearchOptions{
			ReturnMin:   true,
			ReturnMax:   true,
			ReturnAll:   true,
			ReturnCount: true,
		},
	)
	return searchCMD.Wait()
}

// fetchAndProcessMessages fetches and processes messages based on the search results.
func (e *Email) fetchAndProcessMessages(client *imapclient.Client, searchData *imap.SearchData, inboxID int) error {
	seqSet := imap.SeqSet{}
	seqSet.AddRange(searchData.Min, searchData.Max)

	// Fetch only envelope.
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
	}

	fetchCmd := client.Fetch(seqSet, fetchOptions)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		for fetchItem := msg.Next(); fetchItem != nil; fetchItem = msg.Next() {
			if item, ok := fetchItem.(imapclient.FetchItemDataEnvelope); ok {
				if err := e.processEnvelope(client, item.Envelope, msg.SeqNum, inboxID); err != nil {
					e.lo.Error("error processing envelope", "error", err)
				}
			}
		}
	}

	return nil
}

// processEnvelope processes an email envelope.
func (e *Email) processEnvelope(client *imapclient.Client, env *imap.Envelope, seqNum uint32, inboxID int) error {
	if len(env.From) == 0 {
		e.lo.Warn("no sender received for email", "message_id", env.MessageID)
		return nil
	}

	exists, err := e.messageStore.MessageExists(env.MessageID)
	if err != nil {
		e.lo.Error("error checking if message exists", "message_id", env.MessageID)
		return err
	}

	if exists {
		e.lo.Debug("message already exists", "message_id", env.MessageID)
		return nil
	}

	e.lo.Debug("message does not exist", "message_id", env.MessageID)

	// Make contact.
	firstName, lastName := getContactName(env.From[0])
	var contact = umodels.User{
		InboxID:         inboxID,
		FirstName:       firstName,
		LastName:        lastName,
		SourceChannel:   null.NewString(e.Channel(), true),
		SourceChannelID: null.NewString(env.From[0].Addr(), true),
		Email:           null.NewString(env.From[0].Addr(), true),
		Type:            user.UserTypeContact,
	}

	incomingMsg := models.IncomingMessage{
		Message: models.Message{
			Channel:    e.Channel(),
			SenderType: conversation.SenderTypeContact,
			Type:       conversation.MessageIncoming,
			InboxID:    inboxID,
			Status:     conversation.MessageStatusReceived,
			Subject:    env.Subject,
			SourceID:   null.StringFrom(env.MessageID),
		},
		Contact: contact,
		InboxID: inboxID,
	}

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	}
	seqSet := imap.SeqSet{}
	seqSet.AddNum(seqNum)

	fullFetchCmd := client.Fetch(seqSet, fetchOptions)
	fullMsg := fullFetchCmd.Next()
	if fullMsg == nil {
		return nil
	}

	for fullFetchItem := fullMsg.Next(); fullFetchItem != nil; fullFetchItem = fullMsg.Next() {
		if fullItem, ok := fullFetchItem.(imapclient.FetchItemDataBodySection); ok {
			e.lo.Debug("fetching full message body", "message_id", env.MessageID)
			return e.processFullMessage(fullItem, incomingMsg)
		}
	}

	return nil
}

// processFullMessage processes the full email message.
func (e *Email) processFullMessage(item imapclient.FetchItemDataBodySection, incomingMsg models.IncomingMessage) error {
	envelope, err := enmime.ReadEnvelope(item.Literal)
	if err != nil {
		e.lo.Error("error parsing email envelope", "error", err, "envelope_errors", envelope.Errors)
		return fmt.Errorf("parsing email envelope: %w", err)
	}

	if len(envelope.HTML) > 0 {
		incomingMsg.Message.Content = envelope.HTML
		incomingMsg.Message.ContentType = conversation.ContentTypeHTML
	} else if len(envelope.Text) > 0 {
		incomingMsg.Message.Content = envelope.Text
		incomingMsg.Message.ContentType = conversation.ContentTypeText
	}

	inReplyTo := strings.ReplaceAll(strings.ReplaceAll(envelope.GetHeader("In-Reply-To"), "<", ""), ">", "")
	references := strings.Fields(envelope.GetHeader("References"))
	for i, ref := range references {
		references[i] = strings.Trim(strings.TrimSpace(ref), " <>")
	}

	incomingMsg.Message.InReplyTo = inReplyTo
	incomingMsg.Message.References = references

	for _, att := range envelope.Attachments {
		incomingMsg.Message.Attachments = append(incomingMsg.Message.Attachments, attachment.Attachment{
			Name:        att.FileName,
			Content:     att.Content,
			ContentType: att.ContentType,
			Size:        len(att.Content),
			Disposition: attachment.DispositionAttachment,
		})
	}
	for _, inline := range envelope.Inlines {
		incomingMsg.Message.Attachments = append(incomingMsg.Message.Attachments, attachment.Attachment{
			Name:        inline.FileName,
			Content:     inline.Content,
			ContentType: inline.ContentType,
			ContentID:   inline.ContentID,
			Size:        len(inline.Content),
			Disposition: attachment.DispositionInline,
		})
	}
	e.lo.Debug("enqueuing prepared incoming message for inserting in DB", "message_id", incomingMsg.Message.SourceID.String, "attachments", len(envelope.Attachments), "inlines", len(envelope.Inlines))
	if err := e.messageStore.EnqueueIncoming(incomingMsg); err != nil {
		return err
	}
	return nil
}

// getContactName extracts the contact's first and last name from the IMAP address.
func getContactName(imapAddr imap.Address) (string, string) {
	from := strings.TrimSpace(imapAddr.Name)
	names := strings.Fields(from)
	if len(names) == 0 {
		return imapAddr.Host, ""
	}
	if len(names) == 1 {
		return names[0], ""
	}
	return names[0], names[1]
}
