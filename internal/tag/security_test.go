package tag

import (
	"fmt"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestSystemTagDefinitionProtection(t *testing.T) {
	db := testutil.NewDB(t, "tag_definition_security")
	if err := EnsureSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	regular, err := m.Create("Ordinary")
	if err != nil {
		t.Fatal(err)
	}
	before := tagSnapshot(t, db)
	for _, required := range requiredSystemTags {
		t.Run(fmt.Sprint(required.ID), func(t *testing.T) {
			for _, name := range []string{required.Name, " " + required.Name + " ", required.Name + "\ufe0f"} {
				if _, err := m.Create(name); err == nil {
					t.Fatalf("reserved creation allowed: %q", name)
				}
				if _, err := m.Update(regular.ID, name); err == nil {
					t.Fatalf("reserved rename allowed: %q", name)
				}
			}
			if _, err := m.Update(required.ID, "Forged"); err == nil {
				t.Fatal("system tag renamed")
			}
			if err := m.Delete(required.ID); err == nil {
				t.Fatal("system tag deleted")
			}
		})
	}
	if after := tagSnapshot(t, db); after != before {
		t.Fatal("rejected operations changed tag data or sequence")
	}
	visible, err := m.GetAll()
	if err != nil || len(visible) != 2 {
		t.Fatalf("ordinary tag and editable Spam must remain visible: %+v %v", visible, err)
	}
	if _, err := m.Update(regular.ID, "Renamed"); err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(regular.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSystemTagDefinitionLegacyIdentities(t *testing.T) {
	db := testutil.NewDB(t, "tag_definition_legacy_security")
	db.MustExec(`INSERT INTO tags(id,name) VALUES(8,'Other name'),(23,'🏢Intern'),(24,'🗑Spam'),(25,'🏢intern')`)
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	before := tagSnapshot(t, db)
	for _, id := range []int{8, 23, 24, 25} {
		if err := m.Delete(id); err == nil {
			t.Fatalf("legacy reserved identity %d deleted", id)
		}
		if _, err := m.Update(id, "Ordinary"); err == nil {
			t.Fatalf("legacy reserved identity %d renamed", id)
		}
	}
	if after := tagSnapshot(t, db); after != before {
		t.Fatal("legacy definitions changed")
	}
}
