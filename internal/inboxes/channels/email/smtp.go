package email

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/smtp"
	"net/textproto"

	"github.com/abhinavxd/artemis/internal/conversations/models"
	"github.com/knadh/smtppool"
)

const (
	hdrReturnPath = "Return-Path"
)

// New returns an SMTP e-mail channels from the given SMTP server configcfg.
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
		default:
			return nil, fmt.Errorf("unknown SMTP auth type '%s'", cfg.AuthProtocol)
		}
		cfg.Opt.Auth = auth

		// TLS config.
		if cfg.TLSType != "none" {
			cfg.TLSConfig = &tls.Config{}
			if cfg.TLSSkipVerify {
				cfg.TLSConfig.InsecureSkipVerify = cfg.TLSSkipVerify
			} else {
				cfg.TLSConfig.ServerName = cfg.Host
			}

			// SSL/TLS, not STARTTLcfg.
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

// Send pushes a message to the server.
func (e *Email) Send(m models.IncomingMessage) error {
	// If there are more than one SMTP servers, send to a random
	// one from the list.
	var (
		ln  = len(e.SMTPPool)
		srv *smtppool.Pool
	)
	if ln > 1 {
		srv = e.SMTPPool[rand.Intn(ln)]
	} else {
		srv = e.SMTPPool[0]
	}

	// Are there attachments?
	var files []smtppool.Attachment
	if m.Attachments != nil {
		files = make([]smtppool.Attachment, 0, len(m.Attachments))
		for _, f := range m.Attachments {
			a := smtppool.Attachment{
				Filename: f.Name,
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
	}

	em.Headers = textproto.MIMEHeader{}

	// Attach SMTP level headercfg.
	for k, v := range e.EmailHeaders {
		em.Headers.Set(k, v)
	}

	// Attach e-mail level headercfg.
	for k, v := range m.Headers {
		em.Headers.Set(k, v[0])
	}

	// If the `Return-Path` header is set, it should be set as the
	// the SMTP envelope sender (via the Sender field of the email struct).
	if sender := em.Headers.Get(hdrReturnPath); sender != "" {
		em.Sender = sender
		em.Headers.Del(hdrReturnPath)
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
