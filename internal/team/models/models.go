package models

type Team struct {
	ID   int    `db:"id" json:"id"`
	Name string `db:"name" json:"name"`
	UUID string `db:"uuid" json:"uuid"`
}
