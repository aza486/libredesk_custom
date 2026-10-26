package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/aifeedback"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/zerodha/logf"
)

func manualFixture(t *testing.T, name string) (*Manager, string, int, string) {
	t.Helper()
	m, cuuid, cid := completionFixture(t, name)
	lo := logf.New(logf.Opts{})
	m.lo = &lo
	return m, cuuid, cid, input(t, m, cid)
}

func manualNote(t *testing.T, m *Manager, cid int, original string) string {
	t.Helper()
	analysis := AIAnalysis{Kind: "n8n_manual_review", Version: 1, OriginalMessageUUID: original}
	if err := analysis.Validate(); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]any{"ai_analysis": analysis})
	var note string
	if err := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content,meta)
 VALUES($1,'outgoing','sent',101,'agent',true,'Manual review',$2::jsonb) RETURNING uuid`, cid, string(raw)).Scan(&note); err != nil {
		t.Fatal(err)
	}
	return note
}

func outboxCount(t *testing.T, m *Manager) int {
	t.Helper()
	var count int
	if err := m.db.Get(&count, `SELECT count(*) FROM manual_resolution_feedback`); err != nil {
		t.Fatal(err)
	}
	return count
}

type manualFeedbackProbe struct {
	rows []aifeedback.ManualResolutionSubmission
	fail bool
}

func (p *manualFeedbackProbe) SubmitManualResolution(_ context.Context, s aifeedback.ManualResolutionSubmission) error {
	p.rows = append(p.rows, s)
	if p.fail {
		return errors.New("receiver unavailable or commit response lost")
	}
	return nil
}

func TestManualResolutionSentRetryAndRestart(t *testing.T) {
	m, cuuid, cid, original := manualFixture(t, "manual_feedback_retry")
	note := manualNote(t, m, cid, original)
	manualNote(t, m, cid, original) // Both n8n producers may post the same semantic note.
	outgoing := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	state := stateOf(t, m, outgoing)
	if state.ManualReviewNoteUUID == nil || *state.ManualReviewNoteUUID != note || state.ResponseSource != nil {
		t.Fatal("manual source classification changed")
	}
	probe := &manualFeedbackProbe{fail: true}
	m.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 0 || outboxCount(t, m) != 0 {
		t.Fatal("pending generated feedback")
	}
	if _, err := m.q.UpdateMessageStatus.Exec("failed", outgoing); err != nil {
		t.Fatal(err)
	}
	if outboxCount(t, m) != 0 {
		t.Fatal("failed generated feedback")
	}
	sent(t, m, outgoing)
	if outboxCount(t, m) != 1 {
		t.Fatal("missing sent outbox")
	}
	m.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 1 {
		t.Fatal("first attempt missing")
	}
	first := probe.rows[0]
	if first.ID == "" || first.OutgoingMessageUUID != outgoing || first.MailEventMessageUUID != original || first.ConversationUUID != cuuid || first.AgentID != "101" || first.FeedbackReason != nil || first.HumanAnswer != "<p>Actual rendered transport body</p>" || first.FeedbackType != "manual_resolution" || first.ResponseSource != "human" || first.SentAt.IsZero() {
		t.Fatalf("bad payload: %+v", first)
	}
	// Backoff persists even when the worker is restarted.
	restarted := &Manager{db: m.db, lo: m.lo, inboxStore: completionInboxGuard{t: t}}
	restarted.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 1 {
		t.Fatal("backoff lost on restart")
	}
	if _, err := m.db.Exec(`UPDATE manual_resolution_feedback SET next_attempt_at=NOW()`); err != nil {
		t.Fatal(err)
	}
	probe.fail = false
	restarted.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 2 || probe.rows[1] != first {
		t.Fatal("retry changed feedback identity or snapshot")
	}
	sent(t, m, outgoing)
	restarted.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 2 || outboxCount(t, m) != 1 {
		t.Fatal("committed feedback repeated")
	}
	var attempts int
	var committed time.Time
	if err := m.db.QueryRow(`SELECT attempt_count,committed_at FROM manual_resolution_feedback`).Scan(&attempts, &committed); err != nil || attempts != 2 || committed.IsZero() {
		t.Fatalf("receipt: %d %v", attempts, err)
	}
	// Migration replay does not erase deduplication or backfill old sends.
	if err := migrations.V2_15_0(m.db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if outboxCount(t, m) != 1 {
		t.Fatal("migration replay changed records")
	}
	var count int
	if err := m.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE NOT private AND type='outgoing'`); err != nil || count != 1 {
		t.Fatal("feedback retry created another customer reply")
	}
}

