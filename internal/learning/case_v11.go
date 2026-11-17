package learning

import (
	"context"
	_ "embed"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CaseV11Schema is an opt-in pg-n8n artifact. Never execute at application startup.
//
//go:embed artifacts/005_case_v11.sql
var CaseV11Schema string

// CaseChecks are explicit confirmations from an authenticated reviewer.
type CaseChecks struct {
	Privacy        bool `json:"privacy"`
	Generalization bool `json:"generalization"`
	Quality        bool `json:"quality"`
}

type CaseDecision struct {
	Action   string     `json:"action"`
	Revision int64      `json:"revision"`
	Checks   CaseChecks `json:"checks"`
}

// ValidateCaseExample checks structure only. Upstream anonymization and explicit
// human privacy, generalization and quality confirmations govern case content;
// review and publication do not repeat automatic privacy/regex checks.
func ValidateCaseExample(item Item) error {
	c := item.Content
	if c.Type != "case_example" || !text(c.Question, 6000) || utf8.RuneCountInString(strings.TrimSpace(c.Question)) < 12 ||
		!text(c.Answer, 6000) || strings.TrimSpace(c.Answer) == "" || !text(c.Retrieval, 900) || utf8.RuneCountInString(strings.TrimSpace(c.Retrieval)) < 12 || !text(item.Reason, 6000) {
		return invalid("case_fields_invalid")
	}
	return nil
}

var caseActorID = regexp.MustCompile(`^[1-9][0-9]*$`)

// ReviewCase authorizes the stored source under the item lock and invokes SQL005.
// It never embeds or schedules publication, and never accepts an actor from JSON.
func (m *Manager) ReviewCase(ctx context.Context, id, actor string, d CaseDecision) (int64, error) {
	if _, err := uuid.Parse(id); err != nil {
		return 0, ErrInput
	}
	if !caseActorID.MatchString(actor) || d.Revision < 0 || !contains([]string{"approve", "reject", "block"}, d.Action) {
		return 0, ErrInput
	}
	if d.Action == "approve" && (!d.Checks.Privacy || !d.Checks.Generalization || !d.Checks.Quality) {
		return 0, invalid("case_checks_missing")
	}
	tx, err := m.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var raw []byte
	if err = tx.QueryRowContext(ctx, projection+` WHERE r.id=$1::uuid FOR UPDATE OF r`, id).Scan(&raw); err != nil {
		return 0, err
	}
	item, err := decodeItem(raw)
	if err != nil {
		return 0, err
	}
	if m.View != nil {
		if err = m.View(ctx, &item); err != nil {
			return 0, err
		}
	}
	if item.Content.Type != "case_example" || item.Revision != d.Revision {
		return 0, ErrConflict
	}
	if d.Action == "approve" {
		if err = ValidateCaseExample(item); err != nil {
			return 0, err
		}
	}
	decision := map[string]string{"approve": "approved", "reject": "rejected", "block": "blocked"}[d.Action]
	var revision int64
	err = tx.QueryRowContext(ctx, `SELECT public.learning_case_review_v11($1::uuid,$2::bigint,$3::text,$4::text,$5::boolean,$6::boolean,$7::boolean,'[]'::jsonb)`, id, d.Revision, decision, actor, d.Checks.Privacy, d.Checks.Quality, d.Checks.Generalization).Scan(&revision)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "40001" {
			return 0, ErrConflict
		}
		return 0, err
	}
	return revision, tx.Commit()
}
