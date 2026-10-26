package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/migrations"
)

func setReason(t *testing.T, m *Manager, c, n string, agent int, text string) ManualReason {
	t.Helper()
	r, e := m.ManualReviewFeedback(context.Background(), c, n, agent, &text)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func getReason(t *testing.T, m *Manager, c, n string, agent int) ManualReason {
	t.Helper()
	r, e := m.ManualReviewFeedback(context.Background(), c, n, agent, nil)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestManualReasonNormalization(t *testing.T) {
	for _, value := range []string{" \r\nhello\rworld \n", strings.Repeat("🙂", 2500), "<img src=x onerror=alert(1)>"} {
		if _, e := NormalizeManualReason(value); e != nil {
			t.Fatal(e)
		}
	}
	value, e := NormalizeManualReason(" \r\nhello\rworld \n")
	if e != nil || value != "hello\nworld" {
		t.Fatal(value, e)
	}
	for _, value := range []string{strings.Repeat("🙂", 2501), "x\x00y", string([]byte{0xff})} {
		if _, e := NormalizeManualReason(value); !errors.Is(e, ErrManualReasonInput) {
			t.Fatal("invalid reason accepted")
		}
	}
}
func TestManualReasonCRUDSnapshotAndRetry(t *testing.T) {
	m, c, cid, original := manualFixture(t, "manual_reason_crud")
	n := manualNote(t, m, cid, original)
	duplicate := manualNote(t, m, cid, original)
	if r := getReason(t, m, c, n, 101); r.Reason != nil || !r.Editable {
		t.Fatal(r)
	}
	reason := "Kurze Antwort und Entschuldigung"
	setReason(t, m, c, n, 101, " "+reason+"\r\n")
	setReason(t, m, c, n, 101, reason)
	if r := getReason(t, m, c, duplicate, 101); r.Reason == nil || *r.Reason != reason {
		t.Fatal("duplicate notes lost annotation")
	}
	if r := getReason(t, m, c, n, 102); r.Reason != nil {
		t.Fatal("another employee's reason leaked")
	}
	var count int
	m.db.Get(&count, `SELECT count(*) FROM manual_review_annotations`)
	if count != 1 {
		t.Fatal("non-idempotent save")
	}
	if r := setReason(t, m, c, n, 101, ""); r.Reason != nil {
		t.Fatal("delete failed")
	}
	setReason(t, m, c, n, 101, reason)
	probe := &manualFeedbackProbe{}
	m.deliverManualResolutionFeedback(context.Background(), probe)
	if outboxCount(t, m) != 0 || len(probe.rows) != 0 {
		t.Fatal("annotation submitted feedback before send")
	}
	outgoing := queuedCompletion(t, m, c, cid, map[string]any{})
	// Editing remains possible during pending/failed transport, until SENT commits.
	m.q.UpdateMessageStatus.Exec("failed", outgoing)
	setReason(t, m, c, duplicate, 101, reason+"!")
	if outboxCount(t, m) != 0 {
		t.Fatal("failed generated feedback")
	}
	sent(t, m, outgoing)
	r := getReason(t, m, c, n, 101)
	if r.Editable {
		t.Fatal("SENT still editable")
	}
	changed := "too late"
	if _, e := m.ManualReviewFeedback(context.Background(), c, n, 101, &changed); !errors.Is(e, ErrManualReasonLocked) {
		t.Fatal(e)
	}
	if _, e := m.ManualReviewFeedback(context.Background(), c, n, 101, ptr("")); !errors.Is(e, ErrManualReasonLocked) {
		t.Fatal("late deletion allowed")
	}
	if _, e := m.db.Exec(`UPDATE manual_resolution_feedback SET feedback_reason='tampered'`); e == nil {
		t.Fatal("snapshot mutable")
	}
	probe.fail = true
	m.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 1 || probe.rows[0].FeedbackReason == nil || *probe.rows[0].FeedbackReason != reason+"!" || probe.rows[0].HumanAnswer != "<p>Actual rendered transport body</p>" {
		t.Fatal("wrong snapshot")
	}
	// Restart and retry retain the exact payload; no SMTP implementation available.
	m.db.MustExec(`UPDATE manual_resolution_feedback SET next_attempt_at=NOW()`)
	restarted := &Manager{db: m.db, lo: m.lo, inboxStore: completionInboxGuard{t: t}}
	probe.fail = false
	restarted.deliverManualResolutionFeedback(context.Background(), probe)
	restarted.deliverManualResolutionFeedback(context.Background(), probe)
	if len(probe.rows) != 2 || !reflect.DeepEqual(probe.rows[0], probe.rows[1]) || outboxCount(t, m) != 1 {
		t.Fatal("retry mutated or duplicated feedback")
	}
	if e := migrations.V2_16_0(m.db, nil, nil); e != nil {
		t.Fatal(e)
	}
	var saved string
	if e := m.db.Get(&saved, `SELECT feedback_reason FROM manual_resolution_feedback`); e != nil || saved != reason+"!" {
		t.Fatal("migration replay lost data", e)
	}
}
func TestManualReasonInvalidNotesAndStaleSources(t *testing.T) {
	for _, kind := range []string{"foreign", "deleted", "public", "incoming", "ordinary", "ai", "version", "fake", "stale", "conflict", "agent"} {
		t.Run(kind, func(t *testing.T) {
			m, c, cid, original := manualFixture(t, "reason_invalid_"+kind)
			n := manualNote(t, m, cid, original)
			switch kind {
			case "foreign":
				var foreign string
				m.db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`).Scan(&foreign)
				c = foreign
			case "deleted":
				m.db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"deleted_at":"now"}' WHERE uuid=$1`, n)
			case "public":
				m.db.MustExec(`UPDATE conversation_messages SET private=false WHERE uuid=$1`, n)
			case "incoming":
				m.db.MustExec(`UPDATE conversation_messages SET type='incoming' WHERE uuid=$1`, n)
			case "ordinary":
				m.db.MustExec(`UPDATE conversation_messages SET meta='{}' WHERE uuid=$1`, n)
			case "ai":
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{ai_analysis,kind}','"n8n_ai_analysis"') WHERE uuid=$1`, n)
			case "version":
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{ai_analysis,version}','2') WHERE uuid=$1`, n)
			case "fake":
				n = "00000000-0000-4000-8000-000000000001"
			case "stale":
				input(t, m, cid)
			case "conflict":
				m.db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content,meta) VALUES($1,'outgoing','sent',101,'agent',true,'AI',$2::jsonb)`, cid, `{"ai_analysis":{"kind":"n8n_ai_analysis","version":1,"original_message_uuid":"`+original+`","suggested_message":"AI"}}`)
			}
			agent := 101
			if kind == "agent" {
				agent = 105
			}
			_, e := m.ManualReviewFeedback(context.Background(), c, n, agent, ptr("no"))
			if e == nil {
				t.Fatal("invalid annotation accepted")
			}
			var count int
			m.db.Get(&count, `SELECT count(*) FROM manual_review_annotations`)
			if count != 0 || outboxCount(t, m) != 0 {
				t.Fatal("invalid annotation persisted")
			}
		})
	}
}
func queuedActor(t *testing.T, m *Manager, c string, cid, agent int) string {
	t.Helper()
	meta := map[string]any{}
	state, e := m.prepareMailCompletion(c, agent, meta)
	if e != nil {
		t.Fatal(e)
	}
	state.SnapshotReady = true
	state.SentMessage = "<p>Actual body</p>"
	meta["mail_completion"] = state
	raw, _ := json.Marshal(meta)
	var id string
	if e = m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta) VALUES($1,'outgoing','pending',$2,'agent','reply',$3::jsonb) RETURNING uuid`, cid, agent, string(raw)).Scan(&id); e != nil {
		t.Fatal(e)
	}
	return id
}
func TestManualReasonTwoEmployeesAndNewInput(t *testing.T) {
	m, c, cid, original := manualFixture(t, "reason_two_agents")
	n := manualNote(t, m, cid, original)
	setReason(t, m, c, n, 101, "A")
	setReason(t, m, c, n, 102, "B")
	a := queuedActor(t, m, c, cid, 101)
	b := queuedActor(t, m, c, cid, 102)
	sent(t, m, b)
	sent(t, m, a)
	var reason string
	var agent int
	if e := m.db.QueryRow(`SELECT agent_id,feedback_reason FROM manual_resolution_feedback`).Scan(&agent, &reason); e != nil || reason != "B" || agent != 102 || outboxCount(t, m) != 1 {
		t.Fatal("cross-employee contamination", e)
	}
	newer := input(t, m, cid)
	n2 := manualNote(t, m, cid, newer)
	if r := getReason(t, m, c, n2, 101); r.Reason != nil {
		t.Fatal("reason jumped to new input")
	}
	out := queuedActor(t, m, c, cid, 101)
	sent(t, m, out)
	var nullReason *string
	if e := m.db.QueryRow(`SELECT feedback_reason FROM manual_resolution_feedback WHERE outgoing_message_uuid=$1`, out).Scan(&nullReason); e != nil || nullReason != nil {
		t.Fatal("normal null behavior changed", e)
	}
}
func TestManualReasonConcurrentSentSnapshot(t *testing.T) {
	m, c, cid, original := manualFixture(t, "reason_race_sent")
	n := manualNote(t, m, cid, original)
	setReason(t, m, c, n, 101, "before")
	out := queuedActor(t, m, c, cid, 101)
	tx, e := m.db.Beginx()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SELECT 1 FROM conversation_mail_versions WHERE conversation_id=$1 FOR UPDATE`, cid); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, err := m.q.UpdateMessageStatus.Exec("sent", out); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("SENT bypassed annotation lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	tx.MustExec(`UPDATE manual_review_annotations SET feedback_reason='atomic snapshot' WHERE original_message_uuid=$1`, original)
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SENT blocked")
	}
	var reason string
	m.db.Get(&reason, `SELECT feedback_reason FROM manual_resolution_feedback`)
	if reason != "atomic snapshot" {
		t.Fatal("lost concurrent annotation")
	}
	if _, e = m.ManualReviewFeedback(context.Background(), c, n, 101, ptr("late")); !errors.Is(e, ErrManualReasonLocked) {
		t.Fatal("late update accepted", e)
	}
}

func TestManualReasonParallelActors(t *testing.T) {
	m, c, cid, original := manualFixture(t, "reason_parallel_actors")
	n := manualNote(t, m, cid, original)
	setReason(t, m, c, n, 101, "agent A")
	setReason(t, m, c, n, 102, "agent B")
	a := queuedActor(t, m, c, cid, 101)
	b := queuedActor(t, m, c, cid, 102)
	done := make(chan error, 2)
	for _, out := range []string{a, b} {
		go func(id string) { _, e := m.q.UpdateMessageStatus.Exec("sent", id); done <- e }(out)
	}
	for i := 0; i < 2; i++ {
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}
	var agent int
	var reason string
	if e := m.db.QueryRow(`SELECT agent_id,feedback_reason FROM manual_resolution_feedback`).Scan(&agent, &reason); e != nil {
		t.Fatal(e)
	}
	if outboxCount(t, m) != 1 || (agent == 101 && reason != "agent A") || (agent == 102 && reason != "agent B") {
		t.Fatal("parallel actor mix-up")
	}
}

func TestManualReasonDoesNotBypassSendExclusions(t *testing.T) {
	for _, kind := range []string{"pending", "failed", "private", "automated", "new_input", "contradiction"} {
		t.Run(kind, func(t *testing.T) {
			m, c, cid, original := manualFixture(t, "reason_no_feedback_"+kind)
			n := manualNote(t, m, cid, original)
			setReason(t, m, c, n, 101, "Do not leak to another send")
			out := queuedActor(t, m, c, cid, 101)
			switch kind {
			case "failed":
				m.q.UpdateMessageStatus.Exec("failed", out)
			case "private":
				m.db.MustExec(`UPDATE conversation_messages SET private=true WHERE uuid=$1`, out)
			case "automated":
				m.db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"is_automated":true}' WHERE uuid=$1`, out)
			case "new_input":
				input(t, m, cid)
			case "contradiction":
				m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{ai_analysis,kind}','"n8n_ai_analysis"') WHERE uuid=$1`, n)
			}
			if kind != "pending" && kind != "failed" {
				sent(t, m, out)
			}
			probe := &manualFeedbackProbe{}
			m.deliverManualResolutionFeedback(context.Background(), probe)
			if outboxCount(t, m) != 0 || len(probe.rows) != 0 {
				t.Fatal("annotation bypassed send eligibility")
			}
		})
	}
}
func TestManualReasonAPIRacesSent(t *testing.T) {
	m, c, cid, original := manualFixture(t, "reason_api_race")
	n := manualNote(t, m, cid, original)
	setReason(t, m, c, n, 101, "before")
	out := queuedActor(t, m, c, cid, 101)
	start := make(chan struct{})
	saved := make(chan error, 1)
	sentResult := make(chan error, 1)
	go func() {
		<-start
		_, e := m.ManualReviewFeedback(context.Background(), c, n, 101, ptr("race"))
		saved <- e
	}()
	go func() { <-start; _, e := m.q.UpdateMessageStatus.Exec("sent", out); sentResult <- e }()
	close(start)
	saveErr := <-saved
	if e := <-sentResult; e != nil {
		t.Fatal(e)
	}
	var reason string
	if e := m.db.Get(&reason, `SELECT feedback_reason FROM manual_resolution_feedback`); e != nil {
		t.Fatal(e)
	}
	if saveErr == nil && reason != "race" {
		t.Fatal("acknowledged write lost")
	}
	if saveErr != nil && (!errors.Is(saveErr, ErrManualReasonLocked) || reason != "before") {
		t.Fatal("late write corrupted snapshot", saveErr)
	}
}
