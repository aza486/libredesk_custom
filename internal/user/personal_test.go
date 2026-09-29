package user

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

func TestPersonalContactListAndExport(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_contact")
	db.MustExec(`INSERT INTO conversations(id,contact_id,inbox_id,status_id,custom_attributes) VALUES(101,105,101,101,'{"access_mode":"personal","visible_users":[101]}')`)
	lo := logf.New(logf.Opts{})
	m, err := New(testutil.NewI18n(t), Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []models.User{{ID: 101}, {ID: 102}, {ID: 103, Roles: []string{"Kundensupport"}}, {ID: 104, Roles: []string{"Admin"}}} {
		want := viewer.ID == 101 || viewer.ID == 104
		allowed, err := m.CanAccessContact(105, viewer)
		if err != nil || allowed != want {
			t.Fatalf("contact %d: %v %v", viewer.ID, allowed, err)
		}
		contacts, err := m.GetContacts(1, 20, "desc", "users.created_at", "[]", "UTC", viewer)
		if err != nil {
			t.Fatal(err)
		}
		if (len(contacts) > 0) != want {
			t.Fatalf("list %d: %v", viewer.ID, contacts)
		}
		data, err := m.ExportContactData(105, viewer)
		if err != nil {
			t.Fatal(err)
		}
		var exported struct {
			Conversations []json.RawMessage `json:"conversations"`
		}
		if err := json.Unmarshal(data, &exported); err != nil {
			t.Fatal(err)
		}
		if (len(exported.Conversations) > 0) != want {
			t.Fatalf("export %d: %s", viewer.ID, data)
		}
	}
	// A shared public contact remains globally available; personal content still does not.
	db.MustExec(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101)`)
	allowed, err := m.CanAccessContact(105, models.User{ID: 102})
	if err != nil || !allowed {
		t.Fatal("shared public contact hidden", err)
	}
	data, err := m.ExportContactData(105, models.User{ID: 102})
	if err != nil {
		t.Fatal(err)
	}
	var exported struct {
		Conversations []json.RawMessage `json:"conversations"`
	}
	json.Unmarshal(data, &exported)
	if len(exported.Conversations) != 1 {
		t.Fatal("personal content in shared contact export")
	}
}
