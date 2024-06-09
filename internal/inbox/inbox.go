package inbox

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/abhinavxd/artemis/internal/message/models"
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
	ID int
}

func (e *InboxNotFound) Error() string {
	return fmt.Sprintf("inbox not found: %d", e.ID)
}

// Closer provides function for closing a channel.
type Closer interface {
	Close() error
}

// Identifier provides a method for obtaining a unique identifier for the inbox.
type Identifier interface {
	Identifier() int
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
	FromAddress() string
	Channel() string
}

// Opts contains the options for the initializing the inbox manager.
type Opts struct {
	QueueSize   int
	Concurrency int
}

// Manager manages the inbox.
type Manager struct {
	queries      queries
	inboxes      map[int]Inbox
	incomingMsgQ chan models.IncomingMessage
	lo           *logf.Logger
}

// InboxRecord represents a inbox record in DB.
type InboxRecord struct {
	ID      int             `db:"id"`
	Name    string          `db:"name"`
	Channel string          `db:"channel"`
	Enabled string          `db:"enabled"`
	From    string          `db:"from"`
	Config  json.RawMessage `db:"config"`
}

// Prepared queries.
type queries struct {
	ActiveInboxes *sqlx.Stmt `query:"get-active-inboxes"`
}

// New returns a new inbox manager.
func New(lo *logf.Logger, db *sqlx.DB, incomingMsgQ chan models.IncomingMessage) (*Manager, error) {
	var q queries

	// Scan the sql	file into the queries struct.
	if err := utils.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}

	m := &Manager{
		lo:           lo,
		incomingMsgQ: incomingMsgQ,
		inboxes:      make(map[int]Inbox),
		queries:      q,
	}
	return m, nil
}

// Register registers the inbox with the manager.
func (m *Manager) Register(i Inbox) {
	m.inboxes[i.Identifier()] = i
}

// GetInbox returns the inbox with the given ID.
func (m *Manager) GetInbox(id int) (Inbox, error) {
	i, ok := m.inboxes[id]
	if !ok {
		return nil, &InboxNotFound{ID: id}
	}
	return i, nil
}

// GetActiveInboxes returns all the active inboxes from the DB.
func (m *Manager) GetActiveInboxes() ([]InboxRecord, error) {
	var inboxes []InboxRecord
	if err := m.queries.ActiveInboxes.Select(&inboxes); err != nil {
		m.lo.Error("fetching active inboxes", "error", err)
		return nil, err
	}
	return inboxes, nil
}

// Receive starts receiver for each inbox.
func (m *Manager) Receive() {
	for _, inb := range m.inboxes {
		go inb.Receive(m.incomingMsgQ)
	}
}
