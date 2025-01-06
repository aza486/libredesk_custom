package user

import (
	"fmt"
	"strings"

	"github.com/abhinavxd/artemis/internal/user/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

// CreateContact creates a new contact user.
func (u *Manager) CreateContact(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("generating password", "error", err)
		return fmt.Errorf("generating password: %w", err)
	}

	// Normalize email address.
	user.Email = null.NewString(strings.ToLower(user.Email.String), user.Email.Valid)

	if err := u.q.InsertContact.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, pq.Array(user.Roles), user.InboxID, user.SourceChannelID).Scan(&user.ID, &user.ContactChannelID); err != nil {
		u.lo.Error("creating user", "error", err)
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}
