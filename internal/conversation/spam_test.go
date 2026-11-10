package conversation

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
)

const spamViewFilter = `[{"model":"conversations","field":"tags","operator":"contains","value":"[12]"}]`

func TestSpamViewFilter(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{spamViewFilter, true},
		{`[{"model":"conversations","field":"tags","operator":"contains","value":"[112]"}]`, false},
		{`[{"model":"conversations","field":"tags","operator":"not contains","value":"[12]"}]`, false},
		{`{"logic":"OR","rules":[{"model":"conversations","field":"tags","operator":"contains","value":"[12]"}]}`, false},
		{`[]`, false}, {`broken`, false},
	} {
		if got := isSpamViewFilter(tc.raw); got != tc.want {
			t.Errorf("%s: %v", tc.raw, got)
		}
	}
}

func TestSpamLifecycle(t *testing.T) {
	m := emailThreadFixture(t, "spam_lifecycle")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam'),(42,'Rekla')`)
	actor := umodels.User{ID: 101, FirstName: "Owner"}
	for _, start := range []string{"Open", "Closed", "Snoozed", "Resolved"} {
		t.Run(start, func(t *testing.T) {
			var uuid string
			if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) SELECT 105,102,id,'{"keep":"yes"}' FROM conversation_statuses WHERE name=$1 RETURNING uuid`, start); err != nil {
				t.Fatal(err)
			}
			var before string
			if err := m.db.Get(&before, `SELECT jsonb_build_object('closed_at',closed_at,'resolved_at',resolved_at)::text FROM conversations WHERE uuid=$1`, uuid); err != nil {
				t.Fatal(err)
			}
			if err := m.SetConversationTags(uuid, amodels.ActionAddTags, []string{"🗑Spam", "Rekla"}, actor); err != nil {
				t.Fatal(err)
			}
			var status string
			m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE uuid=$1`, uuid)
			if status != "Closed" {
				t.Fatalf("got %s", status)
			}
			var closedAt string
			m.db.Get(&closedAt, `SELECT closed_at::text FROM conversations WHERE uuid=$1`, uuid)
			if _, _, changed, err := m.mutateTagsWithSpam(uuid, amodels.ActionAddTags, []string{"🗑Spam"}, actor); err != nil || changed {
				t.Fatalf("duplicate close: %v %v", changed, err)
			}
			var again string
			m.db.Get(&again, `SELECT closed_at::text FROM conversations WHERE uuid=$1`, uuid)
			if again != closedAt {
				t.Fatal("duplicate changed closed_at")
			}
			// Status API must refuse reopening before touching SLA/automation stores.
			if err := m.UpdateConversationStatus(uuid, 0, "Open", "", actor); err == nil {
				t.Fatal("allowed opening Spam")
			}
			if err := m.ReOpenConversation(uuid, actor); err != nil {
				t.Fatal(err)
			}
			if err := m.ProcessIncomingMessageHooks(uuid, false); err != nil {
				t.Fatal(err)
			}
			// Stale custom attribute writes cannot delete or forge restoration state.
			if err := m.UpdateConversationCustomAttributes(uuid, map[string]any{"keep": "yes", "_spam_previous_state": 999}); err != nil {
				t.Fatal(err)
			}
			if err := m.SetConversationTags(uuid, amodels.ActionRemoveTags, []string{"🗑Spam"}, actor); err != nil {
				t.Fatal(err)
			}
			m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE uuid=$1`, uuid)
			var after string
			if err := m.db.Get(&after, `SELECT jsonb_build_object('closed_at',closed_at,'resolved_at',resolved_at)::text FROM conversations WHERE uuid=$1`, uuid); err != nil {
				t.Fatal(err)
			}
			if before != after {
				t.Fatalf("lifecycle timestamps changed: %s -> %s", before, after)
			}
			want := start
			if start == "Snoozed" {
				want = "Open"
			}
			if status != want {
				t.Fatalf("restored %s want %s", status, want)
			}
			tags, err := m.getConversationTags(uuid)
			if err != nil || len(tags) != 1 || tags[0] != "Rekla" {
				t.Fatalf("tags: %v %v", tags, err)
			}
			var count int
			m.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=(SELECT id FROM conversations WHERE uuid=$1) AND type='outgoing'`, uuid)
			if count != 0 {
				t.Fatal("spam sent an outgoing message")
			}
		})
	}
}

