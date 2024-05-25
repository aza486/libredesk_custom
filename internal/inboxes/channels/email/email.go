package email

import (
	"github.com/abhinavxd/artemis/internal/conversations/models"
	"github.com/abhinavxd/artemis/internal/inboxes"
	"github.com/knadh/smtppool"
)

// Compile time check to ensure that Email implements the inbox.Inbox interface.
var _ inboxes.Inbox = (*Email)(nil)

type Config struct {
	SMTP []SMTPConfig `json:"smtp"`
	IMAP []IMAPConfig `json:"imap"`
}

// SMTP represents an SMTP server's credentials with the smtppool options.
type SMTPConfig struct {
	UUID          string            `json:"uuid"`
	Enabled       bool              `json:"enabled"`
	Host          string            `json:"host"`
	HelloHostname string            `json:"hello_hostname"`
	Port          int               `json:"port"`
	AuthProtocol  string            `json:"auth_protocol"`
	Username      string            `json:"username"`
	Password      string            `json:"password,omitempty"`
	EmailHeaders  map[string]string `json:"email_headers"`
	MaxMsgRetries int               `json:"max_msg_retries"`
	IdleTimeout   string            `json:"idle_timeout"`
	WaitTimeout   string            `json:"wait_timeout"`
	TLSType       string            `json:"tls_type"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`

	// SMTP pool options.
	smtppool.Opt
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

// Opts holds the options for the Email channel.
type Opts struct {
	ID           int64
	EmailHeaders map[string]string
	Config       Config
}

// Email represents an email inbox with multiple SMTP servers and IMAP clients.
type Email struct {
	id           int64
	emailHeaders map[string]string
	smtpPool     []*smtppool.Pool
	imapClients  []*IMAP
}

// Returns a new instance of the Email inbox.
func New(opts Opts) (*Email, error) {
	pool, err := NewSmtpPool(opts.Config.SMTP)

	if err != nil {
		return nil, err
	}

	e := &Email{
		smtpPool:     pool,
		imapClients:  make([]*IMAP, 0, len(opts.Config.IMAP)),
		emailHeaders: opts.EmailHeaders,
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
func (e *Email) Identifier() int64 {
	return e.id
}

// Close closes the email inbox and releases all the resources.
func (e *Email) Close() error {
	// Close smtp pool.
	for _, p := range e.smtpPool {
		p.Close()
	}

	// Logout from the IMAP clients.
	for _, i := range e.imapClients {
		i.Client.Logout()
	}

	return nil
}

func (e *Email) Receive(msgChan chan models.IncomingMessage) error {
	for _, imap := range e.imapClients {
		imap.ReadIncomingMessages(e.Identifier(), msgChan)
	}
	return nil
}
