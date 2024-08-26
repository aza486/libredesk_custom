// Package email sends out emails.
package email

import (
	"math/rand"
	"net/textproto"

	"github.com/abhinavxd/artemis/internal/attachment"
	"github.com/abhinavxd/artemis/internal/inbox/channel/email"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
)

// Email implements the MessageSender interface for sending emails.
type Email struct {
	lo               *logf.Logger
	from             string
	smtpPools        []*smtppool.Pool
	userStore        notifier.UserStore
	templateRenderer notifier.TemplateRenderer
}

// Opts contains options for creating a new Email sender.
type Opts struct {
	Lo        *logf.Logger
	FromEmail string
}

// New initializes a new Email sender.
func New(smtpConfig []email.SMTPConfig, userStore notifier.UserStore, templateRenderer notifier.TemplateRenderer, opts Opts) (*Email, error) {
	pools, err := email.NewSmtpPool(smtpConfig)
	if err != nil {
		return nil, err
	}
	return &Email{
		lo:               opts.Lo,
		smtpPools:        pools,
		from:             opts.FromEmail,
		userStore:        userStore,
		templateRenderer: templateRenderer,
	}, nil
}

// Send sends a notification message via email.
func (e *Email) Send(msg notifier.NotificationMessage) error {
	recipientEmails, err := e.getUserEmails(msg.UserIDs)
	if err != nil {
		return err
	}

	templateBody, err := e.templateRenderer.RenderDefault(map[string]string{
		"Content": msg.Content,
	})
	if err != nil {
		return err
	}

	emailMessage := e.prepareEmail(msg.Subject, templateBody, recipientEmails, msg)

	return e.send(emailMessage)
}

// Name returns the name of the provider.
func (e *Email) Name() string {
	return notifier.ProviderEmail
}

// getUserEmails fetches email addresses for specified user IDs.
func (e *Email) getUserEmails(userIDs []int) ([]string, error) {
	var recipientEmails []string
	for _, userID := range userIDs {
		userEmail, err := e.userStore.GetEmail(userID)
		if err != nil {
			e.lo.Error("error fetching user email", "error", err)
			continue
		}
		recipientEmails = append(recipientEmails, userEmail)
	}
	return recipientEmails, nil
}

// send sends an email message.
func (e *Email) send(em smtppool.Email) error {
	srv := e.selectSmtpPool()
	return srv.Send(em)
}

// selectSmtpPool selects a random SMTP pool if multiple are available.
func (e *Email) selectSmtpPool() *smtppool.Pool {
	if len(e.smtpPools) > 1 {
		return e.smtpPools[rand.Intn(len(e.smtpPools))]
	}
	return e.smtpPools[0]
}

// prepareEmail prepares the email message with attachments and headers.
func (e *Email) prepareEmail(subject, content string, recipients []string, msg notifier.NotificationMessage) smtppool.Email {
	var files []smtppool.Attachment
	if len(msg.Attachments) > 0 {
		files = e.prepareAttachments(msg.Attachments)
	}

	em := smtppool.Email{
		From:        e.from,
		To:          recipients,
		Subject:     subject,
		Attachments: files,
		Headers:     textproto.MIMEHeader{},
	}

	// Set content based on provided type
	switch msg.ContentType {
	case "plain":
		em.Text = []byte(content)
	default:
		em.HTML = []byte(content)
		if len(msg.AltContent) > 0 {
			em.Text = []byte(msg.AltContent)
		}
	}

	// Set any additional headers
	for headerKey, headerValue := range msg.Headers {
		em.Headers[headerKey] = headerValue
	}

	return em
}

// prepareAttachments prepares email attachments.
func (e *Email) prepareAttachments(attachments []attachment.Attachment) []smtppool.Attachment {
	files := make([]smtppool.Attachment, len(attachments))
	for i, f := range attachments {
		files[i] = smtppool.Attachment{
			Filename: f.Name,
			Header:   f.Header,
			Content:  append([]byte(nil), f.Content...),
		}
	}
	return files
}
