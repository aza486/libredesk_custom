package models

type Template struct {
	Body    string `db:"body"`
	Subject string `db:"subject"`
}
