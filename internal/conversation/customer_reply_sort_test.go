package conversation

import (
	"slices"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

func TestCustomerReplySort(t *testing.T) {
	db := testutil.NewPersonalDB(t, "customer_reply_sort")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: personalUsers{}, settingsStore: personalSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	insert := func(inbox int, subject string) int {
		t.Helper()
		var id int
		if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,subject,created_at) VALUES(105,$1,101,$2,'2026-01-01') RETURNING id`, inbox, subject).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	a, b, tie, empty, empty2 := insert(102, "keep"), insert(102, "keep"), insert(102, "keep"), insert(102, "keep"), insert(102, "keep")
	hidden, excluded := insert(101, "keep"), insert(102, "exclude")
	db.MustExec(`UPDATE conversations SET custom_attributes='{"access_mode":"personal","owner_user_id":101,"visible_users":[101]}'::jsonb WHERE id=$1`, hidden)
	message := func(id int, kind, sender string, private bool, at string) {
		t.Helper()
		senderID := 101
		if sender == "contact" {
			senderID = 105
		}
		if _, err := db.Exec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content,created_at) VALUES($1,$2,'sent',$3,$4,$5,'test',$6)`, id, kind, senderID, sender, private, at); err != nil {
			t.Fatal(err)
		}
	}
	message(a, "incoming", "contact", false, "2026-01-01T08:00:00Z")
	message(b, "incoming", "contact", false, "2026-01-01T09:00:00Z")
	message(tie, "incoming", "contact", false, "2026-01-01T09:00:00Z")
	message(hidden, "incoming", "contact", false, "2026-01-01T12:00:00Z")
	message(excluded, "incoming", "contact", false, "2026-01-01T12:00:00Z")
	filters := `[{"model":"conversations","field":"subject","operator":"equals","value":"keep"}]`
	check := func() {
		t.Helper()
		for _, tc := range []struct {
			order string
			want  []int
		}{{"asc", []int{a, b, tie, empty, empty2}}, {"desc", []int{tie, b, a, empty2, empty}}} {
			// Separate pages must concatenate to exactly the same stable global order.
			var got []int
			for page := 1; page <= 3; page++ {
				rows, err := m.GetAllConversationsList(102, tc.order, "customer_reply.last_customer_reply_at", filters, page, 2)
				if err != nil {
					t.Fatal(err)
				}
				for _, row := range rows {
					got = append(got, row.ID)
					if row.Total != 5 {
						t.Fatalf("count includes filtered/inaccessible conversations: %d", row.Total)
					}
					if (row.ID == empty || row.ID == empty2) && row.LastCustomerReplyAt.Valid {
						t.Fatal("missing customer reply must remain NULL")
					}
				}
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("%s: got %v, want %v", tc.order, got, tc.want)
			}
		}
	}
	check()
	for _, tc := range []struct {
		kind, sender string
		private      bool
	}{{"outgoing", "agent", false}, {"outgoing", "agent", true}, {"activity", "agent", true}, {"incoming", "agent", false}, {"incoming", "contact", true}} {
		message(a, tc.kind, tc.sender, tc.private, "2026-01-01T14:00:00Z")
		message(empty, tc.kind, tc.sender, tc.private, "2026-01-01T14:00:00Z")
		db.MustExec(`UPDATE conversations SET last_message_at='2026-01-01T14:00:00Z' WHERE id=$1`, a)
		check()
	}
	// An older incoming message inserted later must not replace the latest reply.
	message(b, "incoming", "contact", false, "2026-01-01T07:00:00Z")
	check()

	// Two matching custom folders remain exclusive, with sorting and pagination inside the first.
	db.MustExec(`UPDATE conversations SET assigned_user_id=104 WHERE id=ANY($1)`, pq.Array([]int{a, b, tie, empty, empty2, excluded}))
	db.MustExec(`INSERT INTO conversation_assignees(conversation_id,user_id) SELECT id,104 FROM conversations WHERE assigned_user_id=104 ON CONFLICT DO NOTHING`)
	var folders []SidebarFolder
	for position, name := range []string{"First", "Second"} {
		var err error
		folders, err = m.MutateSidebarFolder(104, "my_inbox", 104, 0, "create", SidebarFolder{Name: name, Filters: []byte(filters), ShowCount: true, SortOrder: position}, nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		order string
		want  []int
	}{{"asc", []int{a, b, tie, empty, empty2}}, {"desc", []int{tie, b, a, empty2, empty}}} {
		var got []int
		for page := 1; page <= 3; page++ {
			rows, err := m.GetSidebarFolderConversations(104, "my_inbox", 104, folders[0].ID, tc.order, "customer_reply.last_customer_reply_at", page, 2)
			if err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				got = append(got, row.ID)
			}
		}
		if !slices.Equal(got, tc.want) {
			t.Fatalf("folder %s: %v, want %v", tc.order, got, tc.want)
		}
		rows, err := m.GetSidebarFolderConversations(104, "my_inbox", 104, folders[1].ID, tc.order, "customer_reply.last_customer_reply_at", 1, 10)
		if err != nil || len(rows) != 0 {
			t.Fatalf("later folder lost exclusivity: %+v, %v", rows, err)
		}
	}
}

func TestCustomerReplySortKeepsScopeAndFolderFilters(t *testing.T) {
	m := &Manager{settingsStore: personalSettings{}}
	folder := SidebarFolder{ID: 1, Filters: []byte(`[{"model":"conversations","field":"subject","operator":"equals","value":"folder"}]`)}
	for _, order := range []string{"asc", "desc"} {
		query, args, err := m.makeScopedConversationsListQuery(102, 102, nil, []string{models.AllConversations}, `SELECT conversations.id FROM conversations WHERE 1=1 %s`, order, "customer_reply.last_customer_reply_at", 2, 10, `[{"model":"conversations","field":"priority_id","operator":"equals","value":"101"}]`, false, false, 1, []SidebarFolder{folder}, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, fragment := range []string{"conversations.inbox_id", "conversations.subject", "conversations.priority_id", "ORDER BY customer_reply.last_customer_reply_at " + strings.ToUpper(order) + " NULLS LAST, conversations.id " + strings.ToUpper(order) + " NULLS LAST", "LIMIT"} {
			if !strings.Contains(query, fragment) {
				t.Fatalf("missing %q in %s", fragment, query)
			}
		}
		if args[len(args)-2] != 10 || args[len(args)-1] != 10 {
			t.Fatalf("pagination args: %v", args)
		}
	}
	if _, ok := conversationListAllowedFields["customer_reply"]; ok {
		t.Fatal("sort mutated the shared filter allowlist")
	}
}
