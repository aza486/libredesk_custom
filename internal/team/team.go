package team

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/abhinavxd/artemis/internal/dbutil"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Team struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
	UUID string `db:"uuid" json:"uuid"`
}

type Manager struct {
	lo *logf.Logger
	q  queries
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetTeams       *sqlx.Stmt `query:"get-teams"`
	GetTeam        *sqlx.Stmt `query:"get-team"`
	GetTeamMembers *sqlx.Stmt `query:"get-team-members"`
}

func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (u *Manager) GetAll() ([]Team, error) {
	var teams []Team
	if err := u.q.GetTeams.Select(&teams); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return teams, nil
		}
		u.lo.Error("error fetching teams from db", "error", err)
		return teams, fmt.Errorf("error fetching teams")
	}
	return teams, nil
}

func (u *Manager) GetTeam(uuid string) (Team, error) {
	var team Team
	if err := u.q.GetTeam.Get(&team, uuid); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return team, nil
		}
		u.lo.Error("error fetching team from db", "uuid", uuid, "error", err)
		return team, fmt.Errorf("error fetching team")
	}
	return team, nil
}

func (u *Manager) GetTeamMembers(name string) ([]umodels.User, error) {
	var users []umodels.User
	if err := u.q.GetTeamMembers.Select(&users, name); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return users, nil
		}
		u.lo.Error("error fetching team members from db", "team_name", name, "error", err)
		return users, fmt.Errorf("fetching team members: %w", err)
	}
	return users, nil
}
