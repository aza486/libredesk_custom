package user

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestNormalizeWorkPhone(t *testing.T) {
	for _, value := range []string{"", "   ", "0351 40424-23", "0351 40424-99", "+49 351 40424-23", "+49 (351) 40424-23", "030 12345678", "+49 351 1234567 Durchwahl 23", "  0351 40424-23  "} {
		got, err := NormalizeWorkPhone(value)
		if err != nil || got.String != strings.TrimSpace(value) || got.Valid != (strings.TrimSpace(value) != "") {
			t.Fatalf("%q: %#v %v", value, got, err)
		}
	}
	for _, value := range []string{"abc", "12", "<b>0351</b>", "{{123}}", "0351\n", "\t0351", "0351\x00", "0351\u202e", "0351 & 42", strings.Repeat("1", 65), string([]byte{0xff})} {
		if _, err := NormalizeWorkPhone(value); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
}

func TestWorkPhoneLifecycleAndMigration(t *testing.T) {
	db := testutil.NewPersonalDB(t, "work_phone")
	var before, after string
	const snapshot = `SELECT jsonb_agg(to_jsonb(u) - 'work_phone_number' ORDER BY id)::text FROM users u`
	if err := db.Get(&before, snapshot); err != nil {
		t.Fatal(err)
	}
	// Recreate a pre-migration users table with populated profiles.
	db.MustExec(`ALTER TABLE users DROP COLUMN work_phone_number`)
	if err := migrations.V2_22_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := migrations.V2_22_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&after, snapshot); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("repeated migration modified users")
	}
	var count int
	db.Get(&count, `SELECT count(*) FROM users WHERE work_phone_number IS NOT NULL`)
	if count != 0 {
		t.Fatal("migration populated phones")
	}
	lo := logf.New(logf.Opts{})
	m, err := New(testutil.NewI18n(t), Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	original, err := m.GetAgent(101, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{" 0351 40424-23 ", "+49 (351) 1234567 Durchwahl 23", ""} {
		if err := m.UpdateWorkPhone(101, value); err != nil {
			t.Fatal(err)
		}
		if _, exists := m.GetAgentFromCache(101); exists {
			t.Fatal("stale cache after update")
		}
		got, err := m.GetAgentCachedOrLoad(101)
		if err != nil || got.WorkPhoneNumber.String != strings.TrimSpace(value) {
			t.Fatalf("reload: %#v %v", got, err)
		}
		if got.FirstName != original.FirstName || got.Email != original.Email || got.Enabled != original.Enabled {
			t.Fatal("unrelated fields changed")
		}
		raw, _ := json.Marshal(got)
		if strings.Contains(string(raw), "work_phone_number") {
			t.Fatal("business phone leaked through generic user serialization")
		}
	}
	if err := m.UpdateWorkPhone(101, "<b>123</b>"); err == nil {
		t.Fatal("invalid phone accepted")
	}
	if err := m.UpdateWorkPhone(105, "0351 40424-23"); err == nil {
		t.Fatal("contact updated as agent")
	}
	var phone *string
	if err := db.Get(&phone, `SELECT work_phone_number FROM users WHERE id=101`); err != nil || phone != nil {
		t.Fatal("clear must persist NULL", err)
	}
}
