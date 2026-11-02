package learning

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func austriaContent() Content {
	c := validContent()
	c.Topic = "Versand nach Österreich"
	c.Category = "versandstatus"
	c.Question = "Ist ein Versand nach Österreich möglich?"
	c.Rule = "Wir versenden nach Österreich."
	c.Evidence = c.Rule
	c.Source = "human_answer"
	c.Limits = "Allgemeiner Versand, ohne konkrete Liefertermine."
	c.Retrieval = "Versand nach Österreich: allgemeine Versandinformation. "
	c.Retrieval += strings.Repeat("x", 311-len([]rune(c.Retrieval)))
	return c
}
func TestLocalPrivacyRejectsStructuredPersonalData(t *testing.T) {
	for _, tt := range []struct{ value, code string }{
		{"Kontakt: person@example.org", "privacy_email"},
		{"Kontakt: person&#64;example.org", "privacy_email"},
		{"Telefon +43 1 23456789", "privacy_phone"},
		{"Telefon 030 1234567", "privacy_phone"},
		{"Tel. 123456", "privacy_phone"},
		{"Konto AT61 1904 3002 3457 3201", "privacy_iban"},
		{"Konto CH93 0076 2011 6238 5295 7", "privacy_iban"},
		{"Musterstraße 12", "privacy_address"},
		{"Hauptstr. 4/2", "privacy_address"},
		{"Am Markt 12", "privacy_address"},
		{"Neubaugürtel 12/3", "privacy_address"},
		{"221B Baker Street", "privacy_address"},
		{"Privatanschrift: 1010 Wien", "privacy_address"},
		{"Kundennummer: 1234", "privacy_identifier"},
		{"versteckte\u200bAngabe", "privacy_ambiguous_text"},
	} {
		t.Run(tt.code+tt.value[:3], func(t *testing.T) {
			c := austriaContent()
			c.Retrieval = tt.value
			err := validatePrivacy(c)
			if !errors.Is(err, ErrInput) || DiagnosticCode(err) != tt.code {
				t.Fatalf("expected %s, got %v / %s", tt.code, err, DiagnosticCode(err))
			}
		})
	}
	// All persisted candidate text fields, not just the vector input, are checked.
	for _, field := range []string{"topic", "question", "answer", "rule", "evidence", "source", "limits"} {
		t.Run(field, func(t *testing.T) {
			c := austriaContent()
			p := map[string]*string{"topic": &c.Topic, "question": &c.Question, "answer": &c.Answer, "rule": &c.Rule, "evidence": &c.Evidence, "source": &c.Source, "limits": &c.Limits}[field]
			*p = "person@example.org"
			if err := Validate(c, allChecks(), true); !errors.Is(err, ErrInput) {
				t.Fatal("PII field allowed", err)
			}
		})
	}
}
func TestLocalPrivacyAllowsGeneralPlacesAndQuantities(t *testing.T) {
	for _, value := range []string{"Versand nach Österreich und in die Schweiz.", "Versand von 1000 Stück ist möglich.", "Österreich, Region Wien, Postleitzahl 1010."} {
		c := austriaContent()
		c.Retrieval = value
		if err := Validate(c, allChecks(), true); err != nil {
			t.Fatal("general information rejected", err)
		}
	}
}
func TestEvidenceIsAnExactQuoteFromTheSelectedSource(t *testing.T) {
	fact := "Wir versenden nach Österreich."
	for _, tt := range []struct{ name, source, kind, quote, code string }{
		{"HTML whitespace", "<p>Wir\n versenden nach\tÖsterreich.</p>", "human_answer", fact, ""},
		{"literal reason", "Bitte beachten: " + fact, "feedback_reason", fact, ""},
		{"no HTML decoding in reason", "<b>Wir</b> versenden nach Österreich.", "feedback_reason", fact, "evidence_quote_not_found"},
		{"not a paraphrase", fact, "human_answer", "Lieferungen nach Österreich sind möglich.", "evidence_quote_not_found"},
		{"not case folded", fact, "human_answer", "wir versenden nach Österreich.", "evidence_quote_not_found"},
		{"empty quote", fact, "human_answer", " ", "evidence_quote_not_found"},
		{"source kind", fact, "raw_text", fact, "evidence_source_invalid"},
		{"oversize", strings.Repeat("x", maxEvidenceSourceBytes+1), "human_answer", fact, "evidence_source_too_large"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := evidenceInSource(tt.source, tt.kind, tt.quote)
			if tt.code == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrInput) || DiagnosticCode(err) != tt.code {
				t.Fatal("evidence diagnostic", err, DiagnosticCode(err))
			}
		})
	}
}
func TestPostgresAustriaFactWithLongPrivateSource(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := austriaContent()
	id := candidate(t, m, c)
	// Use preparedCandidate's metadata without changing the protected original later.
	fid := insertPreparedFeedback(t, m, id, c)
	source := "<p>" + c.Evidence + "</p><blockquote>" + strings.Repeat("Frühere Nachricht ohne Lerninhalt. ", 2000) + "</blockquote><p>PERSONAL_SOURCE_ONLY person@example.org, Musterstraße 12, Tel. +43 1 23456789</p>"
	if len([]rune(source)) <= 10000 {
		t.Fatal("source must exceed old limit")
	}
	m.DB.MustExec(`UPDATE ai_feedback SET human_answer=$2 WHERE id=$1`, fid, source)
	gotInput := ""
	m.Embed = func(_ context.Context, s string) ([]float64, error) {
		gotInput = s
		v := make([]float64, 1536)
		v[0] = 1
		return v, nil
	}
	if _, err := m.Decide(ctx, id, "42", Decision{Action: "learn"}); err != nil {
		t.Fatal("Austria approval", err)
	}
	if err := m.PublishOne(ctx); err != nil {
		t.Fatal("Austria publication", err)
	}
	var retrieval, answer, rule, all string
	if err := m.DB.QueryRow(`SELECT retrieval_text,COALESCE(human_answer,''),learning_data->>'fact_statement',to_jsonb(r)::text FROM rag_items r WHERE id=$1`, id).Scan(&retrieval, &answer, &rule, &all); err != nil {
		t.Fatal(err)
	}
	if retrieval != c.Retrieval || gotInput != c.Retrieval || answer != "" || rule != c.Rule || strings.Contains(all, "PERSONAL_SOURCE_ONLY") || strings.Contains(all, "person@example.org") {
		t.Fatal("source copied into RAG or prepared text changed")
	}
}

