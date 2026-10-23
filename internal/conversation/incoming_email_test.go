package conversation

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/inbox"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type threadingUsers struct {
	personalUsers
	entered chan struct{}
	release chan struct{}
	once    *sync.Once
}

func (u threadingUsers) ResolveContact(user *umodels.User, _ umodels.ContactPolicy) error {
	if u.entered != nil {
		u.once.Do(func() {
			close(u.entered)
			<-u.release
		})
	}
	user.ID = 105
	return nil
}

func (threadingUsers) Get(id int, _ string, _ []string) (umodels.User, error) {
	return umodels.User{ID: id, Type: umodels.UserTypeContact, Email: null.StringFrom("contact@example.com")}, nil
}

type threadingMedia struct{ mediaStore }

func (threadingMedia) LinkMessageMediaTx(*sqlx.Tx, int, []mmodels.Media, []string) error { return nil }
func (threadingMedia) GetByModel(int, string) ([]mmodels.Media, error)                   { return nil, nil }

type threadingWebhooks struct{ webhookStore }

func (threadingWebhooks) TriggerEvent(wmodels.WebhookEvent, any) {}

type threadingInboxes struct{ inboxStore }

func (threadingInboxes) Get(int) (inbox.Inbox, error) { return nil, errors.New("no widget in test") }

func emailThreadFixture(t *testing.T, name string) *Manager {
	t.Helper()
	db := testutil.NewPersonalDB(t, name)
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: threadingUsers{},
		settingsStore: personalSettings{}, mediaStore: threadingMedia{}, webhookStore: threadingWebhooks{},
		inboxStore: threadingInboxes{}, wsHub: ws.NewHub(&lo, nil)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	engine, err := automation.New(automation.Opts{DB: db, Lo: &lo, I18n: m.i18n})
	if err != nil {
		t.Fatal(err)
	}
	m.automation = engine
	return m
}

func threadMail(id, parent string) models.IncomingMessage {
	return models.IncomingMessage{Channel: "email", InboxID: 102, Subject: "Order", Content: "Customer text",
		SourceID: null.StringFrom(id), InReplyTo: parent,
		Contact: models.IncomingContact{Email: null.StringFrom("contact@example.com")}}
}

func importMail(t *testing.T, m *Manager, in models.IncomingMessage) models.Message {
	t.Helper()
	msg, err := m.ProcessIncomingMessage(in)
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func TestIncomingEmailThreads(t *testing.T) {
	m := emailThreadFixture(t, "incoming_email_threads")
	a := importMail(t, m, threadMail("a@example.test", ""))
	b := importMail(t, m, threadMail("b@example.test", "a@example.test"))
	c := importMail(t, m, threadMail("c@example.test", "b@example.test"))
	if a.ConversationID != b.ConversationID || b.ConversationID != c.ConversationID {
		t.Fatal("customer replies split before first employee reply")
	}
	var stored string
	if err := m.db.Get(&stored, `SELECT source_id FROM conversation_messages WHERE uuid=$1`, a.UUID); err != nil || stored != "a@example.test" {
		t.Fatalf("incoming source ID was not retained: %q %v", stored, err)
	}
	independent := importMail(t, m, threadMail("independent@example.test", ""))
	if independent.ConversationID == a.ConversationID {
		t.Fatal("same subject/contact merged independent thread")
	}
	if _, err := m.db.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,source_id) VALUES($1,'outgoing','sent',101,'agent','Agent','agent@example.test')`, a.ConversationID); err != nil {
		t.Fatal(err)
	}
	d := importMail(t, m, threadMail("d@example.test", "agent@example.test"))
	if d.ConversationID != a.ConversationID {
		t.Fatal("employee reply threading changed")
	}
	refs := threadMail("refs@example.test", "")
	refs.References = []string{"unknown@example.test", "a@example.test", "b@example.test"}
	if msg := importMail(t, m, refs); msg.ConversationID != a.ConversationID {
		t.Fatal("references-only reply did not match")
	}
	conflict := threadMail("conflict@example.test", "a@example.test")
	conflict.References = []string{"independent@example.test"}
	if msg := importMail(t, m, conflict); msg.ConversationID == a.ConversationID || msg.ConversationID == independent.ConversationID {
		t.Fatal("conflicting references merged tickets")
	}
	plus := threadMail("plus@example.test", "")
	plus.ConversationUUIDFromReplyTo = a.ConversationUUID
	if msg := importMail(t, m, plus); msg.ConversationID != a.ConversationID {
		t.Fatal("plus-address reply did not match")
	}
	if msg := importMail(t, m, threadMail("b@example.test", "a@example.test")); msg.UUID != "" {
		t.Fatal("re-import created duplicate")
	}
	// Raw IDs cannot route between either public or personal inboxes.
	if _, err := m.db.Exec(`INSERT INTO inboxes(id,name,channel,access_mode,"from") VALUES(106,'Other public','email','public','other@example.test')`); err != nil {
		t.Fatal(err)
	}
	for _, inboxID := range []int{101, 106} {
		cross := threadMail("a@example.test", "b@example.test")
		cross.InboxID = inboxID
		msg := importMail(t, m, cross)
		if msg.UUID == "" || msg.ConversationID == a.ConversationID {
			t.Fatalf("cross-inbox duplicate suppression/thread match: inbox %d", inboxID)
		}
	}
}

func TestConcurrentIncomingEmailImport(t *testing.T) {
	m := emailThreadFixture(t, "incoming_email_concurrency")
	entered, release := make(chan struct{}), make(chan struct{})
	m.userStore = threadingUsers{entered: entered, release: release, once: &sync.Once{}}
	parentDone := make(chan error, 1)
	go func() {
		_, err := m.ProcessIncomingMessage(threadMail("parent@example.test", ""))
		parentDone <- err
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("parent import did not start")
	}
	// Another importer must neither create a ticket nor insert a duplicate
	// while the parent is not yet persisted. IMAP retries these on its next scan.
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := range 8 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			in := threadMail(fmt.Sprintf("child%d@example.test", i), "parent@example.test")
			if i == 0 {
				in = threadMail("parent@example.test", "")
			}
			_, err := m.ProcessIncomingMessage(in)
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	close(release)
	for err := range errs {
		if !errors.Is(err, ErrIncomingInboxBusy) {
			t.Errorf("concurrent import must be deferred, got %v", err)
		}
	}
	if err := <-parentDone; err != nil {
		t.Fatal(err)
	}
	for i := range 8 {
		importMail(t, m, threadMail(fmt.Sprintf("child%d@example.test", i), "parent@example.test"))
	}
	importMail(t, m, threadMail("parent@example.test", ""))
	var conversations, messages int
	if err := m.db.Get(&conversations, `SELECT count(*) FROM conversations`); err != nil {
		t.Fatal(err)
	}
	if err := m.db.Get(&messages, `SELECT count(*) FROM conversation_messages WHERE type='incoming'`); err != nil {
		t.Fatal(err)
	}
	if conversations != 1 || messages != 9 {
		t.Fatalf("concurrent import: %d conversations, %d incoming messages", conversations, messages)
	}
}
