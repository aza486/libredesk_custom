package user

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/abhinavxd/artemis/internal/userdb/models"
	"github.com/abhinavxd/artemis/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
	"golang.org/x/crypto/bcrypt"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type UserDB struct {
	lo         *logf.Logger
	q          queries
	bcryptCost int
}

type Opts struct {
	DB         *sqlx.DB
	Lo         *logf.Logger
	BcryptCost int
}

type queries struct {
	GetAgents        *sqlx.Stmt `query:"get-agents"`
	GetAgent         *sqlx.Stmt `query:"get-agent"`
	SetAgentPassword *sqlx.Stmt `query:"set-agent-password"`
	GetTeams         *sqlx.Stmt `query:"get-teams"`
}

var (
	// ErrPasswordTooLong is returned when the password passed to
	// GenerateFromPassword is too long (i.e. > 72 bytes).
	ErrPasswordTooLong = errors.New("bcrypt: password length exceeds 72 bytes")
)

func New(opts Opts) (*UserDB, error) {
	var q queries

	if err := utils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &UserDB{
		q:          q,
		lo:         opts.Lo,
		bcryptCost: opts.BcryptCost,
	}, nil
}

func (u *UserDB) Login(email string, password []byte) (models.User, error) {
	var user models.User

	if email == "" {
		return user, fmt.Errorf("empty `email`")
	}

	// Check if user exists.
	if err := u.q.GetAgent.Get(&user, email); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return user, fmt.Errorf("user not found")
		}
		u.lo.Error("error fetching agents from db", "error", err)
		return user, fmt.Errorf("error logging in")
	}

	if err := u.verifyPassword(password, user.Password); err != nil {
		return user, fmt.Errorf("error logging in")
	}

	return user, nil
}

func (u *UserDB) verifyPassword(pwd []byte, pwdHash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(pwdHash), pwd)
	if err != nil {
		return fmt.Errorf("invalid username or password")
	}
	return nil
}

func (u *UserDB) GetAgents() ([]models.User, error) {
	var agent []models.User
	if err := u.q.GetAgents.Select(&agent); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return agent, nil
		}
		u.lo.Error("error fetching agents from db", "error", err)
		return agent, fmt.Errorf("error fetching agents")
	}

	return agent, nil
}

func (u *UserDB) GetTeams() ([]models.Teams, error) {
	var teams []models.Teams
	if err := u.q.GetTeams.Select(&teams); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return teams, nil
		}
		u.lo.Error("error fetching teams from db", "error", err)
		return teams, fmt.Errorf("error fetching teams")
	}
	return teams, nil
}

func (u *UserDB) GetAgent(email string) (models.User, error) {
	var user models.User
	if err := u.q.GetAgent.Get(&user, email); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return user, fmt.Errorf("agent not found")
		}
		u.lo.Error("error fetching agent from db", "error", err)
		return user, fmt.Errorf("fetching agent")
	}
	return user, nil
}

func (u *UserDB) setPassword(uid int, pwd string) error {
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
	if _, err := u.q.SetAgentPassword.Exec(bytes, uid); err != nil {
		u.lo.Error("setting password", "error", err)
		return fmt.Errorf("error setting password")
	}

	return nil
}
