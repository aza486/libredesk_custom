package learning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAnonymizerFailuresRemainFailClosedAndSafe(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		anon Anonymizer
		kind string
	}{
		{name: "not configured", kind: "not_configured"},
		{name: "empty response", anon: func(context.Context, string) (string, error) { return "", nil }, kind: "invalid_response"},
		{name: "PII remains", anon: func(context.Context, string) (string, error) { return "person@example.com", nil }, kind: "privacy_rejected"},
		{name: "raw provider error", anon: func(context.Context, string) (string, error) { return "", errors.New("PRIVATE_TOKEN_CUSTOMER_TEXT") }, kind: "request_failed"},
		{name: "timeout", anon: func(context.Context, string) (string, error) { return "", context.DeadlineExceeded }, kind: "timeout"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{Anonymize: tt.anon}
			_, err := m.safe(ctx, "Synthetic input")
			var dep *DependencyError
			if !errors.Is(err, ErrUnavailable) || !errors.As(err, &dep) || dep.Kind != tt.kind {
				t.Fatal("unsafe failure classification", err)
			}
			if strings.Contains(err.Error(), "PRIVATE") {
				t.Fatal("provider data leaked")
			}
			if err = m.privacyChecked(ctx, validContent()); !errors.Is(err, ErrUnavailable) {
				t.Fatal("privacy bypass", err)
			}
		})
	}
	m := &Manager{Anonymize: func(_ context.Context, s string) (string, error) { return s, nil }}
	if err := m.privacyChecked(ctx, validContent()); err != nil {
		t.Fatal(err)
	}
	m.Anonymize = func(context.Context, string) (string, error) { return "Changed content", nil }
	if err := m.privacyChecked(ctx, validContent()); !errors.Is(err, ErrInput) {
		t.Fatal("changed anonymized text was approved", err)
	}
}

func TestPostgresMissingAnonymizerRollsBackLearningDecision(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := preparedCandidate(t, m)
	m.Anonymize = nil
	_, err := m.Decide(ctx, id, "42", Decision{Action: "learn", Revision: 0})
	var dep *DependencyError
	if !errors.Is(err, ErrUnavailable) || !errors.As(err, &dep) || dep.Service != "anonymizer" || dep.Kind != "not_configured" {
		t.Fatal("missing anonymizer does not explain 503", err)
	}
	item, err := m.Get(ctx, id)
	if err != nil || item.Active || item.Approved || item.Decision != "pending" || item.Revision != 0 || len(item.History) != 0 {
		t.Fatal("failed decision mutated candidate", err, item)
	}
	m.Anonymize = func(_ context.Context, s string) (string, error) { return s, nil }
	if _, err = m.Decide(ctx, id, "42", Decision{Action: "learn", Revision: 0}); err != nil {
		t.Fatal("configured anonymizer did not unblock decision", err)
	}
}