func TestSpamRollbackAndConcurrentChanges(t *testing.T) {
	m := emailThreadFixture(t, "spam_atomic")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	var uuid string
	m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`)
	actor := umodels.User{ID: 101}
	// Force the second half of the transition to fail: neither tag nor snapshot may remain.
	m.db.MustExec(`CREATE FUNCTION reject_spam_close() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status_id<>OLD.status_id THEN RAISE EXCEPTION 'test close failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_spam_close BEFORE UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION reject_spam_close()`)
	if _, _, _, err := m.mutateTagsWithSpam(uuid, amodels.ActionAddTags, []string{"🗑Spam"}, actor); err == nil {
		t.Fatal("expected transaction failure")
	}
	spam, err := conversationHasSpam(m.db, uuid)
	if err != nil || spam {
		t.Fatalf("tag survived rollback: %v %v", spam, err)
	}
	var attrs string
	m.db.Get(&attrs, `SELECT custom_attributes::text FROM conversations WHERE uuid=$1`, uuid)
	var a map[string]any
	json.Unmarshal([]byte(attrs), &a)
	if _, ok := a["_spam_previous_state"]; ok {
		t.Fatal("snapshot survived rollback")
	}
	m.db.MustExec(`DROP TRIGGER reject_spam_close ON conversations`)
	for i := 0; i < 12; i++ {
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, _, _, err := m.mutateTagsWithSpam(uuid, amodels.ActionAddTags, []string{"🗑Spam"}, actor)
			errs <- err
		}()
		go func() {
			defer wg.Done()
			err := m.UpdateConversationStatus(uuid, 0, "Open", "", actor)
			var input envelope.Error
			if errors.As(err, &input) && input.ErrorType == envelope.InputError {
				err = nil
			}
			errs <- err
		}()
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		var status string
		m.db.Get(&status, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id=c.status_id WHERE uuid=$1`, uuid)
		if status != "Closed" {
			t.Fatalf("concurrent reopen won: %s", status)
		}
		_, _, _, err := m.mutateTagsWithSpam(uuid, amodels.ActionRemoveTags, []string{"🗑Spam"}, actor)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSpamListsCountsPaginationAndFolders(t *testing.T) {
	m := emailThreadFixture(t, "spam_lists")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam'),(42,'Rekla'); INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(101,101) ON CONFLICT DO NOTHING; INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id) VALUES(103,'Other private','email','personal',102)`)
	for _, inbox := range []int{101, 102, 103} {
		for i := 0; i < 5; i++ {
			var id int
			m.db.Get(&id, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,$1,101,$2::jsonb) RETURNING id`, inbox, fmt.Sprintf(`{"customer_visibility":true,"access_mode":"%s"}`, map[bool]string{true: "personal", false: "public"}[inbox != 102]))
			m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,42)`, id)
			if i < 3 {
				m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,12)`, id)
			}
		}
	}
	// Spam includes both legacy Open and Closed records.
	m.db.MustExec(`UPDATE conversations SET status_id=(SELECT id FROM conversation_statuses WHERE name='Closed') WHERE id=(SELECT min(conversation_id) FROM conversation_tags WHERE tag_id=12)`)
	for _, tc := range []struct {
		list  string
		inbox int
		spam  bool
		total int
	}{
		{models.CustomerConversations, 0, false, 2}, {models.CustomerConversations, 0, true, 3},
		{models.PersonalConversations, 101, false, 2}, {models.PersonalConversations, 101, true, 3},
		{models.PersonalConversations, 103, true, 0}, {models.AllConversations, 0, false, 4},
		{models.AllConversations, 0, true, 0},
	} {
		filter := "[]"
		if tc.spam {
			filter = spamViewFilter
		}
		var inboxes []int
		if tc.inbox != 0 {
			inboxes = []int{tc.inbox}
		}
		for page := 1; page <= 2; page++ {
			query, args, err := m.makeConversationsListQuery(101, 101, nil, []string{tc.list}, m.q.GetConversations, "", "", page, 1, filter, false, true, inboxes...)
			if err != nil {
				t.Fatal(err)
			}
			var rows []models.ConversationListItem
			if err = m.db.Select(&rows, query, args...); err != nil {
				t.Fatal(err)
			}
			want := 1
			if tc.total == 0 {
				want = 0
			}
			if len(rows) != want {
				t.Fatalf("%+v rows %d", tc, len(rows))
			}
			for _, row := range rows {
				if row.Total != tc.total {
					t.Fatalf("%+v total %d", tc, row.Total)
				}
			}
		}
	}
	folders, err := m.MutateSidebarFolder(104, "customer", 0, 0, "create", SidebarFolder{Name: "Rekla", Filters: json.RawMessage(`[{"model":"conversations","field":"tags","operator":"contains","value":"[42]"}]`), ShowCount: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := m.SidebarFolderCounts(104, "customer", 0)
	if err != nil {
		t.Fatal(err)
	}
	if counts[folders[0].ID] != 2 {
		t.Fatalf("Spam entered folder count: %v", counts)
	}
	rows, err := m.GetSidebarFolderConversations(104, "customer", 0, folders[0].ID, "", "", 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Total != 2 {
		t.Fatalf("folder pagination: %v", rows)
	}
	rows, err = m.GetConversations(104, 104, nil, []string{models.CustomerConversations}, "", "", spamViewFilter, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("folder stole spam: %d", len(rows))
	}
}

func TestSpamIncomingCustomerReplyIsPersisted(t *testing.T) {
	m := emailThreadFixture(t, "spam_incoming")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	first := importMail(t, m, threadMail("spam-first@example.test", ""))
	if err := m.SetConversationTags(first.ConversationUUID, amodels.ActionAddTags, []string{"🗑Spam"}, umodels.User{ID: 101}); err != nil {
		t.Fatal(err)
	}
	convBefore, err := m.GetConversation(first.ConversationID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.ApplyAction(amodels.RuleAction{Type: amodels.ActionReply, Value: []string{"automatic reply"}}, convBefore, umodels.User{ID: 101}); err != nil {
		t.Fatal(err)
	}
	// No CSAT dependency is configured: a missing guard would panic here.
	if err := m.SendCSATReply(101, convBefore); err != nil {
		t.Fatal(err)
	}
	second := importMail(t, m, threadMail("spam-second@example.test", "spam-first@example.test"))
	if second.ConversationID != first.ConversationID {
		t.Fatal("reply moved to a different conversation")
	}
	conv, err := m.GetConversation(first.ConversationID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if conv.Status.String != "Closed" || conv.InboxID != 102 {
		t.Fatalf("reply changed spam status/inbox: %s %d", conv.Status.String, conv.InboxID)
	}
	spam, err := conversationHasSpam(m.db, conv.UUID)
	if err != nil || !spam {
		t.Fatal("reply lost Spam tag")
	}
	var count int
	if err := m.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND type='incoming'`, first.ConversationID); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("incoming messages: %d", count)
	}
}

