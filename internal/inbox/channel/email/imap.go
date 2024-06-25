package email

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/artemis/internal/attachment"
	amodels "github.com/abhinavxd/artemis/internal/attachment/models"
	cmodels "github.com/abhinavxd/artemis/internal/contact/models"
	"github.com/abhinavxd/artemis/internal/message"
	"github.com/abhinavxd/artemis/internal/message/models"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime"
	"github.com/volatiletech/null/v9"
)

const (
	DefaultReadInterval = time.Duration(5 * time.Minute)
)

func (e *Email) ReadIncomingMessages(ctx context.Context, cfg IMAPConfig) error {
	dur, err := time.ParseDuration(cfg.ReadInterval)
	if err != nil {
		e.lo.Warn("could not IMAP read interval, using the default read interval of 5 minutes.", "error", err)
		dur = DefaultReadInterval
	}

	readTicker := time.NewTicker(dur)
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

func (e *Email) processMailbox(cfg IMAPConfig) error {
	c, err := imapclient.DialTLS(cfg.Host+":"+fmt.Sprint(cfg.Port), &imapclient.Options{})
	if err != nil {
		return fmt.Errorf("error connecting to IMAP server: %w", err)
	}
	defer c.Logout()

	if err := c.Login(cfg.Username, cfg.Password).Wait(); err != nil {
		return fmt.Errorf("error logging in to the IMAP server: %w", err)
	}

	if _, err := c.Select(cfg.Mailbox, &imap.SelectOptions{ReadOnly: true}).Wait(); err != nil {
		return fmt.Errorf("error selecting mailbox: %w", err)
	}

	since := time.Now().Add(time.Hour * 12 * -1)
	before := time.Now().Add(time.Hour * 24)

	searchData, err := e.searchMessages(c, since, before)
	if err != nil {
		return fmt.Errorf("error searching messages: %w", err)
	}

	return e.fetchAndProcessMessages(c, searchData, e.Identifier())
}

func (e *Email) searchMessages(c *imapclient.Client, since, before time.Time) (*imap.SearchData, error) {
	searchCMD := c.Search(&imap.SearchCriteria{
		Since:  since,
		Before: before,
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

func (e *Email) fetchAndProcessMessages(c *imapclient.Client, searchData *imap.SearchData, inboxID int) error {
	seqSet := imap.SeqSet{}
	seqSet.AddRange(searchData.Min, searchData.Max)

	// Fetch only envelope.
	fetchOptions := &imap.FetchOptions{
		Envelope: true,
	}

	fetchCmd := c.Fetch(seqSet, fetchOptions)

	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}

		for fetchItem := msg.Next(); fetchItem != nil; fetchItem = msg.Next() {
			if item, ok := fetchItem.(imapclient.FetchItemDataEnvelope); ok {
				if err := e.processEnvelope(c, item.Envelope, msg.SeqNum, inboxID); err != nil {
					e.lo.Error("error processing envelope", "error", err)
				}
			}
		}
	}

	return nil
}

func (e *Email) processEnvelope(c *imapclient.Client, env *imap.Envelope, seqNum uint32, inboxID int) error {
	if len(env.From) == 0 {
		e.lo.Debug("no sender found for email", "message_id", env.MessageID)
		return nil
	}

	exists, err := e.msgStore.MessageExists(env.MessageID)
	if exists || err != nil {
		e.lo.Debug("email message already exists, skipping", "message_id", env.MessageID)
		return nil
	}

	incomingMsg := models.IncomingMessage{
		Message: models.Message{
			Channel:    e.Channel(),
			SenderType: message.SenderTypeContact,
			Type:       message.TypeIncoming,
			Meta:       "{}",
			InboxID:    int(inboxID),
			Status:     message.StatusReceived,
			Subject:    env.Subject,
			SourceID:   null.StringFrom(env.MessageID),
		},
		Contact: cmodels.Contact{
			Source:   e.Channel(),
			SourceID: env.From[0].Addr(),
			Email:    env.From[0].Addr(),
			InboxID:  int(inboxID),
		},
		InboxID: int(inboxID),
	}
	incomingMsg.Contact.FirstName, incomingMsg.Contact.LastName = getContactName(env.From[0])

	fetchOptions := &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{{}},
	}
	seqSet := imap.SeqSet{}
	seqSet.AddNum(seqNum)

	fullFetchCmd := c.Fetch(seqSet, fetchOptions)
	fullMsg := fullFetchCmd.Next()
	if fullMsg == nil {
		return nil
	}

	for fullFetchItem := fullMsg.Next(); fullFetchItem != nil; fullFetchItem = fullMsg.Next() {
		if fullItem, ok := fullFetchItem.(imapclient.FetchItemDataBodySection); ok {
			return e.processFullMessage(fullItem, &incomingMsg)
		}
	}

	return nil
}

func (e *Email) processFullMessage(item imapclient.FetchItemDataBodySection, incomingMsg *models.IncomingMessage) error {
	envel, err := enmime.ReadEnvelope(item.Literal)
	if err != nil {
		return fmt.Errorf("error parsing email envelope: %w", err)
	}

	if len(envel.HTML) > 0 {
		incomingMsg.Message.Content = envel.HTML
		incomingMsg.Message.ContentType = message.ContentTypeHTML
	} else if len(envel.Text) > 0 {
		incomingMsg.Message.Content = envel.Text
		incomingMsg.Message.ContentType = message.ContentTypeText
	}

	inReplyTo := strings.ReplaceAll(strings.ReplaceAll(envel.GetHeader("In-Reply-To"), "<", ""), ">", "")
	references := strings.Fields(envel.GetHeader("References"))
	for i, ref := range references {
		references[i] = strings.Trim(strings.TrimSpace(ref), " <>")
	}

	incomingMsg.Message.InReplyTo = inReplyTo
	incomingMsg.Message.References = references

	for _, j := range envel.Attachments {
		incomingMsg.Message.Attachments = append(incomingMsg.Message.Attachments, amodels.Attachment{
			Filename:           j.FileName,
			Header:             j.Header,
			Content:            j.Content,
			ContentType:        j.ContentType,
			ContentID:          j.ContentID,
			ContentDisposition: attachment.DispositionAttachment,
			Size:               strconv.Itoa(len(j.Content)),
		})
	}

	for _, j := range envel.Inlines {
		incomingMsg.Message.Attachments = append(incomingMsg.Message.Attachments, amodels.Attachment{
			Filename:           j.FileName,
			Header:             j.Header,
			Content:            j.Content,
			ContentType:        j.ContentType,
			ContentID:          j.ContentID,
			ContentDisposition: attachment.DispositionInline,
			Size:               strconv.Itoa(len(j.Content)),
		})
	}

	if err := e.msgStore.ProcessMessage(*incomingMsg); err != nil {
		return fmt.Errorf("error processing message: %w", err)
	}

	return nil
}

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
