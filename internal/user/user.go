// Package user handles user login, logout and provides functions to fetch user details.
package user

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/user/models"
	"github.com/jmoiron/sqlx"
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

type Manager struct {
	lo         *logf.Logger
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
	GetUsers        *sqlx.Stmt `query:"get-users"`
	GetUser         *sqlx.Stmt `query:"get-user"`
	GetEmail        *sqlx.Stmt `query:"get-email"`
	GetUserByEmail  *sqlx.Stmt `query:"get-user-by-email"`
	SetUserPassword *sqlx.Stmt `query:"set-user-password"`
}

func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:          q,
		lo:         opts.Lo,
		bcryptCost: opts.BcryptCost,
	}, nil
}

func (u *Manager) Login(email string, password []byte) (models.User, error) {
	var user models.User

	if email == "" {
		return user, fmt.Errorf("empty `email`")
	}

	// Check if user exists.
	if err := u.q.GetUserByEmail.Get(&user, email); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return user, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching user from db", "error", err)
		return user, fmt.Errorf("error logging in")
	}

	if err := u.verifyPassword(password, user.Password); err != nil {
		return user, fmt.Errorf("invalid email or password")
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
		return users, fmt.Errorf("error fetching users")
	}

	return users, nil
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
	bytes, err := bcrypt.GenerateFromPassword([]byte(pwd), u.bcryptCost)
	if err != nil {
		u.lo.Error("setting password", "error", err)
		return fmt.Errorf("error setting password")
	}

	// Update password in db.
	if _, err := u.q.SetUserPassword.Exec(bytes, uid); err != nil {
		u.lo.Error("setting password", "error", err)
		return fmt.Errorf("error setting password")
	}

	return nil
}