func TestPostgresEvidenceCannotUseAnotherSourceOrInventQuotes(t *testing.T) {
	for _, sourceKind := range []string{"human_answer", "feedback_reason"} {
		t.Run(sourceKind, func(t *testing.T) {
			m := isolated(t)
			ctx := context.Background()
			c := austriaContent()
			c.Source = sourceKind
			id := candidate(t, m, c)
			fid := insertPreparedFeedback(t, m, id, c)
			// The quote is present only in the OTHER source, never a fallback lookup.
			answer, reason := "Unrelated answer without the quote", c.Evidence
			if sourceKind == "feedback_reason" {
				answer, reason = c.Evidence, "Unrelated reason without the quote"
			}
			m.DB.MustExec(`UPDATE ai_feedback SET human_answer=$2,feedback_reason=$3 WHERE id=$1`, fid, answer, reason)
			if _, err := m.Decide(ctx, id, "42", Decision{Action: "learn"}); !errors.Is(err, ErrInput) || DiagnosticCode(err) != "evidence_quote_not_found" {
				t.Fatal("wrong source accepted", err)
			}
			item, err := m.Get(ctx, id)
			if err != nil || item.Decision != "pending" || item.Revision != 0 || len(item.History) != 0 {
				t.Fatal("bad evidence changed review", err)
			}
			if sourceKind == "human_answer" {
				answer = c.Evidence
			} else {
				reason = c.Evidence
			}
			m.DB.MustExec(`UPDATE ai_feedback SET human_answer=$2,feedback_reason=$3 WHERE id=$1`, fid, answer, reason)
			if _, err = m.Decide(ctx, id, "42", Decision{Action: "learn"}); err != nil {
				t.Fatal("selected source not accepted", err)
			}
		})
	}
}
func TestPostgresEvidenceResourceLimitAndBoundedRead(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	c := austriaContent()
	id := candidate(t, m, c)
	fid := insertPreparedFeedback(t, m, id, c)
	m.DB.MustExec(`UPDATE ai_feedback SET human_answer=$2 WHERE id=$1`, fid, c.Evidence+strings.Repeat("x", maxEvidenceSourceBytes))
	item, err := m.Get(ctx, id)
	if err != nil || item.SentAnswer != "" {
		t.Fatal("oversized source loaded in projection", err)
	}
	if _, err = m.Decide(ctx, id, "42", Decision{Action: "learn"}); !errors.Is(err, ErrInput) || DiagnosticCode(err) != "evidence_source_too_large" {
		t.Fatal("source limit", err)
	}
}
func TestPostgresPublisherEnforcesLocalPrivacyBeforeEmbedding(t *testing.T) {
	for _, pii := range []string{"person@example.org", "Telefon +43 1 23456789", "Musterstraße 12", "AT61 1904 3002 3457 3201"} {
		t.Run(pii[:3], func(t *testing.T) {
			m := isolated(t)
			ctx := context.Background()
			id := preparedCandidate(t, m)
			if _, err := m.Decide(ctx, id, "42", Decision{Action: "learn"}); err != nil {
				t.Fatal(err)
			}
			// Simulate an invalid persisted row through an owner transaction, not a UI edit.
			tx := m.DB.MustBegin()
			tx.MustExec(`SELECT set_config('libredesk.review_writer','true',true)`)
			tx.MustExec(`UPDATE rag_items SET learning_data=learning_data||jsonb_build_object('limitations',$2::text) WHERE id=$1`, id, pii)
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			embedCalls := 0
			m.Embed = func(context.Context, string) ([]float64, error) { embedCalls++; return nil, nil }
			err := m.PublishOne(ctx)
			if !errors.Is(err, ErrInput) || DiagnosticCode(err) == "" || embedCalls != 0 {
				t.Fatal("publisher bypass", err, embedCalls)
			}
			item, err := m.Get(ctx, id)
			if err != nil || item.Active || item.Approved || item.Publish != "failed" || item.BlockReason != "publication_validation_failed" {
				t.Fatal("invalid publication state", err, item.Publish)
			}
		})
	}
}
func TestPostgresDisabledAndDecidedReviewsCannotBeApprovedAgain(t *testing.T) {
	for _, decision := range []string{"disabled", "rejected", "approved"} {
		t.Run(decision, func(t *testing.T) {
			m := isolated(t)
			ctx := context.Background()
			id := preparedCandidate(t, m)
			m.DB.MustExec(`UPDATE learning_reviews SET decision=$2,revision=revision+1 WHERE item_id=$1`, id, decision)
			if _, err := m.Decide(ctx, id, "42", Decision{Action: "learn", Revision: 1}); !errors.Is(err, ErrConflict) {
				t.Fatal("state reapproved", err)
			}
		})
	}
}
