package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/volatiletech/null/v9"
)

type Team struct {
	ID                         int         `db:"id" json:"id"`
	CreatedAt                  time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt                  time.Time   `db:"updated_at" json:"updated_at"`
	Emoji                      null.String `db:"emoji" json:"emoji"`
	Name                       string      `db:"name" json:"name"`
	ConversationAssignmentType string      `db:"conversation_assignment_type" json:"conversation_assignment_type,omitempty"`
	Timezone                   string      `db:"timezone" json:"timezone,omitempty"`
	BusinessHoursID            null.Int    `db:"business_hours_id" json:"business_hours_id,omitempty"`
	SLAPolicyID                null.Int    `db:"sla_policy_id" json:"sla_policy_id,omitempty"`
}

type Teams []Team

// Scan implements the sql.Scanner interface for Teams
func (t *Teams) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}

	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	default:
		return fmt.Errorf("unsupported type for Teams: %T", src)
	}
}

// Value implements the driver.Valuer interface for Teams
func (t Teams) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Names returns the names of the teams in Teams slice.
func (t Teams) Names() []string {
	names := make([]string, len(t))
	for i, team := range t {
		names[i] = team.Name
	}
	return names
}

// IDs returns a slice of all team IDs in the Teams slice.
func (t Teams) IDs() []int {
	ids := make([]int, len(t))
	for i, team := range t {
		ids[i] = team.ID
	}
	return ids
}
