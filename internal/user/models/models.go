package models

import (
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

type User struct {
	ID               int            `db:"id" json:"id"`
	UUID             string         `db:"uuid" json:"uuid"`
	FirstName        string         `db:"first_name" json:"first_name"`
	LastName         string         `db:"last_name" json:"last_name"`
	Email            string         `db:"email" json:"email,omitempty"`
	AvatarURL        null.String    `db:"avatar_url" json:"avatar_url"`
	Disabled         bool           `db:"disabled" json:"disabled"`
	TeamID           int            `db:"team_id" json:"team_id"`
	Password         string         `db:"password" json:"-"`
	TeamName         null.String    `db:"team_name" json:"team_name"`
	Roles            pq.StringArray `db:"roles" json:"roles"`
	SendWelcomeEmail bool           `db:"-" json:"send_welcome_email"`
	Permissions      pq.StringArray `db:"permissions" json:"permissions"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}
