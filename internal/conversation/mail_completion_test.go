package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/google/uuid"
)

type completionInboxGuard struct {
	inboxStore
	t *testing.T
}

func (g completionInboxGuard) Get(int) (inbox.Inbox, error) {
	g.t.Error("completion attempted SMTP access")
	return nil, errors.New("SMTP forbidden")
}
func completionFixture(t *testing.T, name string) (*Manager, string, int) {
	t.Helper()
	db := testutil.NewPersonalDB(t, name)
	m := &Manager{db: db, inboxStore: completionInboxGuard{t: t}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	var cuuid string
	var cid int
	if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid,id`).Scan(&cuuid, &cid); err != nil {
		t.Fatal(err)
	}
	return m, cuuid, cid
}
func input(t *testing.T, m *Manager, cid int) string {
	t.Helper()
	var id string
	if err := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,text_content) VALUES($1,'incoming','sent',105,'contact','Customer input','Customer input') RETURNING uuid`, cid).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
func queuedCompletion(t *testing.T, m *Manager, cuuid string, cid int, meta map[string]any) string {
	t.Helper()
	state, err := m.prepareMailCompletion(cuuid, 101, meta)
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("missing completion")
	}
	state.SnapshotReady = true
	state.SentMessage = "<p>Actual rendered transport body</p>"
	meta["mail_completion"] = state
	raw, _ := json.Marshal(meta)
	var id string
	if err := m.db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,meta) VALUES($1,'outgoing','pending',101,'agent','unrendered',$2::jsonb) RETURNING uuid`, cid, string(raw)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
func sent(t *testing.T, m *Manager, id string) {
	t.Helper()
	if _, err := m.q.UpdateMessageStatus.Exec("sent", id); err != nil {
		t.Fatal(err)
	}
}
func stateOf(t *testing.T, m *Manager, id string) MailCompletion {
	t.Helper()
	var raw []byte
	if err := m.db.Get(&raw, `SELECT meta->'mail_completion' FROM conversation_messages WHERE uuid=$1`, id); err != nil {
		t.Fatal(err)
	}
	var s MailCompletion
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestMailCompletionAllSourcesAndRetry(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_sources")
	source := input(t, m, cid)
	for _, kind := range []string{"ai", "ai_adapted", "human", ""} {
		t.Run(kind, func(t *testing.T) {
			meta := map[string]any{"reply_to_message_uuid": source}
			if kind != "" {
				meta["ai_response"] = map[string]string{"response_source": kind, "source_message_uuid": source}
				if kind == "human" {
					meta["ai_feedback_original_message_uuid"] = source
				}
			}
			out := queuedCompletion(t, m, c, cid, meta)
			probe := &provenanceProbe{fail: true}
			m.deliverAIProvenance(context.Background(), probe)
			if probe.calls != 0 {
				t.Fatal("pending emitted")
			}
			if _, err := m.q.UpdateMessageStatus.Exec("failed", out); err != nil {
				t.Fatal(err)
			}
			m.deliverAIProvenance(context.Background(), probe)
			if probe.calls != 0 {
				t.Fatal("failed emitted")
			}
			sent(t, m, out)
			m.deliverAIProvenance(context.Background(), probe)
			if probe.calls != 1 {
				t.Fatal("missing first attempt")
			}
			probe.fail = false
			m.deliverAIProvenance(context.Background(), probe)
			m.deliverAIProvenance(context.Background(), probe)
			if probe.calls != 2 {
				t.Fatal("commit repeated")
			}
			if kind == "" {
				if probe.payload["response_source"] != nil {
					t.Fatal("normal reply classified")
				}
			} else if probe.payload["response_source"] != kind {
				t.Fatal(probe.payload)
			}
			if probe.payload["source_message_uuid"] != source || probe.payload["outgoing_message_uuid"] != out || probe.payload["agent_id"] != "101" || probe.payload["sent_message"] != "<p>Actual rendered transport body</p>" || probe.payload["expected_incoming_version"] != "1" {
				t.Fatal(probe.payload)
			}
			if !stateOf(t, m, out).Committed {
				t.Fatal("missing durable commit")
			}
		})
	}
	var count int
	m.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND type='outgoing'`, cid)
	if count != 4 {
		t.Fatal("duplicate email")
	}
}
func TestMailCompletionAmbiguityAndSourceValidation(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_mapping")
	a := input(t, m, cid)
	only, err := m.prepareMailCompletion(c, 101, map[string]any{})
	if err != nil || only.SourceUUID == nil || *only.SourceUUID != a {
		t.Fatal("single source", err)
	}
	b := input(t, m, cid)
	unknown, err := m.prepareMailCompletion(c, 101, map[string]any{})
	if err != nil || unknown.SourceUUID != nil || unknown.ResponseSource != nil {
		t.Fatal("guessed latest source", err)
	}
	otherConv := uuid.NewString()
	m.db.MustExec(`INSERT INTO conversations(uuid,contact_id,inbox_id,status_id) VALUES($1,105,102,101)`, otherConv)
	var otherID int
	m.db.Get(&otherID, `SELECT id FROM conversations WHERE uuid=$1`, otherConv)
	foreign := input(t, m, otherID)
	for _, src := range []string{foreign, uuid.NewString()} {
		if _, err := m.prepareMailCompletion(c, 101, map[string]any{"reply_to_message_uuid": src}); !errors.Is(err, ErrReplySource) {
			t.Fatal("invalid source accepted", err)
		}
	}
	if _, err := m.prepareMailCompletion(c, 101, map[string]any{"reply_to_message_uuid": a, "ai_response": map[string]string{"response_source": "ai", "source_message_uuid": b}}); !errors.Is(err, ErrReplySource) {
		t.Fatal("conflicting sources", err)
	}
	out := queuedCompletion(t, m, c, cid, map[string]any{})
	sent(t, m, out)
	probe := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 0 {
		t.Fatal("unassigned event delivered")
	}
	future := input(t, m, cid)
	if err := m.AssignReplySource(context.Background(), c, out, future); !errors.Is(err, ErrReplySource) {
		t.Fatal("future source accepted", err)
	}
	if err := m.AssignReplySource(context.Background(), c, out, b); err != nil {
		t.Fatal(err)
	}
	if err := m.AssignReplySource(context.Background(), c, out, b); err != nil {
		t.Fatal("idempotent assignment", err)
	}
	if err := m.AssignReplySource(context.Background(), c, out, a); !errors.Is(err, ErrReplySource) {
		t.Fatal("association changed", err)
	}
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 1 || probe.payload["source_message_uuid"] != b {
		t.Fatal("repair not delivered")
	}
}
func TestMailCompletionConditionalResolveAndReplay(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_resolve")
	a := input(t, m, cid)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	ctx := context.Background()
	if _, err := m.ResolveMailCompletion(ctx, c, out, "1"); !errors.Is(err, ErrCompletionNotSent) {
		t.Fatal("pending resolved", err)
	}
	sent(t, m, out)
	if _, err := m.ResolveMailCompletion(ctx, c, out, "999"); !errors.Is(err, ErrReplySource) {
		t.Fatal("client version trusted", err)
	}
	result, err := m.ResolveMailCompletion(ctx, c, out, "1")
	if err != nil || result.Outcome != "resolved" || result.Replayed {
		t.Fatal(result, err)
	}
	input(t, m, cid)
	var status string
	m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE c.uuid=$1`, c)
	if status != "Open" {
		t.Fatal("new input stayed resolved")
	}
	result, err = m.ResolveMailCompletion(ctx, c, out, "1")
	if err != nil || !result.Replayed {
		t.Fatal("lost response replay", result, err)
	}
	m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE c.uuid=$1`, c)
	if status != "Open" {
		t.Fatal("replay reclosed ticket")
	}
	out2 := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	sent(t, m, out2)
	for i := 0; i < 2; i++ {
		result, err = m.ResolveMailCompletion(ctx, c, out2, "1")
		if !errors.Is(err, ErrCompletionConflict) || result.Outcome != "superseded" || result.Replayed != (i == 1) {
			t.Fatal("stale resolve", result, err)
		}
	}
}

