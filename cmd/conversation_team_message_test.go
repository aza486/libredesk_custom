package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	inboxemail "github.com/abhinavxd/libredesk/internal/inbox/channel/email"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
)

func TestInternalTeamStoredHTTP(t *testing.T) {
	f := newNotificationHTTP(t, "internal_team_stored")
	f.db.MustExec(`UPDATE roles SET permissions=array_append(permissions,'messages:write') WHERE name IN ('Security creator','Security observer')`)
	path := "/api/v1/settings/notifications/email"
	// The new admin selection works without either old recipient key.
	f.app.internalConversation.ContactEmail = ""
	f.db.MustExec(`DELETE FROM settings WHERE key='notification.email.team_contact_email'`)
	var identity string
	if err := f.db.Get(&identity, `SELECT uuid FROM inboxes WHERE id=3087`); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{"notification.email.inbox_uuid": identity, "notification.email.enabled": false})
	f.request(t, 104, "PUT", path, string(body), 200)
	var before int
	f.db.Get(&before, `SELECT count(*) FROM users`)
	var result struct {
		Data cmodels.Conversation `json:"data"`
	}
	raw := f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 0, 201), 200)
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	c := result.Data
	if c.UUID == "" || c.ContactID != 0 || c.Contact.Email.Valid || c.Contact.ID != 0 || c.AssignedTeamID.Int != 201 || c.AssignedUserID.Valid {
		t.Fatalf("not a contactless team ticket: %+v", c)
	}
	var after, count int
	f.db.Get(&after, `SELECT count(*) FROM users`)
	if after != before {
		t.Fatal("team creation manufactured a contact")
	}
	f.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND type<>'activity'`, c.ID)
	if count != 1 {
		t.Fatalf("expected exactly one initial message, got %d", count)
	}
	messages, _, err := f.app.conversation.GetConversationMessages(c.UUID, 1, 50, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	var initial cmodels.Message
	for _, msg := range messages {
		if msg.Type == cmodels.MessageInternal {
			initial = msg
		}
	}
	if initial.Status != "stored" || initial.Private || initial.Content != "<p>Internal message</p>" || initial.SenderID != 101 || initial.SourceID.Valid || strings.Contains(string(initial.Meta), `"to"`) {
		t.Fatalf("bad stored message: %+v", initial)
	}
	item, err := f.app.conversation.GetConversationListItem(c.UUID)
	if err != nil || !strings.Contains(item.LastMessage.String, "Internal message") {
		t.Fatalf("list preview lost: %+v %v", item, err)
	}
	for actor, want := range map[int]int{101: 200, 102: 200, 103: 403, 104: 200} {
		f.request(t, actor, "GET", "/api/v1/conversations/"+c.UUID, "", want)
		f.request(t, actor, "GET", "/api/v1/conversations/"+c.UUID+"/messages", "", want)
	}
	// The same normal message API stores follow-ups. Forged recipients cannot
	// turn this ticket into an external message, nor impersonate a contact.
	f.request(t, 101, "POST", "/api/v1/conversations/"+c.UUID+"/messages", `{"sender_type":"agent","message":"<p>Follow-up</p>","private":false}`, 200)
	for _, payload := range []string{
		`{"sender_type":"agent","message":"x","to":["external@example.test"]}`,
		`{"sender_type":"contact","message":"x"}`,
		`{"sender_type":"agent","message":"x","cc":["external@example.test"]}`,
	} {
		f.request(t, 101, "POST", "/api/v1/conversations/"+c.UUID+"/messages", payload, 400)
	}
	f.request(t, 103, "POST", "/api/v1/conversations/"+c.UUID+"/messages", `{"sender_type":"agent","message":"forbidden"}`, 403)
	if _, err := f.app.conversation.QueueReply(nil, 3087, 101, 0, c.UUID, "forged", []string{"external@example.test"}, nil, nil, map[string]any{}); err == nil {
		t.Fatal("direct queue bypassed contactless ticket")
	}
	f.db.Get(&count, `SELECT count(*) FROM conversation_messages WHERE conversation_id=$1 AND (type='outgoing' OR status IN ('pending','sent','failed')) AND type<>'activity'`, c.ID)
	if count != 0 {
		t.Fatal("stored team content was marked as transport")
	}
	var contactsNull bool
	f.db.Get(&contactsNull, `SELECT contact_id IS NULL FROM conversations WHERE id=$1`, c.ID)
	if !contactsNull {
		t.Fatal("contactless ticket acquired a recipient")
	}
}

func TestInternalCreationActualTransport(t *testing.T) {
	f := newNotificationHTTP(t, "internal_team_transport")
	port, delivered := notificationSMTP(t)
	transport, err := inboxemail.New(nil, nil, inboxemail.Opts{ID: 3087, Name: "Test SMTP", Lo: f.app.lo, Config: imodels.Config{From: "system@example.test", SMTP: []imodels.SMTPConfig{{Host: "127.0.0.1", Port: port, AuthProtocol: "none", TLSType: "none", MaxConns: 1}}}})
	if err != nil {
		t.Fatal(err)
	}
	f.app.inbox.Register(transport)
	t.Cleanup(func() { transport.Close() })
	f.request(t, 104, "PUT", "/api/v1/settings/notifications/email", notificationChoice(t, f, 3087, ""), 200)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); f.app.conversation.Run(ctx, 0, 1, 10*time.Millisecond) }()
	t.Cleanup(func() { cancel(); <-done; f.app.conversation.Close() })
	for _, team := range []int{0, 201} {
		f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 102, team), 200)
		select {
		case data := <-delivered:
			if !strings.Contains(data, "other@example.com") {
				t.Fatal("employee recipient lost", data)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("employee email not delivered")
		}
	}
	f.request(t, 101, "POST", "/api/v1/conversations", internalCreateBody(0, 0, 201), 200)
	select {
	case <-delivered:
		t.Fatal("team-only ticket caused SMTP delivery or duplicate employee mail")
	case <-time.After(200 * time.Millisecond):
	}
	var sent int
	f.db.Get(&sent, `SELECT count(*) FROM conversation_messages WHERE type='outgoing' AND status='sent'`)
	if sent != 2 {
		t.Fatalf("employee emails not sent once each: %d", sent)
	}
}
