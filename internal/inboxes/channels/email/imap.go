package email

import (
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime"
	"github.com/zerodha/logf"
)

// IMAP represents a wrapper around the IMAP client.
type IMAP struct {
	Client       *imapclient.Client
	lo           *logf.Logger
	mailbox      string
	readInterval time.Duration
}

func NewIMAP(cfg IMAPConfig) (*IMAP, error) {
	client, err := imapclient.DialTLS(cfg.Host+":"+fmt.Sprint(cfg.Port), nil)
	if err != nil {
		return nil, err
	}

	logger := logf.New(logf.Opts{
		EnableColor:          true,
		Level:                logf.DebugLevel,
		CallerSkipFrameCount: 3,
		TimestampFormat:      time.RFC3339Nano,
		EnableCaller:         true,
	})

	// Send the login command
	cmd := client.Login(cfg.Username, cfg.Password)

	// Wait for the login command to complete
	err = cmd.Wait()

	if err != nil {
		return nil, err
	}

	dur, err := time.ParseDuration(cfg.ReadInterval)

	if err != nil {
		return nil, err
	}

	return &IMAP{
		Client:       client,
		mailbox:      cfg.Mailbox,
		lo:           &logger,
		readInterval: dur,
	}, nil
}

// TODO: Remove print statements, Figure out a way to handle imap login and other errors
func (i *IMAP) ReadIncomingMessages(inboxID int64, incomingMsgQ chan<- models.IncomingMessage) error {
	var (
		// Ticket to read messages at regular intervals
		t = time.NewTicker(i.readInterval)

		c = i.Client
	)

	// Select Mailbox
	cmd := c.Select(i.mailbox, &imap.SelectOptions{ReadOnly: true})

	mbox, err := cmd.Wait()

	if err != nil {
		return err
	}

	// Initialize uID, it's the upcoming UID.
	uID := imap.UID(mbox.UIDNext)

	uID = imap.UID(31503)

	// Start reading emails
	for range t.C {
		i.lo.Debug("received tick", "next_uid", uID)

		// Create a new sequence set of UID's.
		seqSet := imap.UIDSet{}

		// Read from the last UID
		// This is `latest_uid:*` in IMAP syntax
		seqSet.AddRange(uID, uID)

		fetchOptions := &imap.FetchOptions{
			Envelope:    true,
			BodySection: []*imap.FetchItemBodySection{{}},
		}

		// Use UIDFetch instead of Fetch
		// UIDFetch will return the UID of the message and not the sequence number
		// This is important because the sequence number can change when a new message is added/deleted from the mailbox
		fetchCmd := c.Fetch(seqSet, fetchOptions)

		// Create a new message
		message := models.IncomingMessage{
			Sender:  models.User{Type: models.UserTypeContact},
			Type:    models.MessageTypeIncoming,
			Source:  models.SourceEmail,
			InboxID: inboxID,
		}

		for {
			msg := fetchCmd.Next()
			if msg == nil {
				break
			}

			for {
				item := msg.Next()
				if item == nil {
					break
				}

				switch item := item.(type) {

				case imapclient.FetchItemDataEnvelope:
					if len(item.Envelope.From) == 0 {
						i.lo.Debug("No sender found for email for message id", "message_id", item.Envelope.MessageID)
						break
					}

					// Get the sender's name and email address from the envelope
					addr := item.Envelope.From[0].Addr()
					i.lo.Debug("incoming email address", "email", addr)
					i.lo.Debug("incoming email from name", "name", item.Envelope.From[0].Name)
					message.Sender.FirstName, message.Sender.LastName = splitName(item.Envelope.From[0].Name)
					message.Sender.Email = addr
					message.SourceID = item.Envelope.MessageID
					message.Subject = item.Envelope.Subject
				case imapclient.FetchItemDataBodySection:
					env, err := enmime.ReadEnvelope(item.Literal)
					if err != nil {
						i.lo.Error("parse email body", "error", err)
						break
					}

					if len(env.HTML) > 0 {
						message.Content = env.HTML
						message.ContentType = "text/html"
					} else if len(env.Text) > 0 {
						message.Content = env.Text
						message.ContentType = "text/plain"
					}

					// Add attachments to the message
					for _, j := range env.Attachments {
						message.Attachments = append(message.Attachments, models.Attachment{
							Name:     j.FileName,
							Header:   j.Header,
							Content:  j.Content,
							IsInline: false,
						})
					}

					for _, j := range env.Inlines {
						// If the content id is found in the html, then it's an inline attachment
						// and needs to be added to the message as an attachment.
						if strings.Contains(message.Content, j.ContentID) {
							message.Attachments = append(message.Attachments, models.Attachment{
								Name:     j.FileName,
								Header:   j.Header,
								Content:  j.Content,
								IsInline: true,
							})
						}
					}

					// The references header is used to track the conversation thread,
					// and has a space separated list of message ids.
					ref := env.GetHeader("References")
					if ref != "" {
						references := strings.Split(ref, " ")
						for _, r := range references {
							message.References = append(message.References, strings.TrimSpace(r))
						}
					}

					incomingMsgQ <- message

					// Increment UID.
					uID++

					// TODO: Commit offset?
				default:
					continue
				}
			}
		}
	}
	return nil
}

func splitName(name string) (string, string) {
	names := strings.Split(name, " ")
	if len(names) == 1 {
		return names[0], ""
	}
	return names[0], names[1]
}
