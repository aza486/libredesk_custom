package inboxes

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/artemis/internal/conversations"
	"github.com/abhinavxd/artemis/internal/conversations/models"
	"github.com/abhinavxd/artemis/internal/utils"

	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	// Embedded filesystem
	//go:embed queries.sql
	efs embed.FS
)

type InboxNotFound struct {
	ID int64
}

func (e *InboxNotFound) Error() string {
	return fmt.Sprintf("inbox not found: %d", e.ID)
}

// Closer provides a method for closing resources.
type Closer interface {
	Close() error
}

// Identifier provides a method for obtaining a unique identifier for the inbox.
type Identifier interface {
	Identifier() int64
}

// MessageHandler defines methods for handling message operations.
type MessageHandler interface {
	Receive(chan models.IncomingMessage) error
	Send(models.Message) error
}

// Inbox combines the operations of an inbox including its lifecycle, identification, and message handling.
type Inbox interface {
	Closer
	Identifier
	MessageHandler
}

// Opts contains the options for the manager.
type Opts struct {
	QueueSize    int
	Concurrency  int
	Conversation *conversations.Conversations
}

// Manager manages the inbox.
type Manager struct {
	queries      queries
	incomingMsgQ chan models.IncomingMessage
	concurrency  int
	inboxes      map[int64]Inbox
	lo           *logf.Logger
	conversation *conversations.Conversations
}

type queries struct {
	ActiveInboxes *sqlx.Stmt `query:"get-active-inboxes"`
}

// NewManager returns a new inbox manager.
func NewManager(lo *logf.Logger, db *sqlx.DB, opts Opts) *Manager {
	m := &Manager{
		lo:           lo,
		incomingMsgQ: make(chan models.IncomingMessage, opts.QueueSize),
		concurrency:  opts.Concurrency,
		// Map of inbox ID in the DB to the inbox.
		inboxes:      make(map[int64]Inbox),
		conversation: opts.Conversation,
	}

	// Scan the sql	file into the queries struct.
	if err := utils.ScanSQLFile("queries.sql", &m.queries, db, efs); err != nil {
		lo.Fatal("failed to scan queries", "error", err)
	}
	return m
}

// Register registers the inbox with the manager.
func (m *Manager) Register(i Inbox) {
	m.inboxes[i.Identifier()] = i
}

// GetInbox returns the inbox with the given id.
func (m *Manager) GetInbox(id int64) (Inbox, error) {
	i, ok := m.inboxes[id]
	if !ok {
		return nil, &InboxNotFound{ID: id}
	}
	return i, nil
}

// ActiveInboxes returns all the active inboxes from the DB.
func (m *Manager) ActiveInboxes() ([]Inbox, error) {
	var inboxes []Inbox
	if err := m.queries.ActiveInboxes.Select(&inboxes); err != nil {
		return nil, err
	}
	return inboxes, nil
}

// Run spawns a N goroutine workers to process incoming messages.
func (m *Manager) Run() {
	m.lo.Info(fmt.Sprintf("spawning %d workers to process incoming messages from inboxes.", m.concurrency))
	for range m.concurrency {
		go m.worker()
	}
}

func (m *Manager) Push(msg models.IncomingMessage) {
	m.incomingMsgQ <- msg
}

func (m *Manager) Receive(msg models.IncomingMessage) error {
	var inboxes, err = m.ActiveInboxes()
	if err != nil {
		return err
	}

	for _, inb := range inboxes {
		go inb.Receive(m.incomingMsgQ)
	}
	return nil
}

// worker is a blocking function that should be invoked with a goroutine,
// it received the incoming message and pushes it to the conversation manager.
func (m *Manager) worker() {
	for msg := range m.incomingMsgQ {
		fmt.Println(msg)
	}
}
