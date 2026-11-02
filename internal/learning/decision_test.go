package learning

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"sync"
	"sync/atomic"
	"testing"
)

func preparedCandidate(t *testing.T, m *Manager) string {
	c := validContent()
	c.Source = "human_answer"
	id := candidate(t, m, c)
	insertPreparedFeedback(t, m, id, c)
	return id
}
func insertPreparedFeedback(t *testing.T, m *Manager, id string, c Content) string {
	t.Helper()
	fid := uuid.NewString()
	m.DB.MustExec(`INSERT INTO ai_feedback(id,human_answer) VALUES($1,$2)`, fid, c.Evidence)
	m.DB.MustExec(`UPDATE rag_items SET feedback_id=$2 WHERE id=$1`, id, fid)
	m.DB.MustExec(`UPDATE rag_items SET learning_data=learning_data||'{"proposal_version":2,"review_required":true,"managed_by_review":true,"rationale":"Allgemeine Firmeninformation aus tatsächlicher Antwort unabhängig prüfen.","review_reason":"Gültigkeit prüfen"}' WHERE id=$1`, id)
	m.DB.MustExec(`INSERT INTO learning_reviews(item_id,revision,original) SELECT id,0,to_jsonb(r)-'embedding' FROM rag_items r WHERE id=$1`, id)
	return fid
}
func TestPostgresTwoDecisionProposalLifecycle(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := preparedCandidate(t, m)
	// Neither display, approval nor publication needs a remote anonymizer.
	var calls atomic.Int64
	m.View = func(_ context.Context, i *Item) error {
		calls.Add(1)
		i.Context = &DisplayContext{Question: "Kunde Original", Answer: "Transporttext"}
		return nil
	}
	page, e := m.List(ctx, Filter{Tab: "pending", Page: 1})
	if e != nil || len(page.Items) != 1 || !page.Items[0].ReviewReady || calls.Load() != 1 {
		t.Fatal("prepared view", e, page)
	}

	var wg sync.WaitGroup
	wins := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := m.Decide(ctx, id, "42", Decision{Action: "learn"}); wins <- e }()
	}
	wg.Wait()
	close(wins)
	success := 0
	for e := range wins {
		if e == nil {
			success++
		} else if !errors.Is(e, ErrConflict) {
			t.Fatal(e)
		}
	}
	if success != 1 {
		t.Fatal("duplicate approval", success)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	if e = m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	item, e := m.Get(ctx, id)
	if e != nil || !item.Active || item.Publish != "published" {
		t.Fatal("publication", e, item)
	}
	var n int
	m.DB.Get(&n, `SELECT count(*) FROM learning_review_audit WHERE item_id=$1`, id)
	if n != 2 {
		t.Fatal("decision and publish audit", n)
	}
	if _, e = m.Decide(ctx, id, "42", Decision{Action: "learn", Revision: item.Revision}); !errors.Is(e, ErrConflict) {
		t.Fatal("repeat publication", e)
	}
}
func TestPostgresDiscardAndLegacyExclusion(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := preparedCandidate(t, m)
	legacy := candidate(t, m, validContent())
	if _, e := m.Decide(ctx, legacy, "42", Decision{Action: "learn"}); !errors.Is(e, ErrConflict) {
		t.Fatal("legacy incomplete approval", e)
	}
	if _, e := m.Decide(ctx, id, "42", Decision{Action: "discard"}); e != nil {
		t.Fatal(e)
	}
	if e := m.PublishOne(ctx); e != nil {
		t.Fatal(e)
	}
	item, e := m.Get(ctx, id)
	if e != nil || item.Active || item.Decision != "rejected" || len(item.History) != 1 {
		t.Fatal("discard", e, item)
	}
	if _, e = m.Decide(ctx, id, "42", Decision{Action: "learn", Revision: 1}); !errors.Is(e, ErrConflict) {
		t.Fatal("rejected republished", e)
	}
}
func TestPostgresAuthorizedViewAndPrivacyIsolation(t *testing.T) {
	m := isolated(t)
	ctx := context.Background()
	id := preparedCandidate(t, m)
	m.View = func(context.Context, *Item) error { return ErrForbidden }
	if _, e := m.Get(ctx, id); !errors.Is(e, ErrForbidden) {
		t.Fatal("cross-ticket detail", e)
	}
	page, e := m.List(ctx, Filter{Tab: "pending", Page: 1})
	if e != nil || len(page.Items) != 0 {
		t.Fatal("cross-ticket list", e, page)
	}
	if _, e = m.Decide(ctx, id, "42", Decision{Action: "learn"}); !errors.Is(e, ErrForbidden) {
		t.Fatal("cross-ticket mutation", e)
	}
	m.View = func(context.Context, *Item) error { return nil }
	m.DB.SetMaxOpenConns(1)
	m.DB.MustExec(`SELECT set_config('libredesk.review_writer','true',false)`)
	m.DB.MustExec(`UPDATE rag_items SET retrieval_text='Bitte Kunde person@example.org kontaktieren' WHERE id=$1`, id)
	if _, e = m.Decide(ctx, id, "42", Decision{Action: "learn"}); e == nil {
		t.Fatal("PII learned")
	}
	m.DB.MustExec(`SELECT set_config('libredesk.review_writer','true',false)`)
	m.DB.MustExec(`UPDATE rag_items SET retrieval_text='Wir stornieren Ihre Bestellung' WHERE id=$1`, id)
	if _, e = m.Decide(ctx, id, "42", Decision{Action: "learn"}); e == nil {
		t.Fatal("cancellation bypass")
	}
	for _, a := range []string{"edit", "create", "derive", "approve", "retry"} {
		if _, e = m.Decide(ctx, id, "42", Decision{Action: a}); !errors.Is(e, ErrInput) {
			t.Fatal("old browser action", a, e)
		}
	}
}
