package email

import (
	"fmt"
	"math/rand"
	"net/textproto"

	"github.com/abhinavxd/artemis/internal/inbox/channel/email"
	"github.com/abhinavxd/artemis/internal/message/models"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
)

// Notifier handles email notifications.
type Notifier struct {
	lo               *logf.Logger
	from             string
	smtpPools        []*smtppool.Pool
	userStore        notifier.UserStore
	TemplateRenderer notifier.TemplateRenderer
}

type Opts struct {
	Lo        *logf.Logger
	FromEmail string
}

// New creates a new instance of email Notifier.
func New(smtpConfig []email.SMTPConfig, userStore notifier.UserStore, TemplateRenderer notifier.TemplateRenderer, opts Opts) (*Notifier, error) {
	pools, err := email.NewSmtpPool(smtpConfig)
	if err != nil {
		return nil, err
	}
	return &Notifier{
		lo:               opts.Lo,
		smtpPools:        pools,
		from:             opts.FromEmail,
		userStore:        userStore,
		TemplateRenderer: TemplateRenderer,
	}, nil
}

// SendMessage sends an email using the default template to multiple users.
func (e *Notifier) SendMessage(userUUIDs []string, subject, content string) error {
	var recipientEmails []string
	for i := 0; i < len(userUUIDs); i++ {
		userEmail, err := e.userStore.GetEmail(0, userUUIDs[i])
		if err != nil {
			e.lo.Error("error fetching user email for user uuid", "error", err)
			return err
		}
		recipientEmails = append(recipientEmails, userEmail)
	}

	// Render with default template.
	templateBody, templateSubject, err := e.TemplateRenderer.RenderDefault(map[string]string{
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

	err = e.Send(m)
	if err != nil {
		e.lo.Error("error sending email notification", "error", err)
		return err
	}
	return nil
}

func (e *Notifier) SendAssignedConversationNotification(userUUIDs []string, convUUID string) error {
	subject := "New conversation assigned to you"
	link := fmt.Sprintf("http://localhost:5173/conversations/%s", convUUID)
	content := fmt.Sprintf("A new conversation has been assigned to you. <br>Please review the details and take necessary action by following this link: %s", link)
	return e.SendMessage(userUUIDs, subject, content)
}

// Send sends an email message using one of the SMTP pools.
func (e *Notifier) Send(m models.Message) error {
	var (
		ln  = len(e.smtpPools)
		srv *smtppool.Pool
	)
	if ln > 1 {
		srv = e.smtpPools[rand.Intn(ln)]
	} else {
		srv = e.smtpPools[0]
	}

	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := smtppool.Attachment{
				Filename: f.Filename,
				Header:   f.Header,
				Content:  make([]byte, len(f.Content)),
			}
			copy(a.Content, f.Content)
			files = append(files, a)
		}
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
	return srv.Send(em)
}