func TestSpamWebSocketMembershipRespectsMailboxAccess(t *testing.T) {
	m := emailThreadFixture(t, "spam_ws")
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam'); INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(101,101),(101,102) ON CONFLICT DO NOTHING`)
	var uuid string
	if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,101,101,'{"access_mode":"personal"}') RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	clients := map[int]*ws.Client{}
	for _, id := range []int{101, 102, 103, 104} {
		client := &ws.Client{ID: id, Hub: m.wsHub, Send: make(chan wsmodels.WSMessage, 10)}
		m.wsHub.AddClient(client)
		clients[id] = client
		defer m.wsHub.RemoveClient(client)
	}
	// No list subscriptions: owners must learn of tickets newly entering Spam.
	m.broadcastTagMembershipUpdate(uuid, map[string]any{"tags": []string{"🗑Spam"}, "status": "Closed"})
	for id, client := range clients {
		want := id != 103
		select {
		case msg := <-client.Send:
			if !want {
				t.Fatalf("private spam event leaked to %d", id)
			}
			var event struct {
				Type string `json:"type"`
				Data struct {
					UUID string   `json:"uuid"`
					Tags []string `json:"tags"`
				} `json:"data"`
			}
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				t.Fatal(err)
			}
			if event.Type != "conversation_update" || event.Data.UUID != uuid || len(event.Data.Tags) != 1 {
				t.Fatalf("bad event %s", msg.Data)
			}
		default:
			if want {
				t.Fatalf("missing event for authorized employee %d", id)
			}
		}
	}
	// An empty authorized set must never invoke the hub's broadcast-to-all fallback.
	for _, id := range []int{101, 102, 104} {
		m.wsHub.RemoveClient(clients[id])
	}
	m.broadcastTagMembershipUpdate(uuid, map[string]any{"tags": []string{}})
	select {
	case msg := <-clients[103].Send:
		t.Fatalf("empty recipient set leaked event: %s", msg.Data)
	default:
	}
}
