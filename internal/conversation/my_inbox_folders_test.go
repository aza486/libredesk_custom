package conversation

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/zerodha/logf"
)

type myInboxUsers struct{ folderUsers }

func (u myInboxUsers) GetAgent(id int, _ string) (umodels.User, error) {
	return u.GetAgentCachedOrLoad(id)
}
func (myInboxUsers) GetAgentCachedOrLoad(id int) (umodels.User, error) {
	user, err := folderUsers{}.GetAgentCachedOrLoad(id)
	user.Permissions = append(user.Permissions, "conversations:read_assigned")
	return user, err
}

func TestMyInboxFoldersIntegration(t *testing.T) {
	db := testutil.NewPersonalDB(t, "my_inbox_folders")
	lo := logf.New(logf.Opts{})
	newManager := func(users userStore) *Manager {
		manager := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: users, settingsStore: personalSettings{}}
		if err := dbutil.ScanSQLFile("queries.sql", &manager.q, db, efs); err != nil {
			t.Fatal(err)
		}
		return manager
	}
	m := newManager(myInboxUsers{})
	db.MustExec(`INSERT INTO teams(id,name,conversation_assignment_type) VALUES(101,'Team','Manual');
 INSERT INTO team_members(team_id,user_id) VALUES(101,101);
 INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(101,101) ON CONFLICT DO NOTHING;
 INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id) VALUES(103,'Other personal','email','personal',102);
 INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(103,102) ON CONFLICT DO NOTHING;
 UPDATE inboxes SET "from"='box@example.com' WHERE id=103;`)
	var closed int
	if err := db.Get(&closed, `SELECT id FROM conversation_statuses WHERE name='Closed'`); err != nil {
		t.Fatal(err)
	}
	insert := func(inbox, status int, attrs string, assigned bool, tag int) int {
		t.Helper()
		var id int
		if err := db.QueryRow(`INSERT INTO conversations(contact_id,inbox_id,status_id,priority_id,subject,assigned_team_id,custom_attributes) VALUES(105,$1,$2,101,'Rekla',101,$3::jsonb) RETURNING id`, inbox, status, attrs).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if assigned {
			db.MustExec(`INSERT INTO conversation_assignees(conversation_id,user_id) VALUES($1,101)`, id)
		}
		if tag > 0 {
			db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) VALUES($1,$2)`, id, tag)
		}
		return id
	}
	open := insert(102, 101, `{}`, true, 0)
	done := insert(102, closed, `{}`, true, 0)
	internal := insert(102, 101, `{"private":true,"visible_users":[101]}`, true, 13)
	_ = insert(102, 101, `{"private":true,"visible_users":[102]}`, true, 13) // assignment is not a visibility bypass
	customer := insert(102, 101, `{"customer_visibility":true,"visible_users":[101]}`, true, 0)
	_ = insert(102, 101, `{"customer_visibility":true,"visible_users":[102]}`, true, 0)
	service := insert(102, 101, `{"visible_users":[101]}`, true, 14)
	ownPrivate := insert(101, 101, `{"access_mode":"personal","visible_users":[101]}`, true, 0)
	sharedPrivate := insert(103, 101, `{"access_mode":"personal","visible_users":[101]}`, true, 0)
	visibleOnlyInternal := insert(102, 101, `{"private":true,"visible_users":[101]}`, false, 13) // visible, but not assigned
	_ = insert(102, 101, `{"visible_users":[101]}`, false, 14)
	ids := func(rows []models.ConversationListItem, err error) []int {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		result := []int{}
		for _, row := range rows {
			result = append(result, row.ID)
		}
		slices.Sort(result)
		return result
	}
	baseline := ids(m.GetAssignedConversationsList(101, 101, false, "", "", "[]", 1, 100))
	want := []int{open, done, internal, customer, service, sharedPrivate}
	slices.Sort(want)
	if !slices.Equal(baseline, want) {
		t.Fatalf("unexpected assigned baseline: %v want %v", baseline, want)
	}
	otherAreas := func() map[string][]int {
		return map[string][]int{
			"visible":          ids(m.GetVisibleConversationsList(101, "", "", "[]", 1, 100)),
			"visible-internal": ids(m.GetVisibleInternalConversationsList(101, "", "", "[]", 1, 100)),
			"customer":         ids(m.GetCustomerConversationsList(103, false, "", "", "[]", 1, 100)),
			"team":             ids(m.GetTeamConversationsList(101, 101, false, "", "", "[]", 1, 100)),
			"personal":         ids(m.GetPersonalConversationsList(101, 101, false, "", "", "[]", 1, 100)),
		}
	}
	before := otherAreas()
	if !slices.Contains(before["visible"], service) || slices.Contains(before["visible-internal"], service) || !slices.Contains(before["visible-internal"], visibleOnlyInternal) {
		t.Fatal("visible/visible-internal baseline did not preserve service/internal rules")
	}
	orFilter := `{"logic":"OR","rules":[{"model":"conversations","field":"subject","operator":"contains","value":"Rekla"},{"model":"conversations","field":"inbox_id","operator":"equals","value":"101"}]}`
	definitions, err := m.MutateSidebarFolder(101, "my_inbox", 101, 0, "create", SidebarFolder{Name: "Alle Reklas", Filters: json.RawMessage(orFilter), ShowCount: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	all := definitions[0]
	if got := ids(m.GetSidebarFolderConversations(101, "my_inbox", 101, all.ID, "", "", 1, 100)); !slices.Equal(got, baseline) {
		t.Fatalf("folder changed assigned access: %v want %v", got, baseline)
	}
	if got := ids(m.GetAssignedConversationsList(101, 101, false, "", "", "[]", 1, 100)); len(got) != 0 {
		t.Fatalf("standard inbox still contains claimed tickets: %v", got)
	}
	if got := ids(m.GetAssignedConversationsList(101, 101, true, "", "", "[]", 1, 100)); len(got) != 0 {
		t.Fatalf("high priority still contains claimed tickets: %v", got)
	}
	for area, got := range otherAreas() {
		if !slices.Equal(got, before[area]) {
			t.Fatalf("my_inbox changed %s: %v want %v", area, got, before[area])
		}
	}
	if !slices.Contains(before["personal"], ownPrivate) || slices.Contains(baseline, ownPrivate) {
		t.Fatal("owner's personal inbox was folded into assigned")
	}
	openFilter := `[{"model":"conversation_statuses","field":"name","operator":"equals","value":"Open"}]`
	definitions, err = m.MutateSidebarFolder(101, "my_inbox", 101, 0, "create", SidebarFolder{Name: "Offene", Filters: json.RawMessage(openFilter), ShowCount: true, SortOrder: 0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	first := definitions[0]
	if got := ids(m.GetSidebarFolderConversations(101, "my_inbox", 101, all.ID, "", "", 1, 100)); !slices.Equal(got, []int{done}) {
		t.Fatalf("priority did not exclude higher matches: %v", got)
	}
	counts, err := m.SidebarFolderCounts(101, "my_inbox", 101)
	if err != nil || counts[first.ID] != len(baseline)-1 || counts[all.ID] != 1 {
		t.Fatalf("wrong exclusive counts: %v %v", counts, err)
	}
	if _, err = m.MutateSidebarFolder(101, "my_inbox", 101, 0, "reorder", SidebarFolder{}, []int{all.ID, first.ID}); err != nil {
		t.Fatal(err)
	}
	counts, err = m.SidebarFolderCounts(101, "my_inbox", 101)
	if err != nil || counts[all.ID] != len(baseline) || counts[first.ID] != 0 {
		t.Fatalf("reorder did not reassign: %v %v", counts, err)
	}
	// Definitions survive a new manager/session, and remain specific to this
	// employee's assignment-based inbox rather than being globally shared.
	another := newManager(myInboxUsers{})
	reloaded, err := another.ListSidebarFolders(101, "my_inbox", 101)
	if err != nil || len(reloaded) != 2 || reloaded[0].ID != all.ID {
		t.Fatal("reload lost order")
	}
	for _, user := range []int{102, 104} {
		if _, err = m.ListSidebarFolders(user, "my_inbox", 101); err == nil {
			t.Fatal("foreign employee definitions leaked")
		}
		if _, err = m.MutateSidebarFolder(user, "my_inbox", 101, all.ID, "delete", SidebarFolder{}, nil); err == nil {
			t.Fatal("foreign employee mutated inbox")
		}
		if _, err = m.SidebarFolderCounts(user, "my_inbox", 101); err == nil {
			t.Fatal("foreign employee counts leaked")
		}
		if _, err = m.GetSidebarFolderConversations(user, "my_inbox", 101, all.ID, "", "", 1, 100); err == nil {
			t.Fatal("foreign employee list leaked")
		}
	}
	for _, scopeID := range []int{0, 102} {
		if _, err = m.ListSidebarFolders(101, "my_inbox", scopeID); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	empty, err := m.ListSidebarFolders(102, "my_inbox", 102)
	if err != nil || len(empty) != 0 {
		t.Fatal("employee inboxes shared different assignments")
	}
	noPermission := newManager(folderUsers{})
	if _, err = noPermission.ListSidebarFolders(101, "my_inbox", 101); err == nil {
		t.Fatal("missing read_assigned permission ignored")
	}
	// Repeatable migration upgrades the old constraints and preserves definitions.
	if err = migrations.V2_19_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	reloaded, err = m.ListSidebarFolders(101, "my_inbox", 101)
	if err != nil || len(reloaded) != 2 {
		t.Fatal("repeat migration changed definitions")
	}
	for _, f := range reloaded {
		if _, err = m.MutateSidebarFolder(101, "my_inbox", 101, f.ID, "delete", SidebarFolder{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := ids(m.GetAssignedConversationsList(101, 101, false, "", "", "[]", 1, 100)); !slices.Equal(got, baseline) {
		t.Fatalf("deleting folders did not restore standard inbox: %v", got)
	}
}

func TestMyInboxUsesAssignedQuery(t *testing.T) {
	m := &Manager{settingsStore: personalSettings{}}
	for _, support := range []bool{false, true} {
		base := `SELECT conversations.id FROM conversations WHERE $1::int>0 AND NOT $2::boolean %s`
		assigned, args, err := m.makeConversationsListQuery(101, 101, nil, []string{models.AssignedConversations}, base, "", "", 1, 10, "[]", false, support)
		if err != nil {
			t.Fatal(err)
		}
		folder, folderArgs, err := m.makeConversationsListQuery(101, 101, nil, []string{"folder-my_inbox"}, base, "", "", 1, 10, "[]", false, support)
		if err != nil || assigned != folder || !reflect.DeepEqual(args, folderArgs) {
			t.Fatalf("folder query diverged from assigned: %s vs %s, %v", assigned, folder, err)
		}
	}
	for _, lt := range []string{models.AssignedConversations, models.AssignedHighPriorityConversations, "folder-my_inbox"} {
		scope, id := sidebarListScope([]string{lt}, nil, nil, 101)
		if scope != "my_inbox" || id != 101 {
			t.Fatalf("incorrect employee scope: %s %d", scope, id)
		}
	}
	for _, lt := range []string{models.VisibleConversations, models.VisibleInternalConversations} {
		scope, _ := sidebarListScope([]string{lt}, nil, nil, 101)
		if scope != "" {
			t.Fatal("independent visibility view acquired assigned exclusivity")
		}
	}
}