func TestManualResolutionFirstSentNotFirstQueuedAndAtomicCommit(t *testing.T) {
	m, cuuid, cid, original := manualFixture(t, "manual_feedback_order")
	manualNote(t, m, cid, original)
	first := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	second := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	tx, err := m.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Stmtx(m.q.UpdateMessageStatus).Exec("sent", second); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if outboxCount(t, m) != 0 {
		tx.Rollback()
		t.Fatal("uncommitted SENT escaped transaction")
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if outboxCount(t, m) != 0 {
		t.Fatal("rolled back send left outbox")
	}
	sent(t, m, second)
	sent(t, m, first)
	var chosen string
	if err = m.db.Get(&chosen, `SELECT outgoing_message_uuid FROM manual_resolution_feedback`); err != nil || chosen != second || outboxCount(t, m) != 1 {
		t.Fatal("first queued rather than first SENT chosen", err)
	}
}

func TestManualResolutionConcurrentSendsAndWorkers(t *testing.T) {
	m, cuuid, cid, original := manualFixture(t, "manual_feedback_concurrent")
	manualNote(t, m, cid, original)
	a := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	b := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, id := range []string{a, b} {
		wg.Add(1)
		go func(id string) { defer wg.Done(); _, err := m.q.UpdateMessageStatus.Exec("sent", id); errs <- err }(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if outboxCount(t, m) != 1 {
		t.Fatal("concurrent sends generated multiple feedback records")
	}
	// Hold the same cross-process worker lock: another instance must not submit.
	ctx := context.Background()
	conn, err := m.db.Connx(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(ctx, `SELECT pg_advisory_lock(719438252)`); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	probe := &manualFeedbackProbe{}
	m.deliverManualResolutionFeedback(ctx, probe)
	if len(probe.rows) != 0 {
		t.Fatal("parallel worker ignored lock")
	}
	conn.ExecContext(ctx, `SELECT pg_advisory_unlock(719438252)`)
	conn.Close()
	m.deliverManualResolutionFeedback(ctx, probe)
	if len(probe.rows) != 1 {
		t.Fatal("worker failed after lock release")
	}
}

func TestManualResolutionConservativeExclusions(t *testing.T) {
	for _, name := range []string{"no_note", "ordinary_note", "private", "incoming", "widget", "automated", "own", "ai", "ai_adapted", "new_input_before_queue", "new_input_before_sent", "deleted_note", "foreign_note", "note_after_queue", "deleted_original", "no_snapshot", "conflicting_source", "new_ai_analysis", "already_answered", "foreign_conversation"} {
		t.Run(name, func(t *testing.T) {
			m, cuuid, cid, original := manualFixture(t, "manual_skip_"+name)
			note := ""
			if name != "no_note" && name != "ordinary_note" && name != "note_after_queue" {
				note = manualNote(t, m, cid, original)
			}
			if name == "ordinary_note" {
				m.db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content) VALUES($1,'outgoing','sent',101,'agent',true,'Human needed HTML only')`, cid)
			}
			if name == "foreign_note" {
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{ai_analysis,original_message_uuid}',to_jsonb('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'::text)) WHERE uuid=$1`, note)
			}
			if name == "foreign_conversation" {
				var foreignID int
				if err := m.db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING id`).Scan(&foreignID); err != nil {
					t.Fatal(err)
				}
				m.db.MustExec(`UPDATE conversation_messages SET conversation_id=$2 WHERE uuid=$1`, note, foreignID)
			}
			if name == "new_input_before_queue" {
				input(t, m, cid)
			}
			if name == "already_answered" {
				old := queuedCompletion(t, m, cuuid, cid, map[string]any{})
				m.db.MustExec(`UPDATE conversation_messages SET meta=meta-'mail_completion',status='sent' WHERE uuid=$1`, old)
			}
			meta := map[string]any{}
			if name == "own" {
				meta["ai_feedback_original_message_uuid"] = original
			}
			if name == "ai" || name == "ai_adapted" {
				meta["ai_response"] = map[string]string{"response_source": name, "source_message_uuid": original}
			}
			if name == "automated" {
				meta["is_automated"] = true
			}
			// Automated sends have no completion state by design.
			var outgoing string
			if name == "automated" {
				if err := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta) VALUES($1,'outgoing','pending',101,'agent','automated','{"is_automated":true}') RETURNING uuid`, cid).Scan(&outgoing); err != nil {
					t.Fatal(err)
				}
			} else {
				outgoing = queuedCompletion(t, m, cuuid, cid, meta)
			}
			switch name {
			case "private":
				m.db.MustExec(`UPDATE conversation_messages SET private=true WHERE uuid=$1`, outgoing)
			case "incoming":
				m.db.MustExec(`UPDATE conversation_messages SET type='incoming',sender_type='contact' WHERE uuid=$1`, outgoing)
			case "widget":
				m.db.MustExec(`UPDATE inboxes SET channel='livechat' WHERE id=102`)
			case "new_input_before_sent":
				input(t, m, cid)
			case "deleted_note":
				m.db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"deleted_at":"now"}' WHERE uuid=$1`, note)
			case "note_after_queue":
				manualNote(t, m, cid, original)
			case "deleted_original":
				m.db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"deleted_at":"now"}' WHERE uuid=$1`, original)
			case "no_snapshot":
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{mail_completion,snapshot_ready}','false') WHERE uuid=$1`, outgoing)
			case "conflicting_source":
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{reply_to_message_uuid}',to_jsonb('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'::text)) WHERE uuid=$1`, outgoing)
			case "new_ai_analysis":
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{ai_analysis}',jsonb_build_object('kind','n8n_ai_analysis','version',1,'original_message_uuid',$2::text,'suggested_message','Proposal')) WHERE uuid=$1`, note, original)
			}
			sent(t, m, outgoing)
			probe := &manualFeedbackProbe{}
			m.deliverManualResolutionFeedback(context.Background(), probe)
			if outboxCount(t, m) != 0 || len(probe.rows) != 0 {
				t.Fatal("unsafe association generated manual feedback")
			}
		})
	}
}

