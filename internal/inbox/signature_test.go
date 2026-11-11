package inbox

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/template"
	tmodels "github.com/abhinavxd/libredesk/internal/template/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestInboxDefaultSignatureLifecycle(t *testing.T) {
	db := testutil.NewPersonalDB(t, "inbox_signature")
	// The additive migration can run twice without changing existing inboxes.
	var before, after string
	const snapshot = `SELECT jsonb_agg(to_jsonb(i) ORDER BY id)::text FROM inboxes i`
	if err := db.Get(&before, snapshot); err != nil {
		t.Fatal(err)
	}
	if err := migrations.V2_21_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&after, snapshot); err != nil || before != after {
		t.Fatal("migration changed existing inboxes", err)
	}
	lo := logf.New(logf.Opts{})
	lang := testutil.NewI18n(t)
	m, err := New(&lo, db, lang, strings.Repeat("x", 32))
	if err != nil {
		t.Fatal(err)
	}
	templates, err := template.New(&lo, db, nil, nil, nil, lang)
	if err != nil {
		t.Fatal(err)
	}
	x, err := templates.Create(tmodels.Template{Name: "Support signature", Type: "email_outgoing", Body: `{{ template "content" . }}<p>Support</p>`})
	if err != nil {
		t.Fatal(err)
	}
	y, err := templates.Create(tmodels.Template{Name: "Personal signature", Type: "email_outgoing", Body: `{{ template "content" . }}<p>Personal</p>`})
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for _, mode := range []string{"public", "personal"} {
		in := imodels.Inbox{Name: "Signature " + mode, Channel: "email", AccessMode: mode, From: "desk@example.com", Enabled: true,
			DefaultSignatureID: null.IntFrom(x.ID), Config: json.RawMessage(`{"imap":[{"password":"secret"}],"smtp":[{"password":"secret"}],"reply_to":"reply@example.com","enable_plus_addressing":true}`)}
		if mode == "personal" {
			in.OwnerUserID = null.IntFrom(101)
		}
		created, err := m.Create(in)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, created.ID)
		read, err := m.GetDBRecord(created.ID)
		if err != nil || read.DefaultSignatureID.Int != x.ID {
			t.Fatal("signature not persisted", err)
		}
		var config map[string]any
		if err := json.Unmarshal(read.Config, &config); err != nil {
			t.Fatal(err)
		}
		read.DefaultSignatureID = null.IntFrom(y.ID)
		updated, err := m.Update(read.ID, read)
		if err != nil || updated.DefaultSignatureID.Int != y.ID {
			t.Fatal("signature update", err)
		}
		var updatedConfig map[string]any
		if err := json.Unmarshal(updated.Config, &updatedConfig); err != nil {
			t.Fatal(err)
		}
		for key, value := range config {
			if !reflect.DeepEqual(updatedConfig[key], value) {
				t.Fatalf("config field %s changed: %#v -> %#v", key, value, updatedConfig[key])
			}
		}
		if updated.From != read.From || updated.AccessMode != mode {
			t.Fatal("unrelated inbox settings changed")
		}
		read.DefaultSignatureID = null.Int{}
		if _, err := m.Update(read.ID, read); err != nil {
			t.Fatal(err)
		}
		read, err = m.GetDBRecord(read.ID)
		if err != nil || read.DefaultSignatureID.Valid {
			t.Fatal("signature not removed", err)
		}
		read.DefaultSignatureID = null.IntFrom(x.ID)
		if _, err := m.Update(read.ID, read); err != nil {
			t.Fatal(err)
		}
	}
	// Notification templates and nonexistent IDs cannot be assigned through the manager.
	notification, err := templates.Create(tmodels.Template{Name: "Private notification", Type: "email_notification", Body: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{-1, 9999999, notification.ID} {
		in, err := m.GetDBRecord(ids[0])
		if err != nil {
			t.Fatal(err)
		}
		in.DefaultSignatureID = null.IntFrom(id)
		_, err = m.Update(in.ID, in)
		var env envelope.Error
		if !errors.As(err, &env) || env.ErrorType != envelope.InputError {
			t.Fatalf("invalid signature %d accepted: %v", id, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO inboxes(name,channel,default_signature_id) VALUES('Chat','livechat',$1)`, x.ID); err == nil {
		t.Fatal("chat signature accepted")
	}
	if _, err := db.Exec(`UPDATE templates SET type='email_notification' WHERE id=$1`, x.ID); err == nil {
		t.Fatal("linked signature converted to notification")
	}
	x.Name, x.Body = "Renamed", `{{ template "content" . }}<p>Updated</p>`
	if _, err := templates.Update(x.ID, x); err != nil {
		t.Fatal(err)
	}
	rendered, err := templates.RenderEmailWithTemplateID(nil, "<p>Reply</p>", x.ID)
	if err != nil || rendered != "<p>Reply</p><p>Updated</p>" {
		t.Fatal("central render/update", rendered, err)
	}
	for _, id := range ids {
		in, err := m.GetDBRecord(id)
		if err != nil || in.DefaultSignatureID.Int != x.ID {
			t.Fatal("rename broke reference", err)
		}
	}
	if err := templates.Delete(x.ID); err != nil {
		t.Fatal(err)
	}
	rendered, err = templates.RenderEmailWithTemplateID(nil, "<p>Saved reply</p>", x.ID)
	if err != nil || !strings.Contains(rendered, "<p>Saved reply</p>") {
		t.Fatal("deleted draft signature did not fall back", rendered, err)
	}
	if _, err := templates.RenderEmailWithTemplateID(nil, "<p>Reply</p>", notification.ID); err == nil {
		t.Fatal("notification rendered as outgoing signature")
	}

	for _, id := range ids {
		in, err := m.GetDBRecord(id)
		if err != nil || in.DefaultSignatureID.Valid {
			t.Fatal("delete did not clear reference", err)
		}
	}
}

func TestInboxSignatureConcurrentDeleteAndTypeChange(t *testing.T) {
	db := testutil.NewPersonalDB(t, "inbox_signature_race")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, testutil.NewI18n(t), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"delete", "change_type"} {
		t.Run(operation, func(t *testing.T) {
			var signatureID int
			if err := db.Get(&signatureID, `INSERT INTO templates(name,type,body) VALUES($1,'email_outgoing','body') RETURNING id`, operation); err != nil {
				t.Fatal(err)
			}
			tx, err := db.Beginx()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			if err := m.validateDefaultSignature(tx, imodels.Inbox{Channel: "email", DefaultSignatureID: null.IntFrom(signatureID)}); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				query := `DELETE FROM templates WHERE id=$1`
				if operation == "change_type" {
					query = `UPDATE templates SET type='email_notification' WHERE id=$1`
				}
				_, err := db.Exec(query, signatureID)
				done <- err
			}()
			select {
			case err := <-done:
				t.Fatalf("template write did not wait for validation lock: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			if _, err := tx.Exec(`UPDATE inboxes SET default_signature_id=$1 WHERE id=102`, signatureID); err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-done:
				if operation == "delete" && err != nil {
					t.Fatal(err)
				}
				if operation == "change_type" && err == nil {
					t.Fatal("concurrent conversion exposed a notification template")
				}
			case <-time.After(30 * time.Second):
				t.Fatal("concurrent write did not complete")
			}
			var stored null.Int
			if err := db.Get(&stored, `SELECT default_signature_id FROM inboxes WHERE id=102`); err != nil {
				t.Fatal(err)
			}
			if operation == "delete" && stored.Valid {
				t.Fatal("dangling reference")
			}
		})
	}
}
