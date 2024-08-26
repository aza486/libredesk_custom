// Package user handles user login, logout and provides functions to fetch user details.
package user

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/stringutil"
	"github.com/abhinavxd/artemis/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
)

var (
	//go:embed queries.sql
	efs embed.FS

	// ErrPasswordTooLong is returned when the password passed to
	// GenerateFromPassword is too long (i.e. > 72 bytes).
	ErrPasswordTooLong = errors.New("password length exceeds 72 bytes")
)

const (
	SystemUserUUID = "00000000-0000-0000-0000-000000000000"
)

// Manager handles user-related operations.
type Manager struct {
	lo   *logf.Logger
	i18n *i18n.I18n
	q    queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	CreateUser      *sqlx.Stmt `query:"create-user"`
	GetUsers        *sqlx.Stmt `query:"get-users"`
	GetUser         *sqlx.Stmt `query:"get-user"`
	GetEmail        *sqlx.Stmt `query:"get-email"`
	GetPermissions  *sqlx.Stmt `query:"get-permissions"`
	GetUserByEmail  *sqlx.Stmt `query:"get-user-by-email"`
	UpdateUser      *sqlx.Stmt `query:"update-user"`
	UpdateAvatar    *sqlx.Stmt `query:"update-avatar"`
	SetUserPassword *sqlx.Stmt `query:"set-user-password"`
}

// New creates and returns a new instance of the Manager.
func New(i18n *i18n.I18n, opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:    q,
		lo:   opts.Lo,
		i18n: i18n,
	}, nil
}

// Login authenticates a user by email and password.
func (u *Manager) Login(email string, password []byte) (models.User, error) {
	var user models.User

	if err := u.q.GetUserByEmail.Get(&user, email); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return user, envelope.NewError(envelope.InputError, u.i18n.T("user.invalidEmailPassword"), nil)
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", "{globals.entities.user}"), nil)
	}

	if err := u.verifyPassword(password, user.Password); err != nil {
		return user, envelope.NewError(envelope.InputError, u.i18n.T("user.invalidEmailPassword"), nil)
	}

	return user, nil
}

// GetUsers retrieves all users.
func (u *Manager) GetUsers() ([]models.User, error) {
	var users []models.User
	if err := u.q.GetUsers.Select(&users); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return users, nil
		}
		u.lo.Error("error fetching users from db", "error", err)
		return users, fmt.Errorf("error fetching users: %w", err)
	}

	return users, nil
}

// Create creates a new user.
func (u *Manager) Create(user *models.User) error {
	password, err := u.generatePassword()
	if err != nil {
		u.lo.Error("error generating password", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating user", nil)
	}

	user.Email = strings.ToLower(user.Email)
	if err := u.q.CreateUser.QueryRow(user.Email, user.FirstName, user.LastName, password, user.AvatarURL, pq.Array(user.Roles)).Scan(&user.ID); err != nil {
		u.lo.Error("error creating user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating user", nil)
	}
	return nil
}

// Get retrieves a user by ID or UUID.
func (u *Manager) Get(id int, uuid string) (models.User, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var user models.User
	if err := u.q.GetUser.Get(&user, id, uu); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, fmt.Errorf("fetching user: %w", err)
	}
	return user, nil
}

// GetByEmail retrieves a user by email
func (u *Manager) GetByEmail(email string) (models.User, error) {
	var user models.User
	if err := u.q.GetUserByEmail.Get(&user, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, err
	}
	return user, nil
}

// GetSystemUser retrieves the system user.
func (u *Manager) GetSystemUser() (models.User, error) {
	return u.Get(0, SystemUserUUID)
}

// UpdateAvatar updates the user avatar.
func (u *Manager) UpdateAvatar(id int, avatar string) error {
	if _, err := u.q.UpdateAvatar.Exec(id, null.NewString(avatar, avatar != "")); err != nil {
		u.lo.Error("error updating user avatar", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating avatar", nil)
	}
	return nil
}

// UpdateUser updates a user.
func (u *Manager) UpdateUser(id int, user models.User) error {
	if _, err := u.q.UpdateUser.Exec(id, user.FirstName, user.LastName, user.Email, pq.Array(user.Roles), user.AvatarURL); err != nil {
		u.lo.Error("error updating user", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating user", nil)
	}
	return nil
}

// GetEmail retrieves the email of a user by ID.
func (u *Manager) GetEmail(id int) (string, error) {
	var email string
	if err := u.q.GetEmail.Get(&email, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return email, fmt.Errorf("user not found: %v", err)
		}
		u.lo.Error("error fetching user email from db", "error", err)
		return email, fmt.Errorf("fetching user: %w", err)
	}
	return email, nil
}

// GetPermissions retrieves the permissions of a user by ID.
func (u *Manager) GetPermissions(id int) ([]string, error) {
	var permissions []string
	if err := u.q.GetPermissions.Select(&permissions, id); err != nil {
		u.lo.Error("error fetching user permissions", "error", err)
		return permissions, envelope.NewError(envelope.GeneralError, "Error fetching user permissions", nil)
	}
	return permissions, nil
}

// verifyPassword compares the provided password with the stored password hash.
func (u *Manager) verifyPassword(pwd []byte, pwdHash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(pwdHash), pwd); err != nil {
		return fmt.Errorf("invalid username or password")
	}
	return nil
}

// setPassword sets a new password for a user.
func (u *Manager) setPassword(uid int, pwd string) error {
	if len(pwd) > 72 {
		return ErrPasswordTooLong
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(pwd), 12)
	if err != nil {
		return err
	}
	if _, err := u.q.SetUserPassword.Exec(bytes, uid); err != nil {
		u.lo.Error("setting password", "error", err)
		return fmt.Errorf("error setting password")
	}
	return nil
}

// generatePassword generates a random password and returns its bcrypt hash.
func (u *Manager) generatePassword() ([]byte, error) {
	password, _ := stringutil.RandomAlNumString(16)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		u.lo.Error("error generating bcrypt password", "error", err)
		return nil, fmt.Errorf("error generating bcrypt password: %w", err)
	}
	return bytes, nil
}
