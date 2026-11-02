package learning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestLocalPrivacyChecksDoNotChangePreparedText(t *testing.T) {
	c := austriaContent()
	before := c
	if len([]rune(c.Retrieval)) != 311 {
		t.Fatal("regression requires the unchanged 311-character text")
	}
	if err := validatePrivacy(c); err != nil {
		t.Fatal(err)
	}
	if c.Topic != before.Topic || c.Retrieval != before.Retrieval || c.Rule != before.Rule || c.Evidence != before.Evidence {
		t.Fatal("local validation modified candidate")
	}
	if err := Validate(c, allChecks(), true); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresLocalValidationDoesNotNeedAnonymizer(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := preparedCandidate(t, m)
	if _, err := m.Decide(ctx, id, "42", Decision{Action: "learn", Revision: 0}); err != nil {
		t.Fatal(err)
	}
	if err := m.PublishOne(ctx); err != nil {
		t.Fatal(err)
	}
	item, err := m.Get(ctx, id)
	if err != nil || !item.Active || item.Publish != "published" {
		t.Fatal("local privacy publication", err, item.Publish)
	}
}

func TestLocalPrivacyDiagnosticsDoNotExposeValues(t *testing.T) {
	c := validContent()
	c.Retrieval = "Email SENSITIVE_PERSON@example.org"
	err := Validate(c, allChecks(), true)
	if !errors.Is(err, ErrInput) || DiagnosticCode(err) != "privacy_email" || strings.Contains(err.Error(), "SENSITIVE") {
		t.Fatal("unsafe input diagnosis", err)
	}
}
