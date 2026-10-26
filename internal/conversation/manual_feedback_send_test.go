package conversation

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/template"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/volatiletech/null/v9"
)

type manualSendUsers struct{ threadingUsers }

func (manualSendUsers) GetAgent(id int, _ string) (umodels.User, error) {
	return umodels.User{ID: id, Type: umodels.UserTypeAgent, FirstName: "Owner", Email: null.StringFrom("owner@example.com")}, nil
}

func (u manualSendUsers) Get(id int, _ string, _ []string) (umodels.User, error) {
	if id == 101 {
		return u.GetAgent(id, "")
	}
	return u.threadingUsers.Get(id, "", nil)
}

type manualSendInbox struct {
	inbox.Inbox
	sent []models.OutboundMessage
	fail bool
}

func (*manualSendInbox) Channel() string          { return inbox.ChannelEmail }
func (*manualSendInbox) FromAddress() string      { return "support@example.com" }
func (*manualSendInbox) FromNameTemplate() string { return "" }
func (i *manualSendInbox) Send(m models.OutboundMessage) error {
	i.sent = append(i.sent, m)
	if i.fail {
		return inbox.ErrInboxNotFound
	}
	return nil
}

type manualSendInboxes struct {
	inboxStore
	transport *manualSendInbox
}

func (i manualSendInboxes) Get(int) (inbox.Inbox, error) { return i.transport, nil }
func (manualSendInboxes) GetDBRecord(any) (imodels.Inbox, error) {
	return imodels.Inbox{ID: 102, Enabled: true, Channel: inbox.ChannelEmail, From: "support@example.com"}, nil
}

// Exercise the actual note API's manager path, QueueReply and SMTP sender with
// an isolated DB and a capturing transport. Feedback retries cannot re-enter Send.
func TestManualResolutionActualRenderedSend(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "manual_real_send"
		if fail {
			name = "manual_real_send_failed"
		}
		t.Run(name, func(t *testing.T) {
			m := emailThreadFixture(t, name)
			incoming := importMail(t, m, threadMail("manual-input@example.test", ""))
			transport := &manualSendInbox{fail: fail}
			m.inboxStore = manualSendInboxes{transport: transport}
			m.userStore = manualSendUsers{}
			var err error
			m.template, err = template.New(m.lo, m.db, nil, nil, nil, m.i18n)
			if err != nil {
				t.Fatal(err)
			}
			m.db.MustExec(`UPDATE templates SET is_default=false WHERE type='email_outgoing'`)
			m.db.MustExec(`INSERT INTO templates(name,type,is_default,body) VALUES('manual test wrapper','email_outgoing',true,'<html><header>Rendered wrapper</header>{{ template "content" . }}</html>')`)
			analysis := AIAnalysis{Kind: "n8n_manual_review", Version: 1, OriginalMessageUUID: incoming.UUID}
			note, err := m.SendAIAnalysisNote(101, incoming.ConversationUUID, "<b>Needs human</b> {{do_not_render}}", analysis)
			if err != nil {
				t.Fatal(err)
			}
			fetched, err := m.GetMessage(note.UUID)
			if err != nil {
				t.Fatal(err)
			}
			var meta struct {
				Analysis AIAnalysis `json:"ai_analysis"`
			}
			if json.Unmarshal(fetched.Meta, &meta) != nil || meta.Analysis != analysis || !fetched.Private || fetched.Content != "<b>Needs human</b> {{do_not_render}}" {
				t.Fatal("manual note metadata or content changed")
			}
			var foreign string
			if err = m.db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`).Scan(&foreign); err != nil {
				t.Fatal(err)
			}
			if _, err = m.SendAIAnalysisNote(101, foreign, "Foreign", analysis); err == nil {
				t.Fatal("foreign original accepted")
			}
			missing := analysis
			missing.OriginalMessageUUID = "dddddddd-dddd-4ddd-dddd-dddddddddddd"
			if _, err = m.SendAIAnalysisNote(101, incoming.ConversationUUID, "Missing", missing); err == nil {
				t.Fatal("missing original accepted")
			}
			outgoing, err := m.QueueReply(nil, 102, 101, 105, incoming.ConversationUUID, "<p>Hello {{ .Author.FirstName }}</p>", []string{"contact@example.com"}, nil, nil, map[string]any{})
			if err != nil {
				t.Fatal(err)
			}
			if outboxCount(t, m) != 0 {
				t.Fatal("queued reply generated feedback")
			}
			m.sendOutgoingMessage(outgoing)
			if len(transport.sent) != 1 {
				t.Fatal("expected exactly one transport attempt")
			}
			want := "<html><header>Rendered wrapper</header><p>Hello Owner</p></html>"
			if transport.sent[0].Content != want {
				t.Fatalf("unexpected rendered content: %q", transport.sent[0].Content)
			}
			probe := &manualFeedbackProbe{fail: true}
			m.deliverManualResolutionFeedback(context.Background(), probe)
			if fail {
				if outboxCount(t, m) != 0 || len(probe.rows) != 0 {
					t.Fatal("SMTP failure generated feedback")
				}
				return
			}
			if len(probe.rows) != 1 || probe.rows[0].HumanAnswer != transport.sent[0].Content {
				t.Fatal("feedback differs from actual SMTP body")
			}
			m.db.MustExec(`UPDATE manual_resolution_feedback SET next_attempt_at=NOW()`)
			probe.fail = false
			m.deliverManualResolutionFeedback(context.Background(), probe)
			m.deliverManualResolutionFeedback(context.Background(), probe)
			if len(transport.sent) != 1 || len(probe.rows) != 2 || probe.rows[0] != probe.rows[1] || outboxCount(t, m) != 1 {
				t.Fatal("retry changed send or feedback identity")
			}
		})
	}
}
