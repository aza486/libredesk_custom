package learning

import (
	"context"
	"encoding/json"
)

// Decision is the entire browser write contract. Content and technical flags are never accepted.
type Decision struct {
	Action   string `json:"action"`
	Revision int64  `json:"revision"`
}

func (m *Manager) Decide(ctx context.Context, id, actor string, d Decision) (string, error) {
	if (d.Action != "learn" && d.Action != "discard") || d.Revision < 0 {
		return "", ErrInput
	}
	item, err := m.Get(ctx, id)
	if err != nil {
		return "", err
	}
	if !Prepared(item) || item.Decision != "pending" || item.Active || item.Publish == "publishing" {
		return "", ErrConflict
	}
	action, reason := "approve", "Vorschlag bewusst geprüft und zum Lernen freigegeben."
	if d.Action == "discard" {
		action = "reject"
		reason = "Vorschlag nach Prüfung verworfen."
	}
	// Checks express this explicit decision over persisted content, never client-controlled flags.
	return m.Mutate(ctx, id, actor, Mutation{decisionV2: true, Action: action, Revision: d.Revision, Reason: reason, Checks: Checks{Privacy: true, Content: true, General: true}})
}

func Prepared(item Item) bool {
	var p struct {
		Version  int  `json:"proposal_version"`
		Required bool `json:"review_required"`
	}
	if !item.ReviewExists || (item.Content.Source != "human_answer" && item.Content.Source != "feedback_reason") || json.Unmarshal(item.Provenance, &p) != nil || p.Version != 2 || !p.Required || item.Content.Type != "employee_fact" || item.Rationale == "" {
		return false
	}
	return Validate(item.Content, Checks{Privacy: true, Content: true, General: true}, true) == nil
}
