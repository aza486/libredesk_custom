package conversation

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/template"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
)

func TestFormAddressValidation(t *testing.T) {
	for _, value := range []string{"", "broken", "a@example.com,b@example.com", "Name <a@example.com>", "a@example.com\r\nBcc:x@example.com", "BESTELLUNG@pixelfotoexpress.de"} {
		if _, err := validFormAddress(value, []string{"bestellung@pixelfotoexpress.de"}); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	if got, err := validFormAddress("kunde+auftrag@example.com", nil); err != nil || got != "kunde+auftrag@example.com" {
		t.Fatal(got, err)
	}
}

func TestFormRecipientResolutionAndTransport(t *testing.T) {
	m := emailThreadFixture(t, "form_recipient")
	m.db.MustExec(`CREATE TABLE mail_events(conversation_uuid text,message_uuid text,is_form boolean,direction text,sender_email text);
 CREATE TABLE cases(conversation_uuid text,customer_email text)`)
	m.formRecipients = &FormRecipients{DB: m.db, TechnicalSenders: []string{"bestellung@pixelfotoexpress.de"}}
	transport := &manualSendInbox{fail: true} // Stop after captured SMTP, before unrelated SLA/feedback processing.
	m.inboxStore = manualSendInboxes{transport: transport}
	m.userStore = manualSendUsers{}
	var err error
	m.template, err = template.New(m.lo, m.db, nil, nil, nil, m.i18n)
	if err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`UPDATE templates SET is_default=false WHERE type='email_outgoing'; INSERT INTO templates(name,type,is_default,body) VALUES('form test','email_outgoing',true,'{{ template "content" . }}')`)
	loadPending := func(uuid string) models.Message {
		t.Helper()
		var pending []models.Message
		if err := m.q.GetOutgoingPendingMessages.Select(&pending, pq.Array([]int{})); err != nil {
			t.Fatal(err)
		}
		for _, message := range pending {
			if message.UUID == uuid {
				return message
			}
		}
		t.Fatalf("pending message %s not found", uuid)
		return models.Message{}
	}
	makeForm := func(name string) models.Message {
		in := importMail(t, m, threadMail(name+"@example.test", ""))
		// Two forms deliberately share the unchanged technical contact.
		m.db.MustExec(`UPDATE users SET email='bestellung@pixelfotoexpress.de' WHERE id=105`)
		m.db.MustExec(`UPDATE conversation_messages SET meta=jsonb_set(meta,'{from}','["bestellung@pixelfotoexpress.de"]'::jsonb) WHERE uuid=$1`, in.UUID)
		return in
	}
	first := makeForm("contact-form")
	second := makeForm("reclamation-form")
	insert := func(in models.Message, email string) {
		m.db.MustExec(`INSERT INTO mail_events VALUES($1,$2,true,'inbound',$3)`, in.ConversationUUID, in.UUID, email)
	}
	checkManual := func(in models.Message) {
		t.Helper()
		to := []string{"corrected@example.com", "alternative@example.com"}
		cc := []string{"cc@example.com"}
		bcc := []string{"bcc@example.com"}
		reply, e := m.QueueReply(nil, 102, 101, 105, in.ConversationUUID, "manual", to, cc, bcc, nil)
		if e != nil {
			t.Fatalf("manual recipients blocked: %v", e)
		}
		count := len(transport.sent)
		reply = loadPending(reply.UUID)
		m.sendOutgoingMessage(reply)
		if len(transport.sent) != count+1 {
			t.Fatal("manual send blocked")
		}
		sent := transport.sent[count]
		if !slices.Equal(sent.To, to) || !slices.Equal(sent.CC, cc) || !slices.Equal(sent.BCC, bcc) {
			t.Fatalf("recipients changed: %+v", sent)
		}
	}
	checkBlocked := func(in models.Message) {
		t.Helper()
		got, e := m.ResolveReplyRecipient(in.ConversationUUID)
		if e == nil || !got.Managed || !got.Blocked || got.Email != "" {
			t.Fatalf("not blocked: %+v %v", got, e)
		}
		if _, e = m.QueueReply(nil, 102, 101, 105, in.ConversationUUID, "test", []string{"bestellung@pixelfotoexpress.de"}, nil, nil, nil); e == nil {
			t.Fatal("queue accepted unsafe form")
		}
		checkManual(in)
	}
	// Delay, cross-conversation records, wrong message UUID and outbound rows never allow fallback.
	checkBlocked(first)
	insert(second, "second@example.com")
	checkBlocked(first)
	m.db.MustExec(`INSERT INTO mail_events VALUES($1,$2,true,'outbound','wrong@example.com'),($1,'foreign-message',true,'inbound','wrong@example.com')`, first.ConversationUUID, first.UUID)
	checkBlocked(first)
	insert(first, "first@example.com")
	for _, tc := range []struct {
		in    models.Message
		email string
	}{{first, "first@example.com"}, {second, "second@example.com"}} {
		got, e := m.ResolveReplyRecipient(tc.in.ConversationUUID)
		if e != nil || got.Email != tc.email || got.Blocked {
			t.Fatal(got, e)
		}
		for _, ai := range []bool{false, true} {
			reply, e := m.QueueReply(nil, 102, 101, 105, tc.in.ConversationUUID, "test", []string{"chosen@example.com", "another@example.com"}, []string{"leak@example.com"}, []string{"hidden@example.com"}, map[string]any{"automatic_email_ai": ai})
			if e != nil {
				t.Fatal(e)
			}
			var meta struct{ To, CC, BCC []string }
			if e = json.Unmarshal(reply.Meta, &meta); e != nil || len(meta.To) != 2 || meta.To[0] != "chosen@example.com" || len(meta.CC) != 1 || len(meta.BCC) != 1 {
				t.Fatal(string(reply.Meta), e)
			}
			// Queued employee addresses remain authoritative even if pg-n8n changes.
			m.db.MustExec(`UPDATE mail_events SET sender_email='changed-after-queue@example.com' WHERE message_uuid=$1 AND direction='inbound'`, tc.in.UUID)
			reply = loadPending(reply.UUID)
			m.sendOutgoingMessage(reply)
			if len(transport.sent) == 0 {
				t.Fatal("transport never called")
			}
			sent := transport.sent[len(transport.sent)-1]
			if len(sent.To) != 2 || sent.To[0] != "chosen@example.com" || len(sent.CC) != 1 || len(sent.BCC) != 1 {
				t.Fatalf("unsafe send: %+v", sent)
			}
			saved, e := m.GetMessage(reply.UUID)
			if e != nil || json.Unmarshal(saved.Meta, &meta) != nil || meta.To[0] != "chosen@example.com" {
				t.Fatal("recipient not persisted", e)
			}
			m.db.MustExec(`UPDATE mail_events SET sender_email=$1 WHERE message_uuid=$2 AND direction='inbound'`, tc.email, tc.in.UUID)
		}
	}
	// Customer follow-up threads via existing references without changing shared contact.
	follow := threadMail("follow@example.test", first.SourceID.String)
	follow.Contact.Email = null.StringFrom("first@example.com")
	follow.ConversationUUIDFromReplyTo = first.ConversationUUID
	incoming := importMail(t, m, follow)
	if incoming.ConversationUUID != first.ConversationUUID {
		t.Fatal("follow-up lost thread")
	}
	got, e := m.ResolveReplyRecipient(first.ConversationUUID)
	if e != nil || got.Email != "first@example.com" {
		t.Fatal(got, e)
	}
	for _, bad := range []string{"", "not email", "bestellung@pixelfotoexpress.de", "Name <first@example.com>", "a@example.com\nBcc:b@example.com"} {
		m.db.MustExec(`UPDATE mail_events SET sender_email=$1 WHERE message_uuid=$2 AND direction='inbound'`, bad, first.UUID)
		checkBlocked(first)
	}
	m.db.MustExec(`UPDATE mail_events SET sender_email='first@example.com' WHERE message_uuid=$1 AND direction='inbound'`, first.UUID)
	m.db.MustExec(`INSERT INTO cases VALUES($1,'contradiction@example.com')`, first.ConversationUUID)
	checkBlocked(first)
	m.db.MustExec(`DELETE FROM cases`)
	m.db.MustExec(`INSERT INTO mail_events VALUES($1,$2,true,'inbound','another@example.com')`, first.ConversationUUID, first.UUID)
	checkBlocked(first)
	m.db.MustExec(`DELETE FROM mail_events WHERE sender_email='another@example.com'`)
	// Unreachable pg-n8n only affects the suggestion; manual sends still work.
	offline, err := sqlx.Open("postgres", "postgres://test:test@127.0.0.1:1/test?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatal(err)
	}
	defer offline.Close()
	m.formRecipients.DB = offline
	checkBlocked(first)
	// An old technical-only job must be flagged without readdressing or SMTP.
	legacy, err := m.QueueReply(nil, 102, 101, 105, first.ConversationUUID, "old", []string{"manual@example.com"}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`UPDATE conversation_messages SET meta='{"to":["bestellung@pixelfotoexpress.de"]}' WHERE uuid=$1`, legacy.UUID)
	legacy = loadPending(legacy.UUID)
	before := len(transport.sent)
	m.sendOutgoingMessage(legacy)
	if len(transport.sent) != before {
		t.Fatal("legacy technical default was sent")
	}
	saved, err := m.GetMessage(legacy.UUID)
	var review struct {
		Required bool     `json:"recipient_review_required"`
		To       []string `json:"to"`
	}
	if err != nil || json.Unmarshal(saved.Meta, &review) != nil || !review.Required || saved.Status != models.MessageStatusFailed || !slices.Equal(review.To, legacy.To) {
		t.Fatal("missing recipient review", saved, err)
	}
	// Explicit employee confirmation is bound to the complete recipient decision.
	meta := map[string]any{RecipientConfirmationKey: ConfirmReplyRecipients(101, legacy.To, nil, nil)}
	explicit, err := m.QueueReply(nil, 102, 101, 105, first.ConversationUUID, "intentional", legacy.To, nil, nil, meta)
	if err != nil {
		t.Fatal(err)
	}
	explicit = loadPending(explicit.UUID)
	m.sendOutgoingMessage(explicit)
	if len(transport.sent) != before+1 || !slices.Equal(transport.sent[before].To, legacy.To) {
		t.Fatal("confirmed decision changed")
	}
	if _, err = m.QueueReply(nil, 102, 101, 105, first.ConversationUUID, "forged", legacy.To, []string{"changed@example.com"}, nil, meta); err == nil {
		t.Fatal("confirmation reused for other recipients")
	}
	for _, bad := range []string{"broken", "a@example.com,b@example.com", "a@example.com\r\nBcc:b@example.com"} {
		if _, err = m.QueueReply(nil, 102, 101, 105, first.ConversationUUID, "bad", []string{bad}, nil, nil, nil); err == nil {
			t.Fatal("invalid address accepted")
		}
	}
	var shared string
	m.db.Get(&shared, `SELECT email FROM users WHERE id=105`)
	if shared != "bestellung@pixelfotoexpress.de" {
		t.Fatal("shared contact changed")
	}
	m.db.MustExec(`UPDATE users SET email='contact@example.com' WHERE id=105`)
	normal := importMail(t, m, threadMail("normal@example.test", ""))
	resolved, err := m.ResolveReplyRecipient(normal.ConversationUUID)
	if err != nil || resolved.Managed {
		t.Fatal(resolved, err)
	}
	normalReply, err := m.QueueReply(nil, 102, 101, 105, normal.ConversationUUID, "normal", []string{"normal@example.com"}, []string{"cc@example.com"}, []string{"bcc@example.com"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var normalMeta struct{ To, CC, BCC []string }
	if json.Unmarshal(normalReply.Meta, &normalMeta) != nil || len(normalMeta.To) != 1 || normalMeta.To[0] != "normal@example.com" || len(normalMeta.CC) != 1 || normalMeta.CC[0] != "cc@example.com" || len(normalMeta.BCC) != 1 || normalMeta.BCC[0] != "bcc@example.com" {
		t.Fatal(string(normalReply.Meta))
	}
	before = len(transport.sent)
	// Notes never consult the external database or enter SMTP.
	note, err := m.SendPrivateNote(nil, 101, first.ConversationUUID, "private note", nil)
	if err != nil {
		t.Fatal(err)
	}
	m.sendOutgoingMessage(note)
	if len(transport.sent) != before {
		t.Fatal("note reached transport")
	}
}