type syncProvenance struct {
	calls   int
	entered chan struct{}
	release chan struct{}
	resolve func()
}

func (p *syncProvenance) Call(_ context.Context, _ string, id string, _ any, result any) (int, error) {
	p.calls++
	if p.entered != nil {
		close(p.entered)
		<-p.release
	}
	if p.resolve != nil {
		p.resolve()
	}
	ack, _ := json.Marshal(map[string]any{"committed": true, "outgoing_message_uuid": id})
	return 200, json.Unmarshal(ack, result)
}
func TestMailCompletionWebhookConcurrencyAndResolveCallback(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_delivery_race")
	a := input(t, m, cid)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	sent(t, m, out)
	first := &syncProvenance{entered: make(chan struct{}), release: make(chan struct{})}
	done := make(chan struct{})
	go func() { m.deliverAIProvenance(context.Background(), first); close(done) }()
	<-first.entered
	second := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), second)
	if second.calls != 0 {
		t.Fatal("parallel duplicate event")
	}
	close(first.release)
	<-done
	// Force a lost ack without touching SENT or invoking SMTP.
	m.db.MustExec(`UPDATE conversation_messages SET meta=meta #- '{mail_completion,committed}' WHERE uuid=$1`, out)
	caller := &syncProvenance{resolve: func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, err := m.ResolveMailCompletion(ctx, c, out, "1"); err != nil {
			t.Error("webhook held row lock over HTTP", err)
		}
	}}
	m.deliverAIProvenance(context.Background(), caller)
	if !stateOf(t, m, out).Committed || stateOf(t, m, out).ResolveResult != "resolved" {
		t.Fatal("delivery clobbered resolve result")
	}
}
func TestMailCompletionInputWinsWhileResolveWaits(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_input_race")
	a := input(t, m, cid)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	sent(t, m, out)
	tx, err := m.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	// Trigger holds the input clock until this transaction commits.
	if _, err = tx.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','sent',105,'contact','parallel input')`, cid); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, e := m.ResolveMailCompletion(context.Background(), c, out, "1"); result <- e }()
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if e := <-result; !errors.Is(e, ErrCompletionConflict) {
		t.Fatal("input race resolved", e)
	}
}
func TestMailCompletionResolveWinsBeforeConcurrentInput(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_resolve_race")
	a := input(t, m, cid)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	sent(t, m, out)
	blocker, err := m.db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback()
	var id int
	if err = blocker.Get(&id, `SELECT id FROM conversations WHERE id=$1 FOR UPDATE`, cid); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resolved := make(chan error, 1)
	go func() { _, e := m.ResolveMailCompletion(ctx, c, out, "1"); resolved <- e }()
	// Wait for the real resolver to own the input clock and wait on our conversation row.
	for {
		var waiting bool
		err = m.db.Get(&waiting, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'SELECT s.name FROM conversations%')`)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("resolver did not reach lock", ctx.Err())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	incoming := make(chan error, 1)
	go func() {
		_, e := m.db.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','sent',105,'contact','arrives during resolve')`, cid)
		incoming <- e
	}()
	if err = blocker.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = <-resolved; err != nil {
		t.Fatal(err)
	}
	if err = <-incoming; err != nil {
		t.Fatal(err)
	}
	var status string
	m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE c.id=$1`, cid)
	if status != "Open" {
		t.Fatal("new input stayed resolved")
	}
}
func TestMailCompletionVersionRollbackAndFilters(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_filters")
	a := input(t, m, cid)
	tx, _ := m.db.Beginx()
	tx.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'incoming','sent',105,'contact','rolled back')`, cid)
	tx.Rollback()
	var version int
	m.db.Get(&version, `SELECT version FROM conversation_mail_versions WHERE conversation_id=$1`, cid)
	if version != 1 {
		t.Fatal("rolled back input changed version")
	}
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	sent(t, m, out)
	// Disabled endpoint means stored work remains available, without a mail retry.
	if stateOf(t, m, out).Committed {
		t.Fatal("inactive receiver marked committed")
	}
	m.db.MustExec(`UPDATE conversation_messages SET private=true WHERE uuid=$1`, out)
	probe := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 0 {
		t.Fatal("private note delivered")
	}
	m.db.MustExec(`UPDATE conversation_messages SET private=false,meta=meta||'{"is_automated":true}'::jsonb WHERE uuid=$1`, out)
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 0 {
		t.Fatal("automation delivered")
	}
	m.db.MustExec(`UPDATE conversation_messages SET meta=meta-'is_automated' WHERE uuid=$1`, out)
	m.db.MustExec(`UPDATE inboxes SET channel='livechat' WHERE id=102`)
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 0 {
		t.Fatal("widget delivered")
	}
}

