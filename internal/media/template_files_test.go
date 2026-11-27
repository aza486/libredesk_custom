package media

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/messagetemplate"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type templateMemoryStore struct {
	Store
	data       map[string][]byte
	failDelete bool
}

func (s *templateMemoryStore) Name() string { return "fs" }
func (s *templateMemoryStore) Put(name, typ string, r io.ReadSeeker) (string, error) {
	b, e := io.ReadAll(r)
	s.data[name] = b
	return name, e
}
func (s *templateMemoryStore) GetBlob(name string) ([]byte, error) {
	b, ok := s.data[name]
	if !ok {
		return nil, errors.New("missing file")
	}
	return b, nil
}
func (s *templateMemoryStore) Delete(name string) error {
	if s.failDelete {
		return errors.New("store unavailable")
	}
	delete(s.data, name)
	return nil
}
func (s *templateMemoryStore) GetURL(name, d, f string) string {
	return "https://bearer.invalid/" + name
}
func TestTemplateBlobReferenceRetentionAndCleanup(t *testing.T) {
	db := testutil.NewPersonalDB(t, "template_blob_lifecycle")
	lo := logf.New(logf.Opts{})
	store := &templateMemoryStore{data: map[string][]byte{}}
	manager, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t), Store: store, RootURL: func() string { return "https://desk.test" }})
	if err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Writer',ARRAY['messages:write']);INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Writer'`)
	var context string
	db.Get(&context, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`)
	original, err := manager.UploadTemplate("document.pdf", "application/pdf", "", null.StringFrom("message_templates"), null.Int{}, bytes.NewReader([]byte("%PDF-original")), 13, null.StringFrom("attachment"), []byte("{}"), true)
	if err != nil {
		t.Fatal(err)
	}
	templates := messagetemplate.New(db)
	a := messagetemplate.Access{UserID: 101, ConversationUUID: context}
	if err = templates.Stage(a, original.ID, 10000); err != nil {
		t.Fatal(err)
	}
	tpl, err := templates.Create(a, messagetemplate.Input{Name: "PDF", Scope: "personal", AttachmentIDs: []int{original.ID}})
	if err != nil {
		t.Fatal(err)
	}
	adopted, err := templates.Apply(a, tpl.UUID, messagetemplate.ApplyInput{DraftType: "reply"}, 10000)
	if err != nil {
		t.Fatal(err)
	}
	clone := adopted.Attachments[0]
	blob, err := manager.GetBlob(clone.UUID)
	if err != nil || string(blob) != "%PDF-original" {
		t.Fatal("shared storage read", string(blob), err)
	}
	if len(store.data) != 1 {
		t.Fatal("physical duplicate")
	}
	if u := manager.GetSignedURL(clone.UUID); !strings.Contains(u, "/uploads/") || strings.Contains(u, "bearer.invalid") {
		t.Fatal("public clone URL", u)
	}
	if err = templates.Delete(a, tpl.UUID, tpl.Revision); err != nil {
		t.Fatal(err)
	}
	if err = manager.Delete(original.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err = manager.GetBlob(clone.UUID); err != nil {
		t.Fatal("template removal deleted draft", err)
	}
	var msg int
	db.Get(&msg, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type) SELECT id,'outgoing','pending',101,'agent' FROM conversations WHERE uuid=$1 RETURNING id`, context)
	var forged int
	db.Get(&forged, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type) SELECT id,'outgoing','pending',102,'agent' FROM conversations WHERE uuid=$1 RETURNING id`, context)
	attack, _ := db.Beginx()
	if err = manager.LinkMessageMediaTx(attack, forged, nil, []string{clone.UUID}); err == nil {
		t.Fatal("foreign inline reference accepted")
	}
	attack.Rollback()
	db.MustExec(`DELETE FROM conversation_messages WHERE id=$1`, forged)
	linking, _ := db.Beginx()
	if err = manager.LinkMessageMediaTx(linking, msg, adopted.Attachments, nil); err != nil {
		t.Fatal(err)
	}
	if err = linking.Commit(); err != nil {
		t.Fatal(err)
	}
	manager.deleteUnlinked()
	if _, err = manager.GetBlob(clone.UUID); err != nil {
		t.Fatal("pending send lost source", err)
	}
	db.MustExec(`UPDATE conversation_messages SET status='sent' WHERE id=$1`, msg)
	manager.deleteUnlinked()
	if _, err = manager.GetBlob(clone.UUID); err != nil {
		t.Fatal("sent message lost source", err)
	}
	db.MustExec(`DELETE FROM conversation_messages WHERE id=$1`, msg)
	db.MustExec(`UPDATE media SET created_at=now()-interval '10 days'`)
	store.failDelete = true
	manager.deleteUnlinked()
	var queued int
	db.Get(&queued, `SELECT count(*) FROM media_storage_deletions`)
	if queued != 1 {
		t.Fatal("lost durable cleanup intent", queued)
	}
	store.failDelete = false
	manager.deleteUnlinked()
	if len(store.data) != 0 {
		t.Fatal("orphan bytes remain")
	}
	db.Get(&queued, `SELECT count(*) FROM media_storage_deletions`)
	if queued != 0 {
		t.Fatal("cleanup retry failed")
	}
}
