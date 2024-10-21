// Package inbox provides functionality to manage inboxes in the system.
package inbox

import (
	"context"
	"embed"
	"errors"
	"sync"

	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	imodels "github.com/abhinavxd/artemis/internal/inbox/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

const (
	ChannelEmail = "email"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS

	// ErrInboxNotFound is returned when an inbox is not found.
	ErrInboxNotFound = errors.New("inbox not found")
)

// Closer provides a function for closing an inbox.
type Closer interface {
	Close() error
}

// Identifier provides a method for obtaining a unique identifier for the inbox.
type Identifier interface {
	Identifier() int
}

// MessageHandler defines methods for handling message operations.
type MessageHandler interface {
	Receive(context.Context) error
	Send(models.Message) error
}

// Inbox combines the operations of an inbox including its lifecycle, identification, and message handling.
type Inbox interface {
	Closer
	Identifier
	MessageHandler
	FromAddress() string
	Channel() string
}

// MessageStore defines methods for storing and processing messages.
type MessageStore interface {
	MessageExists(string) (bool, error)
	EnqueueIncoming(models.IncomingMessage) error
}

// Opts contains the options for initializing the inbox manager.
type Opts struct {
	QueueSize   int
	Concurrency int
}

// Manager manages the inboxes.
type Manager struct {
	queries queries
	inboxes map[int]Inbox
	lo      *logf.Logger
	wg      *sync.WaitGroup
}

// Prepared queries.
type queries struct {
	GetByID     *sqlx.Stmt `query:"get-by-id"`
	GetActive   *sqlx.Stmt `query:"get-active-inboxes"`
	GetAll      *sqlx.Stmt `query:"get-all-inboxes"`
	Update      *sqlx.Stmt `query:"update"`
	Toggle      *sqlx.Stmt `query:"toggle"`
	SoftDelete  *sqlx.Stmt `query:"soft-delete"`
	InsertInbox *sqlx.Stmt `query:"insert-inbox"`
}

// New returns a new inbox manager.
func New(lo *logf.Logger, db *sqlx.DB) (*Manager, error) {
	var q queries

	// Scan the SQL file into the queries struct.
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	m := &Manager{
		lo:      lo,
		inboxes: make(map[int]Inbox),
		queries: q,
		wg:      &sync.WaitGroup{},
	}
	return m, nil
}

// Register registers the inbox with the manager.
func (m *Manager) Register(i Inbox) {
	m.inboxes[i.Identifier()] = i
}

// Get returns the inbox with the given ID.
func (m *Manager) Get(id int) (Inbox, error) {
	i, ok := m.inboxes[id]
	if !ok {
		return nil, ErrInboxNotFound
	}
	return i, nil
}

// GetByID returns an inbox from the DB by the given ID.
func (m *Manager) GetByID(id int) (imodels.Inbox, error) {
	var inbox imodels.Inbox
	if err := m.queries.GetByID.Get(&inbox, id); err != nil {
		m.lo.Error("fetching inbox by ID", "error", err)
		return inbox, err
	}
	return inbox, nil
}

// GetActive returns all active inboxes from the DB.
func (m *Manager) GetActive() ([]imodels.Inbox, error) {
	var inboxes []imodels.Inbox
	if err := m.queries.GetActive.Select(&inboxes); err != nil {
		m.lo.Error("fetching active inboxes", "error", err)
		return nil, err
	}
	return inboxes, nil
}

// GetAll returns all inboxes from the DB.
func (m *Manager) GetAll() ([]imodels.Inbox, error) {
	var inboxes = make([]imodels.Inbox, 0)
	if err := m.queries.GetAll.Select(&inboxes); err != nil {
		m.lo.Error("error fetching inboxes", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching inboxes", nil)
	}
	return inboxes, nil
}

// Create creates an inbox in the DB.
func (m *Manager) Create(inbox imodels.Inbox) error {
	if _, err := m.queries.InsertInbox.Exec(inbox.Channel, inbox.Config, inbox.Name, inbox.From, nil); err != nil {
		m.lo.Error("error creating inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating inbox", nil)
	}
	return nil
}

// Update updates an inbox in the DB.
func (m *Manager) Update(id int, inbox imodels.Inbox) error {
	if _, err := m.queries.Update.Exec(id, inbox.Channel, inbox.Config, inbox.Name, inbox.From); err != nil {
		m.lo.Error("error updating inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating inbox", nil)
	}
	return nil
}

// Toggle toggles the status of an inbox in the DB.
func (m *Manager) Toggle(id int) error {
	if _, err := m.queries.Toggle.Exec(id); err != nil {
		m.lo.Error("error toggling inbox", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error toggling inbox", nil)
	}
	return nil
}

// Delete soft deletes an inbox in the DB.
func (m *Manager) Delete(id int) error {
	if _, err := m.queries.SoftDelete.Exec(id); err != nil {
		m.lo.Error("error deleting inbox", "error", err)
		return err
	}
	return nil
}

// Receive starts the receiver for each inbox.
func (m *Manager) Receive(ctx context.Context) error {
	for _, inb := range m.inboxes {
		m.wg.Add(1)
		go func(inbox Inbox) {
			defer m.wg.Done()
			if err := inbox.Receive(ctx); err != nil {
				m.lo.Error("error starting inbox receiver", "error", err)
			}
		}(inb)
	}
	m.wg.Wait()
	return nil
}

// Close closes all inboxes.
func (m *Manager) Close() {
	for _, inb := range m.inboxes {
		inb.Close()
	}
	m.wg.Wait()
}
