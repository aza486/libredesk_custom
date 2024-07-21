package auth

import (
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

type Manager struct {
	lo *logf.Logger
}

type ConversationStore interface {
	GetAssigneedUserID(conversationID int) (int, error)
}

func New(db *sqlx.DB, lo *logf.Logger) (*Manager, error) {
	return &Manager{
		lo: lo,
	}, nil
}

func (e *Manager) HasPermission(userID int, perm string) (bool, error) {
	return true, nil
}
