package email

import (
	"fmt"
	"math/rand"
	"net/textproto"

	amodels "github.com/abhinavxd/artemis/internal/attachment/models"
	"github.com/abhinavxd/artemis/internal/inbox/channel/email"
	"github.com/abhinavxd/artemis/internal/message/models"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
)

// Email implements the Notifier interface for sending emails.
type Email struct {
	lo               *logf.Logger
	from             string
	smtpPools        []*smtppool.Pool
	userStore        notifier.UserStore
	templateRenderer notifier.TemplateRenderer
}

// Opts contains options for creating a new Email notifier.
type Opts struct {
	Lo        *logf.Logger
	FromEmail string
}

// New creates a new instance of the Email notifier.
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

// SendMessage sends an email using the default template to multiple users.
func (e *Email) SendMessage(userIDs []int, subject, content string) error {
	recipientEmails, err := e.getUserEmails(userIDs)
	if err != nil {
		return err
	}

	templateBody, templateSubject, err := e.templateRenderer.RenderDefault(map[string]string{
		"Content": content,
	})
	if err != nil {
		return err
	}

	if subject == "" {
		subject = templateSubject
	}

	m := models.Message{
		Subject: subject,
		Content: templateBody,
		From:    e.from,
		To:      recipientEmails,
	}

	return e.send(m)
}

// SendAssignedConversationNotification sends a email notification for an assigned conversation to the passed user.
func (e *Email) SendAssignedConversationNotification(userIDs []int, convUUID string) error {
	subject := "New conversation assigned to you"
	link := fmt.Sprintf("http://localhost:5173/conversations/%s", convUUID)
	content := fmt.Sprintf("A new conversation has been assigned to you. <br>Please review the details and take necessary action by following this link: %s", link)
	return e.SendMessage(userIDs, subject, content)
}

// getUserEmails fetches the email addresses of the specified user IDs.
func (e *Email) getUserEmails(userIDs []int) ([]string, error) {
	var recipientEmails []string
	for _, userID := range userIDs {
		userEmail, err := e.userStore.GetEmail(userID, "")
		if err != nil {
			e.lo.Error("error fetching user email", "error", err)
			return nil, err
		}
		recipientEmails = append(recipientEmails, userEmail)
	}
	return recipientEmails, nil
}

// send sends an email message.
func (e *Email) send(m models.Message) error {
	srv := e.selectSmtpPool()
	em := e.prepareEmail(m)
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
func (e *Email) prepareEmail(m models.Message) smtppool.Email {
	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = e.prepareAttachments(m.Attachments)
	}

	em := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Subject:     m.Subject,
		Attachments: files,
		Headers:     textproto.MIMEHeader{},
	}

	for k, v := range m.Headers {
		em.Headers.Set(k, v[0])
	}

	switch m.ContentType {
	case "plain":
		em.Text = []byte(m.Content)
	default:
		em.HTML = []byte(m.Content)
		if len(m.AltContent) > 0 {
			em.Text = []byte(m.AltContent)
		}
	}
	return em
}

// prepareAttachments prepares email attachments.
func (e *Email) prepareAttachments(attachments []amodels.Attachment) []smtppool.Attachment {
	files := make([]smtppool.Attachment, 0, len(attachments))
	for _, f := range attachments {
		a := smtppool.Attachment{
			Filename: f.Filename,
			Header:   f.Header,
			Content:  make([]byte, len(f.Content)),
		}
		copy(a.Content, f.Content)
		files = append(files, a)
	}
	return files
}
