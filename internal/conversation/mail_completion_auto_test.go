package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestAutomaticAnchorUsesInputVersionNotIDsOrTimestamps(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_auto_order")
	a := input(t, m, cid)
	var reserved int
	if e := m.db.Get(&reserved, `SELECT nextval(pg_get_serial_sequence('conversation_messages','id'))`); e != nil {
		t.Fatal(e)
	}
	b := input(t, m, cid)
	var latest string
	if e := m.db.QueryRow(`INSERT INTO conversation_messages(id,conversation_id,type,status,sender_id,sender_type,content,created_at) VALUES($1,$2,'incoming','sent',105,'contact','C','2000-01-01') RETURNING uuid`, reserved, cid).Scan(&latest); e != nil {
		t.Fatal(e)
	}
	// C has the oldest timestamp and an ID lower than B, but input version 3.
	var beforeA, beforeB []byte
	m.db.Get(&beforeA, `SELECT meta FROM conversation_messages WHERE uuid=$1`, a)
	m.db.Get(&beforeB, `SELECT meta FROM conversation_messages WHERE uuid=$1`, b)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	state := stateOf(t, m, out)
	if state.SourceUUID == nil || *state.SourceUUID != latest || state.ExpectedVersion != "3" || state.QueuedVersion != "3" || state.ResponseSource != nil {
		t.Fatal("C was not chosen", state)
	}
	sent(t, m, out)
	probe := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), probe)
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 1 || probe.payload["source_message_uuid"] != latest || probe.payload["response_source"] != nil {
		t.Fatal("wrong or duplicate anchor event", probe.payload)
	}
	var afterA, afterB []byte
	m.db.Get(&afterA, `SELECT meta FROM conversation_messages WHERE uuid=$1`, a)
	m.db.Get(&afterB, `SELECT meta FROM conversation_messages WHERE uuid=$1`, b)
	if string(beforeA) != string(afterA) || string(beforeB) != string(afterB) {
		t.Fatal("older customer messages modified")
	}
	// Numeric ordering must also prefer 10 over 9.
	for i := 4; i <= 10; i++ {
		latest = input(t, m, cid)
	}
	s, e := m.prepareMailCompletion(c, 101, map[string]any{})
	if e != nil || s.SourceUUID == nil || *s.SourceUUID != latest || s.ExpectedVersion != "10" || s.QueuedVersion != "10" {
		t.Fatal("versions sorted as strings", s, e)
	}
}
func TestExplicitSourcesSurviveNewerCustomerEmails(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_auto_explicit")
	a := input(t, m, cid)
	input(t, m, cid)
	for _, kind := range []string{"ai", "ai_adapted", "human", "feedback_only"} {
		meta := map[string]any{"reply_to_message_uuid": a}
		if kind != "feedback_only" {
			meta["ai_response"] = map[string]string{"response_source": kind, "source_message_uuid": a}
		}
		if kind == "human" || kind == "feedback_only" {
			meta["ai_feedback_original_message_uuid"] = a
		}
		state, e := m.prepareMailCompletion(c, 101, meta)
		want := kind
		if kind == "feedback_only" {
			want = "human"
		}
		if e != nil || state.SourceUUID == nil || *state.SourceUUID != a || state.ResponseSource == nil || *state.ResponseSource != want || state.ExpectedVersion != "1" || state.QueuedVersion != "2" {
			t.Fatal("explicit source replaced", kind, state, e)
		}
	}
}
func TestAutomaticAnchorWithNoCustomerEmail(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_auto_none")
	meta := map[string]any{"reply_to_message_uuid": "stale-draft-reference"}
	out := queuedCompletion(t, m, c, cid, meta)
	state := stateOf(t, m, out)
	if state.SourceUUID != nil || state.ResponseSource != nil || state.ExpectedVersion != "" || state.QueuedVersion != "0" || meta["reply_to_message_uuid"] != nil {
		t.Fatal("invented source", state)
	}
	sent(t, m, out)
	probe := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 0 {
		t.Fatal("unassigned event delivered")
	}
	if _, e := m.ResolveMailCompletion(context.Background(), c, out, "0"); !errors.Is(e, ErrCompletionNotSent) {
		t.Fatal("unsourced send resolved", e)
	}
}
func TestAutomaticAnchorSkipsIneligibleSources(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_auto_filter")
	a := input(t, m, cid)
	deleted := input(t, m, cid)
	m.db.MustExec(`UPDATE conversation_messages SET meta=meta||'{"deleted_at":"now"}'::jsonb WHERE uuid=$1`, deleted)
	var private, outgoing string
	if e := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content) VALUES($1,'incoming','sent',105,'contact',true,'private') RETURNING uuid`, cid).Scan(&private); e != nil {
		t.Fatal(e)
	}
	if e := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'outgoing','sent',101,'agent','reply') RETURNING uuid`, cid).Scan(&outgoing); e != nil {
		t.Fatal(e)
	}
	state, e := m.prepareMailCompletion(c, 101, map[string]any{})
	if e != nil || state.SourceUUID == nil || *state.SourceUUID != a || state.ExpectedVersion != "1" || state.QueuedVersion != "2" {
		t.Fatal(state, e)
	}
	for _, source := range []string{deleted, private, outgoing} {
		if _, e = m.prepareMailCompletion(c, 101, map[string]any{"ai_response": map[string]string{"response_source": "ai", "source_message_uuid": source}}); !errors.Is(e, ErrReplySource) {
			t.Fatal("ineligible explicit source accepted", source, e)
		}
	}
}
func TestAutomaticAnchorConcurrentInputsUseOneSnapshot(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_auto_concurrent")
	a := input(t, m, cid)
	tx, e := m.db.Beginx()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','sent',105,'contact','B holds clock')`, cid); e != nil {
		t.Fatal(e)
	}
	type incomingResult struct {
		uuid string
		err  error
	}
	done := make(chan incomingResult, 1)
	go func() {
		var id string
		err := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','sent',105,'contact','simultaneous C') RETURNING uuid`, cid).Scan(&id)
		done <- incomingResult{id, err}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		var waiting bool
		if e = m.db.Get(&waiting, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%simultaneous C%')`); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("second incoming did not wait", ctx.Err())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	state, e := m.prepareMailCompletion(c, 101, map[string]any{})
	if e != nil || state.SourceUUID == nil || *state.SourceUUID != a || state.ExpectedVersion != "1" || state.QueuedVersion != "1" {
		t.Fatal("mixed or uncommitted input state", state, e)
	}
	state.SnapshotReady = true
	state.SentMessage = "Rendered reply"
	raw, _ := json.Marshal(map[string]any{"mail_completion": state, "reply_to_message_uuid": a})
	var out string
	if e = m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta) VALUES($1,'outgoing','pending',101,'agent','draft',$2::jsonb) RETURNING uuid`, cid, string(raw)).Scan(&out); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	incoming := <-done
	if incoming.err != nil {
		t.Fatal(incoming.err)
	}
	latest, e := m.prepareMailCompletion(c, 101, map[string]any{})
	if e != nil || latest.SourceUUID == nil || *latest.SourceUUID != incoming.uuid || latest.ExpectedVersion != "3" || latest.QueuedVersion != "3" {
		t.Fatal("latest committed input missing", latest, e)
	}
	sent(t, m, out)
	if _, e = m.ResolveMailCompletion(context.Background(), c, out, "1"); !errors.Is(e, ErrCompletionConflict) {
		t.Fatal("new input did not prevent resolve", e)
	}
	result, e := m.ResolveMailCompletion(context.Background(), c, out, "1")
	if !errors.Is(e, ErrCompletionConflict) || !result.Replayed {
		t.Fatal("superseded was not terminal", result, e)
	}
}
