package models

type User struct {
	ID        string  `db:"id" json:"-"`
	UUID      string  `db:"uuid" json:"uuid"`
	FirstName string  `db:"first_name" json:"first_name"`
	LastName  *string `db:"last_name" json:"last_name"`
	Email     string  `db:"email" json:"email,omitempty"`
	AvatarURL *string `db:"avatar_url" json:"avatar_url,omitempty"`
	Password  string  `db:"password" json:"-"`
}

type Teams struct {
	ID   string `db:"id" json:"-"`
	Name string `db:"name" json:"name"`
	UUID string `db:"uuid" json:"uuid"`
}
