package conversation

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

func TestUnassignedListUsesViewingUserForTicketVisibility(t *testing.T) {
	db := testutil.NewPersonalDB(t, "unassigned_viewer_visibility")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: personalUsers{}, settingsStore: personalSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	insert := func(inbox int, attrs string) int {
		t.Helper()
		var id int
		if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes)
 VALUES(105,$1,101,$2::jsonb) RETURNING id`, inbox, attrs).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	visibleCustomer := insert(102, `{"customer_visibility":true,"visible_users":[102]}`)
	otherCustomer := insert(102, `{"customer_visibility":true,"visible_users":[101]}`)
	visibleInternal := insert(102, `{"private":true,"creator_id":101,"visible_users":[102]}`)
	otherInternal := insert(102, `{"private":true,"creator_id":101,"visible_users":[101]}`)
	insert(101, `{"access_mode":"personal","owner_user_id":101,"visible_users":[101]}`)
	spam := insert(102, `{"customer_visibility":true,"visible_users":[102]}`)
	service := insert(102, `{"customer_visibility":true,"visible_users":[102]}`)
	assigned := insert(102, `{"customer_visibility":true,"visible_users":[102]}`)
	db.MustExec(`INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,12),($2,14)`, spam, service)
	db.MustExec(`UPDATE conversations SET assigned_user_id=101 WHERE id=$1`, assigned)
	db.MustExec(`INSERT INTO conversation_assignees(conversation_id,user_id) VALUES($1,101) ON CONFLICT DO NOTHING`, assigned)

	db.MustExec(`UPDATE conversations SET created_at=TIMESTAMPTZ '2020-01-01' + id * INTERVAL '1 second'`)

	for _, tc := range []struct {
		name string
		user int
		want []int
	}{
		{"explicit visible employee", 102, []int{visibleCustomer, visibleInternal}},
		{"other employee and internal creator", 101, []int{otherCustomer, visibleInternal, otherInternal}},
		{"admin retains access and folder exclusions", 104, []int{visibleCustomer, otherCustomer, visibleInternal, otherInternal}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := m.GetUnassignedConversationsList(tc.user, "ASC", "conversations.created_at", "[]", 1, 100)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]int, 0, len(rows))
			for _, row := range rows {
				got = append(got, row.ID)
				if row.Total != len(tc.want) {
					t.Fatalf("count includes an inaccessible or excluded ticket: %d, want %d", row.Total, len(tc.want))
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("unassigned list for viewer %d: %v, want %v", tc.user, got, tc.want)
			}
		})
	}
	t.Run("visibility is applied before pagination", func(t *testing.T) {
		for page, id := range []int{visibleCustomer, visibleInternal} {
			rows, err := m.GetUnassignedConversationsList(102, "ASC", "conversations.created_at", "[]", page+1, 1)
			if err != nil || len(rows) != 1 || rows[0].ID != id || rows[0].Total != 2 {
				t.Fatalf("page %d: %+v, %v", page+1, rows, err)
			}
		}
	})
	t.Run("revoked visibility is excluded", func(t *testing.T) {
		db.MustExec(`UPDATE conversations SET custom_attributes='{"customer_visibility":true,"visible_users":[]}' WHERE id=$1`, visibleCustomer)
		rows, err := m.GetUnassignedConversationsList(102, "ASC", "conversations.created_at", "[]", 1, 100)
		if err != nil || len(rows) != 1 || rows[0].ID != visibleInternal || rows[0].Total != 1 {
			t.Fatalf("revoked ticket remained visible: %+v, %v", rows, err)
		}
	})
}
