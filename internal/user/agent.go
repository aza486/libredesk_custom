package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"golang.org/x/crypto/bcrypt"
)

// MonitorAgentAvailability continuously checks for user activity and sets them offline if inactive for more than 5 minutes.
func (u *Manager) MonitorAgentAvailability(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			u.markInactiveAgentsOffline()
		case <-ctx.Done():
			return
		}
	}
}

// GetAgent retrieves an agent by ID.
func (u *Manager) GetAgent(id int, email string) (models.User, error) {
	return u.Get(id, email, models.UserTypeAgent)
}

// GetAgentsCompact returns a compact list of users with limited fields.
func (u *Manager) GetAgentsCompact() ([]models.User, error) {
	var users = make([]models.User, 0)
	if err := u.q.GetAgentsCompact.Select(&users); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return users, nil
		}
		u.lo.Error("error fetching users from db", "error", err)
		return users, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", u.i18n.P("globals.terms.user")), nil)
	}
	return users, nil
}

// CreateAgent creates a new agent user.
func (u *Manager) CreateAgent(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil)
	}
	user.Email = null.NewString(strings.TrimSpace(strings.ToLower(user.Email.String)), user.Email.Valid)
	if err := u.q.InsertAgent.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, pq.Array(user.Roles)).Scan(&user.ID); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return envelope.NewError(envelope.GeneralError, u.i18n.T("user.sameEmailAlreadyExists"), nil)
		}
		u.lo.Error("error creating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// UpdateAgent updates an agent in the database, including their password if provided.
func (u *Manager) UpdateAgent(id int, user models.User) error {
	var (
		hashedPassword any
		err            error
	)

	// Set password?
	if user.NewPassword != "" {
		if IsStrongPassword(user.NewPassword) {
			return envelope.NewError(envelope.InputError, PasswordHint, nil)
		}
		hashedPassword, err = bcrypt.GenerateFromPassword([]byte(user.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			u.lo.Error("error generating bcrypt password", "error", err)
			return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
		}
		u.lo.Debug("setting new password for user", "user_id", id)
	}

	// Update user in the database.
	if _, err := u.q.UpdateAgent.Exec(id, user.FirstName, user.LastName, user.Email, pq.Array(user.Roles), user.AvatarURL, hashedPassword, user.Enabled, user.AvailabilityStatus); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorUpdating", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// SoftDeleteAgent soft deletes an agent by ID.
func (u *Manager) SoftDeleteAgent(id int) error {
	// Disallow if user is system user.
	systemUser, err := u.GetSystemUser()
	if err != nil {
		return err
	}
	if id == systemUser.ID {
		return envelope.NewError(envelope.InputError, u.i18n.T("user.cannotDeleteSystemUser"), nil)
	}
	if _, err := u.q.SoftDeleteAgent.Exec(id); err != nil {
		u.lo.Error("error deleting user", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", "{globals.terms.user}"), nil)
	}
	return nil
}

// markInactiveAgentsOffline sets agents offline if they have been inactive for more than 5 minutes.
func (u *Manager) markInactiveAgentsOffline() {
	if res, err := u.q.UpdateInactiveOffline.Exec(); err != nil {
		u.lo.Error("error setting users offline", "error", err)
	} else {
		rows, _ := res.RowsAffected()
		if rows > 0 {
			u.lo.Info("set inactive users offline", "count", rows)
		}
	}
}

// GetAllAgents returns a list of all agents.
func (u *Manager) GetAgents() ([]models.User, error) {
	// Some dirty hack.
	return u.GetAllUsers(1, 999999999, models.UserTypeAgent, "desc", "users.updated_at", "")
}
