// Package search provides search functionality.
package search

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager is the search manager
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts contains the options for creating a new search manager
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains all the prepared queries
type queries struct {
	SearchConversationsByRefNum       *sqlx.Stmt `query:"search-conversations-by-reference-number"`
	SearchConversationsByContactEmail *sqlx.Stmt `query:"search-conversations-by-contact-email"`
	SearchMessages                    *sqlx.Stmt `query:"search-messages"`
	SearchContacts                    *sqlx.Stmt `query:"search-contacts"`
}

// New creates a new search manager
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo}, nil
}

// Conversations searches conversations based on the query
func (s *Manager) Conversations(query string) ([]models.Conversation, error) {
	var refNumResults = make([]models.Conversation, 0)
	if err := s.q.SearchConversationsByRefNum.Select(&refNumResults, query); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error searching conversations", nil)
	}

	var emailResults = make([]models.Conversation, 0)
	if err := s.q.SearchConversationsByContactEmail.Select(&emailResults, query); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error searching conversations", nil)
	}
	return append(refNumResults, emailResults...), nil
}

// Messages searches messages based on the query
func (s *Manager) Messages(query string) ([]models.Message, error) {
	var results = make([]models.Message, 0)
	if err := s.q.SearchMessages.Select(&results, query); err != nil {
		s.lo.Error("error searching messages", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error searching messages", nil)
	}
	return results, nil
}

// Contacts searches contacts based on the query
func (s *Manager) Contacts(query string) ([]models.Contact, error) {
	var results = make([]models.Contact, 0)
	if err := s.q.SearchContacts.Select(&results, query); err != nil {
		s.lo.Error("error searching contacts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error searching contacts", nil)
	}
	return results, nil
}
