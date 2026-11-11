package conversation

import (
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

type folderUsers struct{ personalUsers }

func (u folderUsers) GetAgent(id int, _ string) (umodels.User, error) {
	return u.GetAgentCachedOrLoad(id)
}
func (folderUsers) GetAgentCachedOrLoad(id int) (umodels.User, error) {
	user, _ := personalUsers{}.GetAgentCachedOrLoad(id)
	user.Permissions = []string{"conversations:read", "conversations:read_team_all"}
	return user, nil
}

func TestSidebarFoldersIntegration(t *testing.T) {
	db := testutil.NewPersonalDB(t, "sidebar_folders")
	// Migration must be repeatable.
	if err := migrations.V2_18_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: folderUsers{}, settingsStore: personalSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO teams(id,name,conversation_assignment_type) VALUES(101,'One','Manual'),(102,'Two','Manual');
 INSERT INTO team_members(team_id,user_id) VALUES(101,101),(101,102),(102,103);
 INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(101,101),(101,102) ON CONFLICT DO NOTHING;
 INSERT INTO tags(id,name) VALUES(42,'Rekla');`)
	var closed int
	if err := db.Get(&closed, `SELECT id FROM conversation_statuses WHERE name='Closed'`); err != nil {
		t.Fatal(err)
	}
	insert := func(inbox, status int, attrs string, team int, tag bool) int {
		t.Helper()
		var id int
		if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,subject,assigned_team_id,custom_attributes) VALUES(105,$1,$2,'Rekla',NULLIF($3,0),$4::jsonb) RETURNING id`, inbox, status, team, attrs).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if tag {
			db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,42)`, id)
		}
		return id
	}
	customerOpen := insert(102, 101, `{"customer_visibility":true}`, 0, true)
	customerClosed := insert(102, closed, `{"customer_visibility":true}`, 0, true)
	other := insert(102, 101, `{"customer_visibility":true}`, 0, false)
	internal := insert(102, 101, `{"private":true,"customer_visibility":true,"visible_users":[103]}`, 0, true)
	personal := insert(101, 101, `{"access_mode":"personal","owner_user_id":101}`, 0, true)
	teamOne := insert(102, 101, `{}`, 101, true)
	teamTwo := insert(102, 101, `{}`, 102, true)
	service := insert(102, 101, `{}`, 101, true)
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,14)`, service)
	create := func(user int, scope string, scopeID int, name, filters string, position int) SidebarFolder {
		t.Helper()
		all, err := m.MutateSidebarFolder(user, scope, scopeID, 0, "create", SidebarFolder{Name: name, Filters: json.RawMessage(filters), ShowCount: true, SortOrder: position}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range all {
			if f.Name == name {
				return f
			}
		}
		t.Fatal("missing created folder")
		return SidebarFolder{}
	}
	assertList := func(user int, scope string, scopeID, folderID int, want ...int) {
		t.Helper()
		rows, err := m.GetSidebarFolderConversations(user, scope, scopeID, folderID, "", "", 1, 100)
		if err != nil {
			t.Fatal(err)
		}
		got := []int{}
		for _, row := range rows {
			got = append(got, row.ID)
			if row.Total != len(want) {
				t.Fatalf("total=%d want %d", row.Total, len(want))
			}
		}
		slices.Sort(got)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Fatalf("%s folder %d: got %v want %v", scope, folderID, got, want)
		}
		counts, err := m.SidebarFolderCounts(user, scope, scopeID)
		if err != nil || counts[folderID] != len(want) {
			t.Fatalf("count=%v want %d err=%v", counts, len(want), err)
		}
	}
	all := create(103, "customer", 0, "Alle Reklas", folderTagFilter, 0)
	assertList(103, "customer", 0, all.ID, customerOpen, customerClosed)
	rows, err := m.GetCustomerConversationsList(103, false, "", "", "[]", 1, 100)
	if err != nil || len(rows) != 2 {
		t.Fatalf("standard customer list: %v %v", rows, err)
	}
	// Internal tickets remain governed by the existing standard list; a customer
	// folder cannot claim one, even if it has a customer_visibility attribute.
	for _, row := range rows {
		if row.ID != other && row.ID != internal {
			t.Fatal("assigned customer still in standard list")
		}
	}
	openFilter := fmt.Sprintf(`{"logic":"AND","rules":[{"model":"conversations","field":"tags","operator":"contains","value":"[42]"},{"model":"conversations","field":"status_id","operator":"equals","value":"101"}]}`)
	open := create(104, "customer", 0, "Offene Reklas", openFilter, 0)
	assertList(103, "customer", 0, open.ID, customerOpen)
	assertList(103, "customer", 0, all.ID, customerClosed)
	// Both authorized employees read identical shared definitions.
	left, e1 := m.ListSidebarFolders(103, "customer", 0)
	right, e2 := m.ListSidebarFolders(104, "customer", 0)
	if e1 != nil || e2 != nil || len(left) != 2 || left[0].ID != right[0].ID {
		t.Fatal("shared definitions diverged")
	}
	// Tag removal and status changes move logical membership without side effects.
	db.MustExec(`DELETE FROM conversation_tags WHERE conversation_id=$1 AND tag_id=42`, customerOpen)
	assertList(103, "customer", 0, open.ID)
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,42)`, customerOpen)
	db.MustExec(`UPDATE conversations SET status_id=$2 WHERE id=$1`, customerOpen, closed)
	assertList(103, "customer", 0, all.ID, customerOpen, customerClosed)
	db.MustExec(`UPDATE conversations SET status_id=101 WHERE id=$1`, customerOpen)
	if _, err = m.MutateSidebarFolder(103, "customer", 0, 0, "reorder", SidebarFolder{}, []int{all.ID, open.ID}); err != nil {
		t.Fatal(err)
	}
	assertList(103, "customer", 0, all.ID, customerOpen, customerClosed)
	assertList(103, "customer", 0, open.ID)
	// Pagination is applied after exclusive assignment.
	paged, err := m.GetSidebarFolderConversations(103, "customer", 0, all.ID, "", "", 2, 1)
	if err != nil || len(paged) != 1 || paged[0].Total != 2 {
		t.Fatalf("pagination: %#v %v", paged, err)
	}
	// Global views continue to see claimed tickets.
	global, err := m.GetViewConversationsList(103, 103, nil, []string{models.AllConversations}, "", "", folderTagFilter, 1, 100)
	if err != nil || len(global) < 2 {
		t.Fatalf("global view lost tickets: %v", err)
	}
	// OR must remain inside each scope/access boundary.
	orFilter := `{"logic":"OR","rules":[{"model":"conversations","field":"subject","operator":"contains","value":"Rekla"},{"model":"conversations","field":"inbox_id","operator":"equals","value":"101"}]}`
	tf := create(101, "team", 101, "Team", orFilter, 0)
	pf := create(101, "personal", 101, "Private", orFilter, 0)
	assertList(102, "team", 101, tf.ID, teamOne)
	assertList(102, "personal", 101, pf.ID, personal)
	_ = teamTwo
	for _, req := range []struct {
		user  int
		scope string
		id    int
	}{{102, "customer", 0}, {103, "team", 101}, {103, "personal", 101}, {104, "personal", 101}, {101, "personal", 102}, {101, "team", 102}} {
		if _, err = m.ListSidebarFolders(req.user, req.scope, req.id); err == nil {
			t.Fatalf("unauthorized definitions: %+v", req)
		}
		if _, err = m.MutateSidebarFolder(req.user, req.scope, req.id, tf.ID, "delete", SidebarFolder{}, nil); err == nil {
			t.Fatalf("unauthorized mutation: %+v", req)
		}
		if _, err = m.GetSidebarFolderConversations(req.user, req.scope, req.id, tf.ID, "", "", 1, 20); err == nil {
			t.Fatalf("unauthorized list: %+v", req)
		}
	}
	if _, err = m.GetSidebarFolderConversations(101, "team", 101, pf.ID, "", "", 1, 20); err == nil {
		t.Fatal("cross-scope ID accepted")
	}
	if _, err = m.MutateSidebarFolder(101, "team", 101, pf.ID, "delete", SidebarFolder{}, nil); err == nil {
		t.Fatal("cross-scope delete accepted")
	}
	if _, err = m.MutateSidebarFolder(103, "customer", 0, 0, "reorder", SidebarFolder{}, []int{all.ID, all.ID}); err == nil {
		t.Fatal("duplicate permutation accepted")
	}
	// Concurrent reorder operations leave one complete deterministic order.
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ids := []int{all.ID, open.ID}
			if i%2 == 0 {
				slices.Reverse(ids)
			}
			_, err := m.MutateSidebarFolder(103, "customer", 0, 0, "reorder", SidebarFolder{}, ids)
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	after, err := m.ListSidebarFolders(103, "customer", 0)
	if err != nil || len(after) != 2 || after[0].SortOrder != 0 || after[1].SortOrder != 1 {
		t.Fatal("inconsistent concurrent ordering")
	}
	// Shared badge settings persist without affecting membership.
	updated := after[0]
	updated.ShowCount = false
	updated.Name = "Umbenannt"
	if _, err = m.MutateSidebarFolder(104, "customer", 0, updated.ID, "update", updated, nil); err != nil {
		t.Fatal(err)
	}
	shared, err := m.ListSidebarFolders(103, "customer", 0)
	if err != nil || shared[0].Name != "Umbenannt" || shared[0].ShowCount {
		t.Fatal("shared edit not persisted")
	}
	counts, err := m.SidebarFolderCounts(103, "customer", 0)
	if err != nil || counts[all.ID]+counts[open.ID] != 2 {
		t.Fatal("disabled badge changed membership")
	}
	// Deleting definitions changes no conversation, message, tag or visibility.
	var before, afterData string
	snapshot := `SELECT jsonb_agg(to_jsonb(c) ORDER BY id)::text FROM conversations c`
	if err = db.Get(&before, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, folder := range after {
		if _, err = m.MutateSidebarFolder(104, "customer", 0, folder.ID, "delete", SidebarFolder{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err = db.Get(&afterData, snapshot); err != nil || before != afterData {
		t.Fatal("folder deletion mutated conversations")
	}
	rows, err = m.GetCustomerConversationsList(103, false, "", "", "[]", 1, 100)
	if err != nil || len(rows) != 4 {
		t.Fatalf("standard fallback after delete: %d %v", len(rows), err)
	}
}

func TestSidebarRejectsInvalidFilterTypes(t *testing.T) {
	db := testutil.NewPersonalDB(t, "sidebar_invalid_types")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: folderUsers{}, settingsStore: personalSettings{}}
	for _, filter := range []string{
		`[{"model":"conversations","field":"status_id","operator":"equals","value":"not an ID"}]`,
		`[{"model":"conversations","field":"priority_id","operator":"contains","value":"1"}]`,
		`[{"model":"conversations","field":"created_at","operator":"less than","value":"bad date"}]`,
		`[{"model":"conversations","field":"subject","operator":"contains","value":"valid"},{}]`,
	} {
		if _, err := m.MutateSidebarFolder(103, "customer", 0, 0, "create", SidebarFolder{Name: "Invalid", Filters: json.RawMessage(filter), ShowCount: true}, nil); err == nil {
			t.Fatalf("accepted bad filter %s", filter)
		}
	}
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM sidebar_folders`); err != nil || n != 0 {
		t.Fatal("invalid definitions persisted")
	}
}

func TestSidebarNullSemantics(t *testing.T) {
	db := testutil.NewPersonalDB(t, "sidebar_null")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: folderUsers{}, settingsStore: personalSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,102,101,NULL,'{"customer_visibility":true}'),(105,102,101,'Rekla','{"customer_visibility":true}')`)
	folders, err := m.MutateSidebarFolder(103, "customer", 0, 0, "create", SidebarFolder{Name: "Mit Betreff", Filters: json.RawMessage(`[{"model":"conversations","field":"subject","operator":"not equals","value":"Other"}]`), ShowCount: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := m.SidebarFolderCounts(103, "customer", 0)
	if err != nil || counts[folders[0].ID] != 1 {
		t.Fatalf("NULL matched folder: %v %v", counts, err)
	}
	rows, err := m.GetCustomerConversationsList(103, false, "", "", "[]", 1, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("NULL disappeared from standard list: %d %v", len(rows), err)
	}
}
