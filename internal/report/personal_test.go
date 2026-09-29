package report

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

func TestPersonalDashboardVisibility(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_report")
	db.MustExec(`INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,101,101,'{"access_mode":"personal","visible_users":[101]}')`)
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	for _, viewer := range []umodels.User{{ID: 101}, {ID: 102}, {ID: 103, Roles: []string{"Kundensupport"}}, {ID: 104, Roles: []string{"Admin"}}} {
		data, err := m.GetOverViewCounts(viewer)
		if err != nil {
			t.Fatal(err)
		}
		var counts map[string]int
		json.Unmarshal(data, &counts)
		want := 0
		if viewer.ID == 101 || viewer.ID == 104 {
			want = 1
		}
		if counts["open"] != want {
			t.Fatalf("viewer %d: %s", viewer.ID, data)
		}
		for _, query := range []func(int, umodels.User) (json.RawMessage, error){m.GetOverviewSLA, m.GetOverviewChart, m.GetOverviewCSAT, m.GetOverviewMessageVolume, m.GetOverviewTagDistribution} {
			if _, err := query(7, viewer); err != nil {
				t.Fatal(err)
			}
		}
	}
}
