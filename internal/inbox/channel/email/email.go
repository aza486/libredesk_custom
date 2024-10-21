// Package email provides functionality for an email inbox with multiple SMTP servers and IMAP clients.
package email

import (
	"context"
	"sync"

	"github.com/abhinavxd/artemis/internal/inbox"
	"github.com/knadh/smtppool"
	"github.com/zerodha/logf"
)

const (
	ChannelEmail = "email"
)

// Config holds the email inbox configuration with multiple SMTP servers and IMAP clients.
type Config struct {
	SMTP []SMTPConfig `json:"smtp"`
	IMAP []IMAPConfig `json:"imap"`
	From string       `json:"from"`
}

// SMTPConfig represents an SMTP server's credentials with the smtppool options.
type SMTPConfig struct {
	Username      string            `json:"username"`
	Password      string            `json:"password"`
	AuthProtocol  string            `json:"auth_protocol"`
	TLSType       string            `json:"tls_type"`
	TLSSkipVerify bool              `json:"tls_skip_verify"`
	EmailHeaders  map[string]string `json:"email_headers"`
	smtppool.Opt  `json:",squash"`  // SMTP pool options.
}

// IMAPConfig holds IMAP client credentials and configuration.
type IMAPConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Mailbox      string `json:"mailbox"`
	ReadInterval string `json:"read_interval"`
}

// Email represents the email inbox with multiple SMTP servers and IMAP clients.
type Email struct {
	id           int
	smtpPools    []*smtppool.Pool
	imapCfg      []IMAPConfig
	headers      map[string]string
	lo           *logf.Logger
	from         string
	messageStore inbox.MessageStore
	wg           sync.WaitGroup
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
		id:           opts.ID,
		headers:      opts.Headers,
		from:         opts.Config.From,
		imapCfg:      opts.Config.IMAP,
		lo:           opts.Lo,
		smtpPools:    pools,
		messageStore: store,
	}
	return e, nil
}

// Identifier returns the unique identifier of the inbox which is the database ID.
func (e *Email) Identifier() int {
	return e.id
}

// Receive starts reading incoming messages for each IMAP client.
func (e *Email) Receive(ctx context.Context) error {
	for _, cfg := range e.imapCfg {
		e.wg.Add(1)
		go func(cfg IMAPConfig) {
			defer e.wg.Done()
			if err := e.ReadIncomingMessages(ctx, cfg); err != nil {
				e.lo.Error("error reading incoming messages", "error", err)
			}
		}(cfg)
	}
	e.wg.Wait()
	return nil
}

// Close cloes email channel by closing the smtp pool
func (e *Email) Close() error {
	return e.closeSMTPPool()
}

// FromAddress returns the from address for this inbox.
func (e *Email) FromAddress() string {
	return e.from
}

// Channel returns the channel name for this inbox.
func (e *Email) Channel() string {
	return ChannelEmail
}

// closeSMTPPool closes the smtp pool.
func (e *Email) closeSMTPPool() error {
	for _, p := range e.smtpPools {
		p.Close()
	}
	return nil
}
