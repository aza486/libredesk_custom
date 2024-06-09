package email

import (
	"github.com/abhinavxd/artemis/internal/message/models"
	"github.com/knadh/smtppool"
)

// Config holds the email channel config.
type Config struct {
	SMTP []SMTPConfig `json:"smtp"`
	IMAP []IMAPConfig `json:"imap"`
	From string
}

// SMTP represents an SMTP server's credentials with the smtppool options.
type SMTPConfig struct {
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	TLSType       string            `json:"tls_type"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`

	// SMTP pool options.
	smtppool.Opt `json:",squash"`
}

// IMAP holds imap credentials.
type IMAPConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Mailbox      string `json:"mailbox"`
	ReadInterval string `json:"read_interval"`
}

// Email is the email channel with multiple SMTP servers and IMAP clients.
type Email struct {
	id          int
	smtpPools   []*smtppool.Pool
	imapClients []*IMAP
	headers     map[string]string
	from        string
}

// Opts holds the options requierd.
type Opts struct {
	ID      int
	Headers map[string]string
	Config  Config
}

// Returns a new instance of the Email inbox.
func New(opts Opts) (*Email, error) {
	pools, err := NewSmtpPool(opts.Config.SMTP)

	if err != nil {
		return nil, err
	}

	e := &Email{
		smtpPools:   pools,
		imapClients: make([]*IMAP, 0, len(opts.Config.IMAP)),
		headers:     opts.Headers,
		from:        opts.Config.From,
	}

	// Initialize the IMAP clients.
	for _, im := range opts.Config.IMAP {
		imapClient, err := NewIMAP(im)
		if err != nil {
			return nil, err
		}
		// Append the IMAP client to the list of IMAP clients.
		e.imapClients = append(e.imapClients, imapClient)
	}

	e.id = opts.ID

	return e, nil
}

// ID returns the unique identifier of the inbox.
func (e *Email) Identifier() int {
	return e.id
}

// Close closes the email inbox and releases all the resources.
func (e *Email) Close() error {
	// Close smtp pool.
	for _, p := range e.smtpPools {
		p.Close()
	}
	return nil
}

func (e *Email) Receive(msgChan chan models.IncomingMessage) error {
	for _, imap := range e.imapClients {
		imap.ReadIncomingMessages(e.Identifier(), msgChan)
	}
	return nil
}

func (e *Email) FromAddress() string {
	return e.from
}

func (e *Email) Channel() string {
	return "email"
}
