// Package email is the email inbox with multiple SMTP servers and IMAP clients.
package email

import (
	"context"

	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
)

const (
	ChannelEmail = "email"
)

// Config holds the email inbox config with multiple smtp servers & imap clients.
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

// IMAPConfig holds imap client credentials & config.
type IMAPConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Mailbox      string `json:"mailbox"`
	ReadInterval string `json:"read_interval"`
}

// Email is the email inbox with multiple SMTP servers and IMAP clients.
type Email struct {
	id        int
	smtpPools []*smtppool.Pool
	imapCfg   []IMAPConfig
	headers   map[string]string
	lo        *logf.Logger
	from      string
	msgStore  inbox.MessageStore
}

// Opts holds the options required for the email inbox.
type Opts struct {
	ID      int
	Headers map[string]string
	Config  Config
	Lo      *logf.Logger
}

// New returns a new instance of the email inbox.
func New(store inbox.MessageStore, opts Opts) (*Email, error) {
	pools, err := NewSmtpPool(opts.Config.SMTP)
	if err != nil {
		return nil, err
	}

	e := &Email{
		id:        opts.ID,
		headers:   opts.Headers,
		from:      opts.Config.From,
		imapCfg:   opts.Config.IMAP,
		lo:        opts.Lo,
		smtpPools: pools,
		msgStore:  store,
	}

	return e, nil
}

// Identifier returns the unique identifier of the inbox which is the database id.
func (e *Email) Identifier() int {
	return e.id
}

// Close closes the email inbox and releases all the resources.
func (e *Email) Close() error {
	for _, p := range e.smtpPools {
		p.Close()
	}
	return nil
}

// Receive starts receiver for each IMAP client.
func (e *Email) Receive(ctx context.Context) error {
	for _, cfg := range e.imapCfg {
		e.ReadIncomingMessages(ctx, cfg)
	}
	return nil
}

// FromAddress returns the from address for this inbox.
func (e *Email) FromAddress() string {
	return e.from
}

// FromAddress returns the channel name for this inbox.
func (e *Email) Channel() string {
	return ChannelEmail
}
