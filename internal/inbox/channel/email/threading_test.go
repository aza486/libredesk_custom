package email

import (
	"errors"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/inbox"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime/v2"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestThreadMessageIDs(t *testing.T) {
	for _, tc := range []struct {
		name, value, want string
	}{
		{"single", "<a@example.test>", "a@example.test"},
		{"comments", "(outer (nested)) <a@example.test> (parent)", "a@example.test"},
		{"adjacent", "<a@example.test><b@example.test>", "a@example.test,b@example.test"},
		{"folded", "<a@example.test>\r\n\t<b@example.test>", "a@example.test,b@example.test"},
		{"gateway", "<a@@example.test> <b@example.test>", "a@@example.test,b@example.test"},
		{"bare gateway", "a@@example.test", "a@@example.test"},
		{"missing", "", ""},
		{"empty", "<>", ""},
		{"nested brackets", "<<a@example.test>>", ""},
		{"unclosed", "<a@example.test", ""},
		{"damaged suffix", "<a@example.test> garbage", ""},
		{"comment only", "(example <a@example.test>)", ""},
		{"prose", "reply to a@example.test", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.Join(parseThreadMessageIDs(tc.value), ","); got != tc.want {
				t.Fatalf("IDs = %q, want %q", got, tc.want)
			}
		})
	}
}

type threadingStore struct {
	inbox.MessageStore
	got []models.IncomingMessage
	err error
}

func (s *threadingStore) EnqueueIncoming(models.IncomingMessage) error {
	panic("IMAP must wait for persistence instead of queuing related messages concurrently")
}

func (s *threadingStore) ProcessIncomingMessage(m models.IncomingMessage) (models.Message, error) {
	s.got = append(s.got, m)
	return models.Message{}, s.err
}

type threadingLiteral struct{ *strings.Reader }

func (r threadingLiteral) Size() int64 { return r.Reader.Size() }

func TestFullMIMEThreadHeaders(t *testing.T) {
	store := &threadingStore{}
	lo := logf.New(logf.Opts{})
	e := &Email{messageStore: store, lo: &lo}
	raw := "From: customer@example.test\r\nTo: support@example.test\r\n" +
		"Message-ID: <child@example.test>\r\n" +
		"In-Reply-To: (parent) <parent@example.test>\r\n" +
		"References: <root@example.test><parent@example.test>\r\n" +
		"References: (another (nested comment)) <older@example.test>\r\n" +
		"Subject: Same subject\r\nContent-Type: text/plain\r\n\r\nBody"
	env, err := enmime.ReadEnvelope(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if got := extractMessageIDFromHeaders(env); got != "child@example.test" {
		t.Fatalf("source ID: %q", got)
	}
	err = e.processFullMessage(imapclient.FetchItemDataBodySection{Literal: threadingLiteral{strings.NewReader(raw)}}, models.IncomingMessage{SourceID: null.StringFrom("child@example.test")})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.got) != 1 || store.got[0].InReplyTo != "parent@example.test" || strings.Join(store.got[0].References, ",") != "root@example.test,parent@example.test,older@example.test" {
		t.Fatalf("unexpected parsed thread headers: %+v", store.got)
	}
	store.err = errors.New("temporary persistence failure")
	err = e.processFullMessage(imapclient.FetchItemDataBodySection{Literal: threadingLiteral{strings.NewReader(raw)}}, models.IncomingMessage{})
	var persistenceErr incomingEmailPersistenceError
	if !errors.Is(err, store.err) || !errors.As(err, &persistenceErr) {
		t.Fatalf("persistence failure must stop the IMAP batch and remain retryable: %v", err)
	}
}
