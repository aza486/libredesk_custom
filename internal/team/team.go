// Package team handles the management of teams and their members.
package team

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/abhinavxd/artemis/internal/team/models"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles team-related operations.
type Manager struct {
	lo *logf.Logger
	q  queries
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetTeams        *sqlx.Stmt `query:"get-teams"`
	GetTeam         *sqlx.Stmt `query:"get-team"`
	InsertTeam      *sqlx.Stmt `query:"insert-team"`
	UpdateTeam      *sqlx.Stmt `query:"update-team"`
	GetTeamMembers  *sqlx.Stmt `query:"get-team-members"`
	UpsertUserTeams *sqlx.Stmt `query:"upsert-user-teams"`
}

// New creates and returns a new instance of the Manager.
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

// GetAll retrieves all teams.
func (u *Manager) GetAll() ([]models.Team, error) {
	var teams = make([]models.Team, 0)
	if err := u.q.GetTeams.Select(&teams); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return teams, nil
		}
		u.lo.Error("error fetching teams from db", "error", err)
		return teams, fmt.Errorf("error fetching teams")
	}
	return teams, nil
}

// GetTeam retrieves a team by ID.
func (u *Manager) GetTeam(id int) (models.Team, error) {
	var team models.Team
	if err := u.q.GetTeam.Get(&team, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			u.lo.Error("team not found", "id", id, "error", err)
			return team, nil
		}
		u.lo.Error("error fetching team", "id", id, "error", err)
		return team, err
	}
	return team, nil
}

// CreateTeam creates a new team.
func (u *Manager) CreateTeam(t models.Team) error {
	if _, err := u.q.InsertTeam.Exec(t.Name); err != nil {
		u.lo.Error("error inserting team", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating team", nil)
	}
	return nil
}

// UpdateTeam updates an existing team.
func (u *Manager) UpdateTeam(id int, t models.Team) error {
	if _, err := u.q.UpdateTeam.Exec(id, t.Name, t.AutoAssignConversations); err != nil {
		u.lo.Error("error updating team", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating team", nil)
	}
	return nil
}

// GetTeamMembers retrieves members of a team by team name.
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

// UpsertUserTeams updates/inserts exists user teams
func (u *Manager) UpsertUserTeams(id int, teamNames []string) error {
	if _, err := u.q.UpsertUserTeams.Exec(id, pq.Array(teamNames)); err != nil {
		u.lo.Error("error updating user teams", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating user team", nil)
	}
	return nil
}
