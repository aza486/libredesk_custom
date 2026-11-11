package conversation

import (
	"encoding/json"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/template"
)

func TestServiceEmailNormalization(t *testing.T) {
	for _, input := range []string{"", "  ", "example.com", "*.example.com", "Name <sender@example.com>", "@example.com", "a@example.com,b@example.com"} {
		if _, err := normalizeServiceEmail(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	if got, err := normalizeServiceEmail("  Sender@Example.COM  "); err != nil || got != "sender@example.com" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestServiceWorkflowRegistryQueryIntegration(t *testing.T) {
	m := emailThreadFixture(t, "service_workflow_query")
	raw, err := os.ReadFile("../../workflows/E-Mail Anonymisierung - Service Mail Guard.json")
	if err != nil {
		t.Fatal(err)
	}
	var workflow struct {
		Nodes []struct {
			Name       string
			Parameters struct{ Query string }
		}
	}
	if err = json.Unmarshal(raw, &workflow); err != nil {
		t.Fatal(err)
	}
	var query string
	for _, node := range workflow.Nodes {
		if node.Name == "Execute a SQL query2" {
			query = node.Parameters.Query
		}
	}
	if query == "" {
		t.Fatal("missing registry query")
	}
	var uuid, privateUUID string
	if err = m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	if err = m.db.Get(&privateUUID, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,101,101,'{"access_mode":"personal"}') RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	check := func(want bool) {
		t.Helper()
		for _, id := range []string{uuid, privateUUID} {
			var got bool
			if err := m.db.Get(&got, query, id); err != nil || got != want {
				t.Fatalf("workflow classifier: %v want %v, %v", got, want, err)
			}
		}
	}
	check(false)
	if err = m.AddServiceEmailAddress("  CONTACT@EXAMPLE.COM "); err != nil {
		t.Fatal(err)
	}
	check(true)
	m.db.MustExec(`UPDATE users SET email=E'\t CONTACT@EXAMPLE.COM \r\n' WHERE id=105`)
	check(true)
	if err = m.RemoveServiceEmailAddress("contact@example.com"); err != nil {
		t.Fatal(err)
	}
	check(false)
	var got bool
	if err = m.db.Get(&got, query, "bad-uuid'"); err == nil {
		t.Fatal("invalid UUID permitted")
	}
	m.db.MustExec(`ALTER TABLE service_email_addresses RENAME TO unavailable_registry`)
	if err = m.db.Get(&got, query, uuid); err == nil {
		t.Fatal("database outage treated as normal mail")
	}
}

func TestServiceEmailQueuedReplyAndRetryIntegration(t *testing.T) {
	m := emailThreadFixture(t, "service_queued_reply")
	incoming := importMail(t, m, threadMail("service-queue@example.test", ""))
	transport := &manualSendInbox{}
	m.inboxStore = manualSendInboxes{transport: transport}
	m.userStore = manualSendUsers{}
	var err error
	m.template, err = template.New(m.lo, m.db, nil, nil, nil, m.i18n)
	if err != nil {
		t.Fatal(err)
	}
	reply, err := m.QueueReply(nil, 102, 101, 105, incoming.ConversationUUID, "<p>Automatic reply</p>", []string{"contact@example.com"}, nil, nil, map[string]any{"automatic_email_ai": true})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.AddServiceEmailAddress("contact@example.com"); err != nil {
		t.Fatal(err)
	}
	m.sendOutgoingMessage(reply)
	if len(transport.sent) != 0 {
		t.Fatal("waiting automatic reply escaped service classification")
	}
	failed, err := m.GetMessage(reply.UUID)
	if err != nil || failed.Status != models.MessageStatusFailed {
		t.Fatalf("retry status: %s %v", failed.Status, err)
	}
	if err = m.MarkMessageAsPending(reply.UUID); err != nil {
		t.Fatal(err)
	}
	m.sendOutgoingMessage(failed)
	if len(transport.sent) != 0 {
		t.Fatal("retry bypassed classification")
	}
	if err = m.RemoveServiceEmailAddress("contact@example.com"); err != nil {
		t.Fatal(err)
	}
	if len(transport.sent) != 0 {
		t.Fatal("removal replayed historical work")
	}
	if err = m.MarkMessageAsPending(reply.UUID); err != nil {
		t.Fatal(err)
	}
	m.sendOutgoingMessage(failed)
	if len(transport.sent) != 1 {
		t.Fatal("explicit retry of ordinary mail failed")
	}
	// Older queued integration replies lack the new metadata marker.
	legacy, err := m.QueueReply(nil, 102, 101, 105, incoming.ConversationUUID, "Legacy", []string{"contact@example.com"}, nil, nil, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m.automatedAIAgentIDs = []int{101}
	if err = m.AddServiceEmailAddress("contact@example.com"); err != nil {
		t.Fatal(err)
	}
	m.sendOutgoingMessage(legacy)
	if len(transport.sent) != 1 {
		t.Fatal("pre-existing queue bypassed service check")
	}
	// Employee transport continues during a classification-store outage.
	m.automatedAIAgentIDs = nil
	m.db.MustExec(`ALTER TABLE service_email_addresses RENAME TO unavailable_service_registry`)
	m.sendOutgoingMessage(legacy)
	if len(transport.sent) != 2 {
		t.Fatal("manual reply was blocked by registry outage")
	}
}

func TestServiceEmailRegistryAndPublicationIntegration(t *testing.T) {
	m := emailThreadFixture(t, "service_registry")
	db := m.db
	var uuid, privateUUID, internalUUID string
	for _, row := range []struct {
		inbox int
		attrs string
		uuid  *string
	}{{102, `{"customer_visibility":true}`, &uuid}, {101, `{"access_mode":"personal"}`, &privateUUID}, {102, `{"private":true}`, &internalUUID}} {
		if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,$1,101,$2::jsonb) RETURNING uuid`, row.inbox, row.attrs).Scan(row.uuid); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.CheckAutomaticAIAllowed(uuid); err != nil {
		t.Fatal(err)
	}
	publication, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	if err := lockAutomaticAI(publication, uuid); err != nil {
		t.Fatal(err)
	}
	writer, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	var canWrite bool
	if err := writer.Get(&canWrite, `SELECT pg_try_advisory_xact_lock($1)`, serviceEmailLock); err != nil || canWrite {
		t.Fatalf("registry mutation raced publication: %v %v", canWrite, err)
	}
	writer.Rollback()
	publication.Rollback()
	// Simultaneous administration/sidebar requests use one authoritative transaction.
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m.AddServiceEmailAddress("  CONTACT@Example.com ") == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("duplicate successes: %d", successes.Load())
	}
	addresses, err := m.GetServiceEmailAddresses()
	if err != nil || !slices.Equal(addresses, []string{"contact@example.com"}) {
		t.Fatalf("registry: %v %v", addresses, err)
	}
	for _, c := range []struct {
		uuid   string
		tagged bool
	}{{uuid, true}, {privateUUID, false}, {internalUUID, false}} {
		var tagged bool
		if err := db.Get(&tagged, `SELECT EXISTS(SELECT 1 FROM conversation_tags ct JOIN conversations c ON c.id=ct.conversation_id WHERE c.uuid=$1 AND ct.tag_id=14)`, c.uuid); err != nil || tagged != c.tagged {
			t.Fatalf("classification: %s %v %v", c.uuid, tagged, err)
		}
		if err := m.CheckAutomaticAIAllowed(c.uuid); err == nil {
			t.Fatal("service AI allowed, including private mail")
		}
	}
	// All existing automatic note formats are checked in InsertMessage itself.
	for _, meta := range []json.RawMessage{json.RawMessage(`{"automatic_email_ai":true}`), json.RawMessage(`{"ai_analysis":{"kind":"n8n_ai_analysis"}}`), json.RawMessage(`{"ai_analysis":{"kind":"n8n_manual_review"}}`)} {
		if _, err := m.sendPrivateNote(nil, 101, uuid, "automatic note", nil, meta, false); err == nil {
			t.Fatal("service automatic note was published")
		}
	}
	// Ordinary manual notes stay usable even while classification storage is down.
	if _, err := m.sendPrivateNote(nil, 101, uuid, "manual employee note", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`ALTER TABLE service_email_addresses RENAME TO service_email_addresses_unavailable`)
	if err := m.CheckAutomaticAIAllowed(uuid); err == nil {
		t.Fatal("classification failure was treated as ordinary email")
	}
	if _, err := m.sendPrivateNote(nil, 101, uuid, "manual during outage", nil, nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.sendPrivateNote(nil, 101, uuid, "automatic during outage", nil, json.RawMessage(`{"automatic_email_ai":true}`), false); err == nil {
		t.Fatal("automatic note during outage")
	}
	db.MustExec(`ALTER TABLE service_email_addresses_unavailable RENAME TO service_email_addresses`)
	if err := m.RemoveServiceEmailAddress(" CONTACT@example.COM "); err != nil {
		t.Fatal(err)
	}
	if err := m.CheckAutomaticAIAllowed(uuid); err != nil {
		t.Fatal(err)
	}
	if _, err := m.sendPrivateNote(nil, 101, uuid, "future automatic note", nil, json.RawMessage(`{"automatic_email_ai":true}`), false); err != nil {
		t.Fatal(err)
	}
	var notes int
	db.Get(&notes, `SELECT count(*) FROM conversation_messages WHERE private=true`)
	if notes != 3 {
		t.Fatalf("history altered or automatic note leaked: %d", notes)
	}
	if err := m.CheckAutomaticAIAllowed("aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"); err == nil {
		t.Fatal("unknown conversation allowed")
	}
}

func TestServiceMailFoldersIntegration(t *testing.T) {
	m := emailThreadFixture(t, "service_folders")
	m.userStore = folderUsers{}
	db := m.db
	if err := migrations.V2_23_0(db, nil, nil); err != nil {
		t.Fatal(err)
	} // additive/idempotent
	db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'Spam'),(42,'Rekla');`)
	insert := func(inbox int, attrs, subject string, tags []int) int {
		t.Helper()
		var id int
		if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,$1,101,$2,$3::jsonb) RETURNING id`, inbox, subject, attrs).Scan(&id); err != nil {
			t.Fatal(err)
		}
		for _, tag := range tags {
			db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,$2)`, id, tag)
		}
		return id
	}
	first := insert(102, `{"customer_visibility":true}`, "Rekla one", []int{14, 42})
	second := insert(102, `{"customer_visibility":true}`, "Rekla two", []int{14})
	unfiled := insert(102, `{}`, "Other", []int{14})
	insert(102, `{}`, "Rekla spam", []int{14, 12})
	insert(101, `{"access_mode":"personal"}`, "Rekla private", []int{14})
	insert(102, `{"private":true,"visible_users":[102]}`, "Rekla internal", []int{14})
	insert(102, `{"customer_visibility":true}`, "Rekla customer", []int{13})
	if _, err := m.ListSidebarFolders(101, "service_mail", 0); err == nil {
		t.Fatal("ordinary reader could manage shared service folders")
	}
	if _, err := m.ListSidebarFolders(103, "service_mail", 7); err == nil {
		t.Fatal("foreign scope ID allowed")
	}
	makeFolder := func(name, filter string) SidebarFolder {
		t.Helper()
		rows, err := m.MutateSidebarFolder(103, "service_mail", 0, 0, "create", SidebarFolder{Name: name, Filters: json.RawMessage(filter), SortOrder: 99, ShowCount: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return rows[len(rows)-1]
	}
	tags := makeFolder("Tag folder", `[{"model":"conversations","field":"tags","operator":"contains","value":"[42]"}]`)
	subject := makeFolder("Subject folder", `[{"model":"conversations","field":"subject","operator":"contains","value":"Rekla"}]`)
	ids := func(rows []models.ConversationListItem, err error) []int {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		r := []int{}
		for _, row := range rows {
			r = append(r, row.ID)
		}
		slices.Sort(r)
		return r
	}
	if got := ids(m.GetServiceMailConversationsList(103, "", "", "[]", 1, 100)); !slices.Equal(got, []int{first, second, unfiled}) {
		t.Fatalf("all: %v", got)
	}
	for id, want := range map[int][]int{tags.ID: {first}, subject.ID: {second}, -1: {unfiled}} {
		if got := ids(m.GetSidebarFolderConversations(103, "service_mail", 0, id, "", "", 1, 100)); !slices.Equal(got, want) {
			t.Fatalf("folder %d: %v want %v", id, got, want)
		}
	}
	counts, err := m.SidebarFolderCounts(103, "service_mail", 0)
	if err != nil || counts[tags.ID] != 1 || counts[subject.ID] != 1 || counts[-1] != 1 {
		t.Fatalf("counts: %v %v", counts, err)
	}
	if rows, err := m.GetServiceMailConversationsList(103, "asc", "conversations.created_at", "[]", 2, 1); err != nil || len(rows) != 1 || rows[0].ID != second || rows[0].Total != 3 {
		t.Fatalf("pagination: %v %v", rows, err)
	}
	if _, err = m.MutateSidebarFolder(103, "service_mail", 0, 0, "reorder", SidebarFolder{}, []int{subject.ID, tags.ID}); err != nil {
		t.Fatal(err)
	}
	if got := ids(m.GetSidebarFolderConversations(103, "service_mail", 0, subject.ID, "", "", 1, 100)); !slices.Equal(got, []int{first, second}) {
		t.Fatalf("priority: %v", got)
	}
	if got := ids(m.GetSidebarFolderConversations(104, "service_mail", 0, -1, "", "", 1, 100)); !slices.Equal(got, []int{unfiled}) {
		t.Fatalf("admin: %v", got)
	}
	if rows, err := m.ListSidebarFolders(103, "customer", 0); err != nil || len(rows) != 0 {
		t.Fatal("other scope changed")
	}
}