func TestMailCompletionTemporaryResolveFailureOnlyRetriesCompletion(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_temporary")
	a := input(t, m, cid)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": a})
	sent(t, m, out)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.ResolveMailCompletion(ctx, c, out, "1"); err == nil {
		t.Fatal("canceled DB operation succeeded")
	}
	if stateOf(t, m, out).ResolveResult != "" {
		t.Fatal("failed operation became terminal")
	}
	if _, err := m.ResolveMailCompletion(context.Background(), c, out, "1"); err != nil {
		t.Fatal(err)
	}
	var count int
	m.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND type='outgoing'`, cid)
	if count != 1 {
		t.Fatal("second SMTP message queued")
	}
}
func TestMailCompletionLegacyWithoutSnapshotIsNotFabricated(t *testing.T) {
	m, _, cid := completionFixture(t, "completion_no_snapshot")
	input(t, m, cid)
	m.db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content) VALUES($1,'outgoing','sent',101,'agent','unrendered historic draft')`, cid)
	probe := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 0 {
		t.Fatal("historic transport body invented")
	}
}
func TestMailCompletionEmptyRenderedBody(t *testing.T) {
	m, c, cid := completionFixture(t, "completion_empty_body")
	source := input(t, m, cid)
	out := queuedCompletion(t, m, c, cid, map[string]any{"reply_to_message_uuid": source})
	m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{mail_completion,sent_message}','""'::jsonb) WHERE uuid=$1`, out)
	sent(t, m, out)
	probe := &provenanceProbe{}
	m.deliverAIProvenance(context.Background(), probe)
	if probe.calls != 1 || probe.payload["sent_message"] != "" {
		t.Fatal("empty attachment-only body lost")
	}
}