// Automated acknowledgements must not consume the first genuine employee reply.
func TestManualResolutionIgnoresAutomatedAcknowledgements(t *testing.T) {
	m, cuuid, cid, original := manualFixture(t, "manual_automated_ack")
	manualNote(t, m, cid, original)
	m.db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta)
 VALUES($1,'outgoing','sent',101,'agent','ack','{"is_automated":true}')`, cid)
	outgoing := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	sent(t, m, outgoing)
	if outboxCount(t, m) != 1 {
		t.Fatal("automated acknowledgement consumed employee feedback")
	}
}

// The incoming version clock is also used during SENT eligibility. An input
// already holding the clock must commit before this reply can be classified.
func TestManualResolutionConcurrentIncoming(t *testing.T) {
	m, cuuid, cid, original := manualFixture(t, "manual_concurrent_input")
	manualNote(t, m, cid, original)
	outgoing := queuedCompletion(t, m, cuuid, cid, map[string]any{})
	tx, err := m.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content)
 VALUES($1,'incoming','received',105,'contact','New input')`, cid); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, e := m.q.UpdateMessageStatus.Exec("sent", outgoing); done <- e }()
	select {
	case e := <-done:
		t.Fatalf("SENT eligibility did not wait for input: %v", e)
	case <-time.After(100 * time.Millisecond):
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SENT stuck after input commit")
	}
	if outboxCount(t, m) != 0 {
		t.Fatal("concurrent newer input generated false feedback")
	}
}

func TestManualResolutionInactiveWebhook(t *testing.T) {
	// A disabled existing ai_feedback client starts no loop or DB/SMTP work.
	m := &Manager{}
	m.RunManualResolutionFeedback(context.Background(), nil)
}
