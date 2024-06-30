package email

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/smtp"
	"net/textproto"

	"github.com/abhinavxd/artemis/internal/message/models"
	"github.com/knadh/smtppool"
)

const (
	headerReturnPath  = "Return-Path"
	headerMessageID   = "Message-ID"
	headerReferences  = "References"
	headerInReplyTo   = "In-Reply-To"
	dispositionInline = "inline"
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

// Send sends an email.
func (e *Email) Send(m models.Message) error {
	// If there are more than one SMTP servers, send to a random one from the list.
	var (
		ln  = len(e.smtpPools)
		srv *smtppool.Pool
	)
	if ln > 1 {
		srv = e.smtpPools[rand.Intn(ln)]
	} else {
		srv = e.smtpPools[0]
	}

	// Are there attachments?
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

	// Attach SMTP level headercfg.
	for k, v := range e.headers {
		em.Headers.Set(k, v)
	}

	// Attach e-mail level headercfg.
	for k, v := range m.Headers {
		em.Headers.Set(k, v[0])
	}

	// Others.
	if m.InReplyTo != "" {
		em.Headers.Set(headerInReplyTo, "<"+m.InReplyTo+">")
	}
	references := ""
	for _, ref := range m.References {
		references += "<" + ref + "> "
	}
	em.Headers.Set(headerReferences, references)

	fmt.Printf("%+v EMAIL HEADERS -> headers", em.Headers)

	switch m.ContentType {
	case "plain":
		em.Text = []byte(m.Content)
	default:
		em.HTML = []byte(m.Content)
		if len(m.AltContent) > 0 {
			em.Text = []byte(m.AltContent)
		}
	}
	jsonData, err := json.MarshalIndent(em, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling to JSON:", err)
	}

	fmt.Println()
	fmt.Println()
	fmt.Println(string(jsonData))
	fmt.Println()
	fmt.Println()

	return srv.Send(em)
}
