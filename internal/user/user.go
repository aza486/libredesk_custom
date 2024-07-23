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

type Manager struct {
	lo         *logf.Logger
	i18n       *i18n.I18n
	q          queries
	bcryptCost int
}

type Opts struct {
	DB         *sqlx.DB
	Lo         *logf.Logger
	BcryptCost int
}

// Prepared queries.
type queries struct {
	CreateUser      *sqlx.Stmt `query:"create-user"`
	GetUsers        *sqlx.Stmt `query:"get-users"`
	GetUser         *sqlx.Stmt `query:"get-user"`
	GetEmail        *sqlx.Stmt `query:"get-email"`
	GetPermissions  *sqlx.Stmt `query:"get-permissions"`
	GetUserByEmail  *sqlx.Stmt `query:"get-user-by-email"`
	UpdateUser      *sqlx.Stmt `query:"update-user"`
	SetUserPassword *sqlx.Stmt `query:"set-user-password"`
}

func New(i18n *i18n.I18n, opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:          q,
		lo:         opts.Lo,
		i18n:       i18n,
		bcryptCost: opts.BcryptCost,
	}, nil
}

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

func (u *Manager) Create(user *models.User) error {
	var (
		password, _ = u.generatePassword()
	)
	user.Email = strings.ToLower(user.Email)
	if _, err := u.q.CreateUser.Exec(user.Email, user.FirstName, user.LastName, password, user.TeamID, user.AvatarURL, pq.Array(user.Roles)); err != nil {
		u.lo.Error("error creating user", "error", err)
		return err
	}
	return nil
}

func (u *Manager) GetUser(id int, uuid string) (models.User, error) {
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

func (u *Manager) GetSystemUser() (models.User, error) {
	return u.GetUser(0, SystemUserUUID)
}

func (u *Manager) UpdateUser(id int, user models.User) error {
	if _, err := u.q.UpdateUser.Exec(id, user.FirstName, user.LastName, user.Email, user.TeamID, pq.Array(user.Roles)); err != nil {
		u.lo.Error("error updating user", "error", err)
		return err
	}
	return nil
}

func (u *Manager) GetEmail(id int, uuid string) (string, error) {
	var uu interface{}
	if uuid != "" {
		uu = uuid
	}

	var email string
	if err := u.q.GetEmail.Get(&email, id, uu); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return email, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching user from db", "error", err)
		return email, fmt.Errorf("fetching user: %w", err)
	}
	return email, nil
}

func (u *Manager) GetPermissions(id int) ([]string, error) {
	var permissions []string
	if err := u.q.GetPermissions.Select(&permissions, id); err != nil {
		u.lo.Error("error fetching user permissions", "error", err)
		return permissions, envelope.NewError(envelope.GeneralError, "Error fetching user permissions", nil)
	}
	return permissions, nil
}

func (u *Manager) verifyPassword(pwd []byte, pwdHash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(pwdHash), pwd)
	if err != nil {
		return fmt.Errorf("invalid username or password")
	}
	return nil
}

func (u *Manager) setPassword(uid int, pwd string) error {
	// Bcrypt does not operate over 72 bytes.
	if len(pwd) > 72 {
		return ErrPasswordTooLong
	}
	bytes, err := u.generatePassword()
	if err != nil {
		return err
	}
	// Update password in db.
	if _, err := u.q.SetUserPassword.Exec(bytes, uid); err != nil {
		u.lo.Error("setting password", "error", err)
		return fmt.Errorf("error setting password")
	}
	return nil
}

func (u *Manager) generatePassword() ([]byte, error) {
	var password, _ = stringutil.RandomAlNumString(16)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), u.bcryptCost)
	if err != nil {
		u.lo.Error("error generating bcrypt password", "error", err)
		return nil, fmt.Errorf("error generating bcrypt password: %w", err)
	}
	return bytes, nil
}
