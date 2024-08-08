package models

type Template struct {
	ID        int    `db:"id" json:"id"`
	Name      string `db:"name" json:"name"`
	Body      string `db:"body" json:"body"`
	IsDefault bool   `db:"is_default" json:"is_default"`
}
