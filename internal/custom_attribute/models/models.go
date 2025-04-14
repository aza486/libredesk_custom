package models

import (
	"time"

	"github.com/lib/pq"
)

type CustomAttribute struct {
	ID          int            `db:"id" json:"id"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
	AppliesTo   string         `db:"applies_to" json:"applies_to"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description"`
	Key         string         `db:"key" json:"key"`
	Values      pq.StringArray `db:"values" json:"values"`
	DataType    string         `db:"data_type" json:"data_type"`
	Regex       string         `db:"regex" json:"regex"`
	RegexHint   string         `db:"regex_hint" json:"regex_hint"`
}
