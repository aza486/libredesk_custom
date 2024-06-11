package email

import (
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
	"github.com/zerodha/logf"
)

// IMAP represents a wrapper around the IMAP client.
type IMAP struct {
	lo           *logf.Logger
	mailbox      string
	userName     string
	readInterval time.Duration
	cfg          IMAPConfig
}

func NewIMAP(cfg IMAPConfig) (*IMAP, error) {
	logger := logf.New(logf.Opts{
		EnableColor:          true,
		Level:                logf.DebugLevel,
		CallerSkipFrameCount: 3,
		TimestampFormat:      time.RFC3339Nano,
		EnableCaller:         true,
	})
	dur, err := time.ParseDuration(cfg.ReadInterval)

	if err != nil {
		return nil, err
	}

	return &IMAP{
		mailbox:      cfg.Mailbox,
		lo:           &logger,
		userName:     cfg.Username,
		readInterval: dur,
		cfg:          cfg,
	}, nil
}

func (i *IMAP) ReadIncomingMessages(inboxID int, incomingMsgQ chan<- models.IncomingMessage) error {
	var (
		since    = time.Now().Add(time.Hour * 6 * -1)
		tomorrow = time.Now().Add(time.Hour * 24)
		t        = time.NewTicker(i.readInterval)
		c        = &imapclient.Client{}
	)

	for range t.C {
		var err error

		c, err = imapclient.DialTLS(i.cfg.Host+":"+fmt.Sprint(i.cfg.Port), &imapclient.Options{})
		if err != nil {
			i.lo.Error("error connecting to IMAP server", "imap_username", i.userName, "error", err)
			continue
		}

		// Send the login command.
		cmd := c.Login(i.cfg.Username, i.cfg.Password)

		// Wait for the login command to complete.
		err = cmd.Wait()

		if err != nil {
			i.lo.Error("error logging in to the IMAP server", "imap_username", i.userName, "error", err)
			continue
		}

		// Select the Mailbox.
		selectCMD := c.Select(i.mailbox, &imap.SelectOptions{ReadOnly: true})

		_, err = selectCMD.Wait()

		if err != nil {
			i.lo.Error("error doing imap select", "error", err)
		}

		i.lo.Debug("fetching emails", "since", since, "to", tomorrow)

		searchCMD := c.Search(&imap.SearchCriteria{
			Since:  since,
			Before: tomorrow,
		},
			&imap.SearchOptions{
				ReturnMin:   true,
				ReturnMax:   true,
				ReturnAll:   true,
				ReturnCount: true,
			},
		)
		searchData, err := searchCMD.Wait()

		if err != nil {
			i.lo.Error("error executing IMAP search command", "imap_username", i.userName, "error", err)
			continue
		}

		i.lo.Debug("imap search stats", "count", searchData.Count)

		fetchOptions := &imap.FetchOptions{
			Envelope:    true,
			BodySection: []*imap.FetchItemBodySection{{}},
		}

		seqSet := imap.SeqSet{}
		seqSet.AddRange(searchData.Min, searchData.Max)

		fetchCmd := c.Fetch(seqSet, fetchOptions)

		for {
			msg := fetchCmd.Next()
			if msg == nil {
				break
			}

			// Incoming message has message & contact details.
			incomingMsg := models.IncomingMessage{
				Message: models.Message{
					Channel:    "email",
					SenderType: "contact",
					Type:       message.TypeIncoming,
					Meta:       "{}",
					InboxID:    inboxID,
					Status:     message.StatusReceived,
				},
				Contact: cmodels.Contact{
					Source: "email",
				},
				InboxID: inboxID,
			}
			fmt.Println("imap inbox id", inboxID)

			for {
				fetchItem := msg.Next()
				if fetchItem == nil {
					break
				}

				switch item := fetchItem.(type) {
				case imapclient.FetchItemDataEnvelope:
					env := item.Envelope
					if len(env.From) == 0 {
						i.lo.Debug("No sender found for email for message id", "source_id", env.MessageID)
						break
					}

					// Get contact name.
					email := env.From[0].Addr()
					env.From[0].Name = strings.TrimSpace(env.From[0].Name)
					names := strings.SplitN(env.From[0].Name, " ", 2)
					if len(names) == 1 && names[0] != "" {
						incomingMsg.Contact.FirstName, incomingMsg.Contact.LastName = names[0], ""
					} else if len(names) > 1 && names[0] != "" {
						incomingMsg.Contact.FirstName, incomingMsg.Contact.LastName = names[0], names[1]
					} else {
						incomingMsg.Contact.FirstName = env.From[0].Host
					}

					incomingMsg.Message.Subject = env.Subject
					incomingMsg.Message.SourceID = null.StringFrom(env.MessageID)
					// For contact the source will the unique identifier i.e email.
					incomingMsg.Contact.SourceID = email
					incomingMsg.Contact.Email = email
					incomingMsg.Contact.InboxID = inboxID
				case imapclient.FetchItemDataBodySection:
					envel, err := enmime.ReadEnvelope(item.Literal)
					if err != nil {
						i.lo.Error("parsing email envelope", "error", err)
						break
					}
					if len(envel.HTML) > 0 {
						incomingMsg.Message.Content = envel.HTML
						incomingMsg.Message.ContentType = message.ContentTypeHTML
					} else if len(envel.Text) > 0 {
						incomingMsg.Message.Content = envel.Text
						incomingMsg.Message.ContentType = message.ContentTypeText
					}

					// Set in reply to and references.
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
					incomingMsgQ <- incomingMsg
				default:
					continue
				}
			}
		}

		c.Logout().Wait()
	}
	return nil
}
