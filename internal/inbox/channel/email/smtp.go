package email

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"net/textproto"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/knadh/smtppool"
)

const (
	headerReturnPath  = "Return-Path"
	headerMessageID   = "Message-ID"
	headerReferences  = "References"
	headerInReplyTo   = "In-Reply-To"
	dispositionInline = "inline"
)

// NewSmtpPool returns a smtppool
func NewSmtpPool(configs []SMTPConfig) ([]*smtppool.Pool, error) {
	pools := make([]*smtppool.Pool, 0, len(configs))

	for _, cfg := range configs {
		var auth smtp.Auth
		switch cfg.AuthProtocol {
		case "cram":
			auth = smtp.CRAMMD5Auth(cfg.Username, cfg.Password)
		case "plain":
			auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		case "login":
			auth = &smtppool.LoginAuth{Username: cfg.Username, Password: cfg.Password}
		case "", "none":
			// No authentication
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", cfg.AuthProtocol)
		}
		cfg.Opt.Auth = auth

		// TLS config
		if cfg.TLSType != "none" {
			cfg.TLSConfig = &tls.Config{}
			if cfg.TLSSkipVerify {
				cfg.TLSConfig.InsecureSkipVerify = cfg.TLSSkipVerify
			} else {
				cfg.TLSConfig.ServerName = cfg.Host
			}

			// SSL/TLS, not STARTTLS
			if cfg.TLSType == "TLS" {
				cfg.Opt.SSL = true
			}
		}

		pool, err := smtppool.New(cfg.Opt)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// Send sends an email using one of the configured SMTP servers.
func (e *Email) Send(m models.Message) error {
	// Select a random SMTP server if there are multiple
	var (
		serverCount = len(e.smtpPools)
		server      *smtppool.Pool
	)
	if serverCount > 1 {
		server = e.smtpPools[rand.Intn(serverCount)]
	} else {
		server = e.smtpPools[0]
	}

	// Prepare attachments if there are any
	var attachments []smtppool.Attachment
	if m.Attachments != nil {
		attachments = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, file := range m.Attachments {
			attachment := smtppool.Attachment{
				Filename: file.Name,
				Header:   file.Header,
				Content:  make([]byte, len(file.Content)),
			}
			copy(attachment.Content, file.Content)
			attachments = append(attachments, attachment)
		}
	}

	email := smtppool.Email{
		From:        m.From,
		To:          m.To,
		Cc:          m.CC,
		Bcc:         m.BCC,
		Subject:     m.Subject,
		Attachments: attachments,
		Headers:     textproto.MIMEHeader{},
	}

	// Attach SMTP level headers
	for key, value := range e.headers {
		email.Headers.Set(key, value)
	}

	// Attach email level headers
	for key, value := range m.Headers {
		email.Headers.Set(key, value[0])
	}

	// Set In-Reply-To header
	if m.InReplyTo != "" {
		email.Headers.Set(headerInReplyTo, "<"+m.InReplyTo+">")
		e.lo.Debug("In-Reply-To header set", "message_id", m.InReplyTo)
	}

	// Set message id header
	if m.SourceID.String != "" {
		email.Headers.Set(headerMessageID, fmt.Sprintf("<%s>", m.SourceID.String))
		e.lo.Debug("Message-ID header set", "message_id", m.SourceID.String)
	}

	// Set references header
	var references string
	for _, ref := range m.References {
		references += "<" + ref + "> "
	}
	e.lo.Debug("References header set", "references", references)
	email.Headers.Set(headerReferences, references)

	// Set email content
	switch m.ContentType {
	case "plain":
		email.Text = []byte(m.Content)
	default:
		email.HTML = []byte(m.Content)
		if len(m.AltContent) > 0 {
			email.Text = []byte(m.AltContent)
		}
	}
	return server.Send(email)
}
