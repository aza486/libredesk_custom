package models

import (
	"time"

	tmodels "github.com/abhinavxd/artemis/internal/team/models"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

type User struct {
	ID               int            `db:"id" json:"id"`
	CreatedAt        time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at" json:"updated_at"`
	FirstName        string         `db:"first_name" json:"first_name"`
	LastName         string         `db:"last_name" json:"last_name"`
	Email            string         `db:"email" json:"email,omitempty"`
	AvatarURL        null.String    `db:"avatar_url" json:"avatar_url"`
	Disabled         bool           `db:"disabled" json:"disabled"`
	Password         string         `db:"password" json:"-"`
	NewPassword      string         `db:"-" json:"new_password,omitempty"`
	SendWelcomeEmail bool           `db:"-" json:"send_welcome_email,omitempty"`
	Roles            pq.StringArray `db:"roles" json:"roles"`
	Permissions      pq.StringArray `db:"permissions" json:"permissions"`
	Teams            tmodels.Teams  `db:"teams" json:"teams"`
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}
