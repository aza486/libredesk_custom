package main

import (
	"time"

	"github.com/zerodha/fastglue"
)

type Priority struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}

func handleGetPriorities(r *fastglue.Request) error {
	priorities := []Priority{
		{ID: 1, Name: "Low"},
		{ID: 2, Name: "Medium"},
		{ID: 3, Name: "High"},
	}
	return r.SendEnvelope(priorities)
}
