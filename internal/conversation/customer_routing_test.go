package conversation

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/template"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/jmoiron/sqlx"
)

type routingTeams struct{ db *sqlx.DB }

func (r routingTeams) Get(id int) (tmodels.Team, error) {
	var team tmodels.Team
	err := r.db.Get(&team, `SELECT id,name FROM teams WHERE id=$1`, id)
	return team, err
}
func (r routingTeams) GetMembers(id int) ([]tmodels.TeamMember, error) {
	var members []tmodels.TeamMember
	err := r.db.Select(&members, `SELECT user_id AS id,team_id FROM team_members WHERE team_id=$1`, id)
	return members, err
}
func (r routingTeams) UserBelongsToTeam(userID, teamID int) (bool, error) {
	var ok bool
	err := r.db.Get(&ok, `SELECT EXISTS(SELECT 1 FROM team_members WHERE user_id=$1 AND team_id=$2)`, userID, teamID)
	return ok, err
}

type routingUsers struct {
	threadingUsers
	db *sqlx.DB
}

func (r routingUsers) GetAgent(id int, _ string) (umodels.User, error) {
	var user umodels.User
	err := r.db.Get(&user, `SELECT id,first_name,last_name,type,enabled,email FROM users WHERE id=$1`, id)
	return user, err
}
func (r routingUsers) GetSystemUser() (umodels.User, error) { return r.GetAgent(103, "") }
func (r routingUsers) GetAgentCachedOrLoad(id int) (umodels.User, error) {
	u, err := r.GetAgent(id, "")
	u.Permissions = []string{"conversations:read", "conversations:read_assigned", "conversations:read_team_all"}
	if id == 103 {
		u.Roles = []string{"Kundensupport"}
	}
	if id == 104 {
		u.Roles = []string{"Admin"}
	}
	return u, err
}

type routingSLA struct{ slaStore }

func (routingSLA) EvaluateConversationSLA(int) error { return nil }
func routingFixture(t *testing.T, name string) *Manager {
	t.Helper()
	m := emailThreadFixture(t, name)
	m.db.MustExec(`INSERT INTO teams(id,name,conversation_assignment_type) VALUES(201,'Kundensupport','Manual'),(202,'Produktion','Manual'),(203,'Buchhaltung','Manual');
 INSERT INTO team_members(team_id,user_id) VALUES(201,103),(202,101),(202,102),(203,102);
 INSERT INTO users(id,type,email,first_name,last_name) VALUES(106,'agent','multi@example.test','Multi',''),(107,'agent','outsider@example.test','No access','');
 INSERT INTO team_members(team_id,user_id) VALUES(201,106),(202,106);
 INSERT INTO tags(id,name) VALUES(12,'🗑Spam'),(42,'Rekla')`)
	m.teamStore = routingTeams{m.db}
	m.userStore = routingUsers{db: m.db}
	m.slaStore = routingSLA{}
	var err error
	m.template, err = template.New(m.lo, m.db, nil, nil, nil, m.i18n)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func routingTicket(t *testing.T, m *Manager, status string, customer bool) string {
	t.Helper()
	var uuid string
	if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes,subject) SELECT 105,102,id,'{"customer_visibility":true,"visible_users":[103],"visibility_managers":[103],"keep":"yes"}','Routing test' FROM conversation_statuses WHERE name=$1 RETURNING uuid`, status); err != nil {
		t.Fatal(err)
	}
	if customer {
		m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, uuid)
	}
	return uuid
}
func routingAssert(t *testing.T, m *Manager, uuid string, team int, users []int64, status string, external bool) models.Conversation {
	t.Helper()
	c, err := m.GetConversation(0, uuid, "")
	if err != nil {
		t.Fatal(err)
	}
	if c.AssignedTeamID.Int != team || !slices.Equal([]int64(c.AssignedUserIDs), users) || c.Status.String != status || c.CustomerRedistributed != external || c.InboxID != 102 {
		t.Fatalf("state: team=%d users=%v status=%s external=%v inbox=%d", c.AssignedTeamID.Int, c.AssignedUserIDs, c.Status.String, c.CustomerRedistributed, c.InboxID)
	}
	if !c.CustomerTicket {
		t.Fatal("lost customer marker")
	}
	return c
}
func TestCustomerRoutingTransitions(t *testing.T) {
	m := routingFixture(t, "routing_transitions")
	actor := umodels.User{ID: 103}
	u := routingTicket(t, m, "Open", true)
	if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, []int64{101}, "Open", true)
	item, err := m.GetConversationListItem(u)
	if err != nil || !slices.Equal([]int64(item.AssignedUserIDs), []int64{101}) {
		t.Fatalf("WS list item lost canonical assignees: %+v %v", item, err)
	}

	if err := m.UpdateConversationUserAssignee(u, 102, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, []int64{102}, "Open", true)
	if err := m.UpdateConversationTeamAssignee(u, 202, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 202, nil, "Open", true)
	if err := m.UpdateConversationUserAssignee(u, 106, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 202, []int64{106}, "Open", true) // support membership cannot override Production
	if err := m.RemoveConversationAssignee(u, "user", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 202, nil, "Open", true)
	if err := m.UpdateConversationTeamAssignee(u, 203, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 203, nil, "Open", true)
	if err := m.UpdateConversationStatus(u, 0, "Closed", "", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 203, nil, "Closed", true)
	if err := m.ReturnCustomerToSupport(u, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, nil, "Open", false)
	// Support context may be left by selecting an external employee.
	if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, []int64{101}, "Open", true)
	if err := m.RemoveConversationAssignee(u, "user", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, nil, "Open", false)
	if err := m.UpdateConversationTeamAssignee(u, 202, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveConversationAssignee(u, "team", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, nil, "Open", false)
}
func TestCustomerRoutingReturnMatrix(t *testing.T) {
	m := routingFixture(t, "routing_returns")
	actor := umodels.User{ID: 103}
	for _, status := range []string{"Open", "Closed", "Resolved", "Snoozed"} {
		for _, method := range []string{"explicit", "team", "employee", "unassign"} {
			t.Run(status+"/"+method, func(t *testing.T) {
				u := routingTicket(t, m, status, true)
				// Use user-only context so explicit support employee assignment is unambiguous.
				if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
					t.Fatal(err)
				}
				routingAssert(t, m, u, 0, []int64{101}, status, true)
				var err error
				switch method {
				case "explicit":
					err = m.ReturnCustomerToSupport(u, actor)
				case "team":
					err = m.UpdateConversationTeamAssignee(u, 201, actor)
				case "employee":
					err = m.UpdateConversationUserAssignee(u, 103, actor)
				case "unassign":
					err = m.SetConversationUserAssignees(u, nil, actor)
				}
				if err != nil {
					t.Fatal(err)
				}
				var users []int64
				if method == "employee" {
					users = []int64{103}
				}
				team := 0
				if method == "team" {
					team = 201
				}
				c := routingAssert(t, m, u, team, users, "Open", false)
				if c.ClosedAt.Valid || c.ResolvedAt.Valid || c.SnoozedUntil.Valid {
					t.Fatal("return kept completion/snooze timestamps")
				}
			})
		}
	}
	// A multi-team employee alone is deliberately not guessed to be support.
	u := routingTicket(t, m, "Closed", true)
	if err := m.UpdateConversationUserAssignee(u, 106, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, []int64{106}, "Closed", true)
}
func TestCustomerRoutingVisibilityAndACL(t *testing.T) {
	m := routingFixture(t, "routing_visibility")
	actor := umodels.User{ID: 103}
	u := routingTicket(t, m, "Open", true)
	if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateConversationTeamAssignee(u, 203, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.ReturnCustomerToSupport(u, actor); err != nil {
		t.Fatal(err)
	}
	c := routingAssert(t, m, u, 0, nil, "Open", false)
	var a map[string]any
	json.Unmarshal(c.CustomAttributes, &a)
	for _, id := range []int{101, 102, 103} {
		if !slices.Contains(a["visible_users"].([]any), any(float64(id))) {
			t.Fatalf("missing permanent visibility: %d", id)
		}
	}
	if a["keep"] != "yes" || !slices.Contains(a["visibility_managers"].([]any), any(float64(103))) {
		t.Fatal("changed existing authority/attributes")
	}
	former := umodels.User{ID: 101, Permissions: []string{"conversations:read", "conversations:read_assigned"}}
	if !authz.CanReadConversation(former, c) {
		t.Fatal("former assignee lost historical visibility")
	}
	if err := m.RemoveVisibleUser(u, 101); err != nil {
		t.Fatal(err)
	}
	c, _ = m.GetConversation(0, u, "")
	if authz.CanReadConversation(former, c) {
		t.Fatal("removed former employee retained access")
	}
	// A stale general attributes save cannot restore ACL fields or forge managers.
	if err := m.UpdateConversationCustomAttributes(u, map[string]any{"visible_users": []int{101}, "visibility_managers": []int{101}, "customer_visibility": false, "keep": "updated"}); err != nil {
		t.Fatal(err)
	}
	c, _ = m.GetConversation(0, u, "")
	json.Unmarshal(c.CustomAttributes, &a)
	if authz.CanReadConversation(former, c) || a["customer_visibility"] != true || a["keep"] != "updated" {
		t.Fatal("general attribute save changed authority")
	}
	// Reading / broadcasting never silently re-adds visibility.
	for range 2 {
		m.GetConversationListItem(u)
		m.BroadcastConversationUpdate(u, map[string]any{"status": "Open"})
	}
	c, _ = m.GetConversation(0, u, "")
	if authz.CanReadConversation(former, c) {
		t.Fatal("refresh re-added visibility")
	}
	// Explicit visibility removal is allowed while assigned; independent assignment
	// access remains, then disappears after forwarding unless explicitly shared.
	if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveVisibleUser(u, 101); err != nil {
		t.Fatal(err)
	}
	if err := m.SetConversationUserAssignees(u, []int{101}, actor); err != nil {
		t.Fatal(err)
	}
	c, _ = m.GetConversation(0, u, "")
	json.Unmarshal(c.CustomAttributes, &a)
	if slices.Contains(a["visible_users"].([]any), any(float64(101))) {
		t.Fatal("no-op assignment restored manually removed visibility")
	}
	if !authz.CanReadConversation(former, c) {
		t.Fatal("assignment access no longer independent")
	}
	allowed, err := m.FilterAuthorizedListUUIDs(101, []string{u})
	if err != nil || len(allowed) != 1 {
		t.Fatalf("assigned search/subscription access: %v %v", allowed, err)
	}
	assignedRows, err := m.GetConversations(101, 101, nil, []string{models.AssignedConversations}, "", "", "[]", 1, 25)
	if err != nil || len(assignedRows) != 1 {
		t.Fatalf("assigned list after explicit visibility removal: %v %v", assignedRows, err)
	}

	if err := m.UpdateConversationTeamAssignee(u, 203, actor); err != nil {
		t.Fatal(err)
	}
	c, _ = m.GetConversation(0, u, "")
	if authz.CanReadConversation(former, c) {
		t.Fatal("forwarded employee kept ungranted access")
	}
	for _, viewer := range []umodels.User{{ID: 103, Roles: []string{"Kundensupport"}}, {ID: 104, Roles: []string{"Admin"}}} {
		if !authz.CanReadConversation(viewer, c) {
			t.Fatal("lost existing support/admin access")
		}
	}
	if authz.CanReadConversation(umodels.User{ID: 107}, c) {
		t.Fatal("outsider gained access")
	}
	rows, err := m.GetConversations(101, 101, nil, []string{models.VisibleConversations}, "", "", "[]", 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatal("removed visibility still in visible list")
	}
}
func TestCustomerRoutingListsFoldersSpamPagination(t *testing.T) {
	m := routingFixture(t, "routing_lists")
	actor := umodels.User{ID: 103}
	for i := 0; i < 7; i++ {
		u := routingTicket(t, m, "Open", true)
		m.db.MustExec(`UPDATE conversations SET assigned_team_id=202,priority_id=101 WHERE uuid=$1`, u)
		m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,42 FROM conversations WHERE uuid=$1`, u)
	}
	normal := routingTicket(t, m, "Open", true)
	done := routingTicket(t, m, "Closed", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, done)
	sleeping := routingTicket(t, m, "Snoozed", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, sleeping)
	spam := routingTicket(t, m, "Open", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, spam)
	if err := m.SetConversationTags(spam, amodels.ActionAddTags, []string{"🗑Spam"}, actor); err != nil {
		t.Fatal(err)
	}
	// Existing customer folders must not hide any redistributed ticket from All.
	folders, err := m.MutateSidebarFolder(103, "customer", 0, 0, "create", SidebarFolder{Name: "Rekla", Filters: json.RawMessage(folderTagFilter)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for page := 1; page <= 3; page++ {
		rows, err := m.GetConversations(103, 103, nil, []string{models.CustomerRedistributedConversations}, "ASC", "conversations.created_at", "[]", page, 3)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range rows {
			if r.Total != 8 || !r.CustomerRedistributedActive || seen[r.UUID] {
				t.Fatalf("incorrect totals/duplicate: %+v", r)
			}
			seen[r.UUID] = true
		}
	}
	if len(seen) != 8 || !seen[sleeping] || seen[done] || seen[spam] || seen[normal] {
		t.Fatal("routing/status/spam boundary failed")
	}
	rows, err := m.GetCustomerConversationsList(103, false, "", "", "[]", 1, 25)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if seen[r.UUID] || r.UUID == spam {
			t.Fatal("external active leaked into ordinary customer list")
		}
	}
	counts, err := m.SidebarFolderCounts(103, "customer", 0)
	if err != nil {
		t.Fatal(err)
	}
	if counts[folders[0].ID] != 0 {
		t.Fatal("customer folder badge includes redistributed tickets")
	}
	if _, err = m.GetConversations(107, 107, nil, []string{models.CustomerRedistributedConversations}, "", "", "[]", 1, 25); err == nil {
		t.Fatal("unauthorized redistributed management list")
	}
	if err = m.ReturnCustomerToSupport(spam, actor); err == nil {
		t.Fatal("returned Spam")
	}
	if err = m.SetConversationTags(spam, amodels.ActionRemoveTags, []string{"🗑Spam"}, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, spam, 202, nil, "Open", true)
	if err = m.ReturnCustomerToSupport(spam, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, spam, 0, nil, "Open", false)
	// Scope cannot be smuggled into a new stored folder without a migration.
	if _, err = m.MutateSidebarFolder(107, "redistributed", 0, 0, "create", SidebarFolder{Name: "Denied", Filters: json.RawMessage(folderTagFilter)}, nil); err == nil {
		t.Fatal("unauthorized redistributed scope accepted")
	}
}
func TestCustomerRoutingRollbackAndTeamlessReturn(t *testing.T) {
	m := routingFixture(t, "routing_rollback")
	actor := umodels.User{ID: 103}
	u := routingTicket(t, m, "Closed", true)
	if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`CREATE FUNCTION reject_route_open() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.status_id<>OLD.status_id THEN RAISE EXCEPTION 'test routing rollback'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_route_open BEFORE UPDATE ON conversations FOR EACH ROW EXECUTE FUNCTION reject_route_open()`)
	if err := m.ReturnCustomerToSupport(u, actor); err == nil {
		t.Fatal("expected rollback")
	}
	routingAssert(t, m, u, 0, []int64{101}, "Closed", true)
	m.db.MustExec(`DROP TRIGGER reject_route_open ON conversations; UPDATE teams SET name='Renamed support' WHERE id=201`)
	if err := m.RemoveConversationAssignee(u, "user", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, nil, "Open", false)
}
func TestCustomerRoutingIncomingAndInvalidAssignee(t *testing.T) {
	m := routingFixture(t, "routing_incoming")
	actor := umodels.User{ID: 103}
	// Real email persistence/threading is used before the existing reopen hook.
	msg := importMail(t, m, threadMail("route-parent@example.test", ""))
	u := msg.ConversationUUID
	m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1 ON CONFLICT DO NOTHING`, u)
	if err := m.UpdateConversationTeamAssignee(u, 202, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateConversationStatus(u, 0, "Closed", "", actor); err != nil {
		t.Fatal(err)
	}
	child := importMail(t, m, threadMail("route-child@example.test", "route-parent@example.test"))
	if child.ConversationUUID != u {
		t.Fatal("reply moved thread")
	}
	routingAssert(t, m, u, 202, nil, "Open", true)
	if err := m.ReOpenConversation(u, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 202, nil, "Open", true)
	if err := m.SetConversationTags(u, amodels.ActionAddTags, []string{"🗑Spam"}, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.ProcessIncomingMessageHooks(u, false); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 202, nil, "Closed", true)
	for _, status := range []string{"Closed", "Open"} {
		t.Run(status, func(t *testing.T) {
			invalid := routingTicket(t, m, status, true)
			m.db.MustExec(`UPDATE conversations SET assigned_user_id=106 WHERE uuid=$1;`, invalid)
			m.db.MustExec(`INSERT INTO conversation_assignees(conversation_id,user_id) SELECT id,106 FROM conversations WHERE uuid=$1`, invalid)
			m.db.MustExec(`UPDATE users SET enabled=false WHERE id=106`)
			if err := m.ReOpenConversation(invalid, actor); err != nil {
				t.Fatal(err)
			}
			routingAssert(t, m, invalid, 0, nil, "Open", false)
			m.db.MustExec(`UPDATE users SET enabled=true WHERE id=106`)
		})
	}
}
func TestCustomerRoutingConcurrentChanges(t *testing.T) {
	m := routingFixture(t, "routing_concurrent")
	actor := umodels.User{ID: 103}
	for i := 0; i < 6; i++ {
		u := routingTicket(t, m, "Closed", true)
		m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, u)
		var wg sync.WaitGroup
		errors := make(chan error, 4)
		actions := []func() error{func() error { return m.ReturnCustomerToSupport(u, actor) }, func() error { return m.UpdateConversationTeamAssignee(u, 203, actor) }, func() error { return m.RemoveVisibleUser(u, 101) }, func() error {
			_, _, _, err := m.mutateTagsWithSpam(u, amodels.ActionAddTags, []string{"🗑Spam"}, actor)
			return err
		}}
		for _, action := range actions {
			wg.Add(1)
			go func(f func() error) { defer wg.Done(); errors <- f() }(action)
		}
		wg.Wait()
		close(errors)
		// Explicit return is permitted to fail if Spam obtained the lock first.
		for err := range errors {
			if err != nil {
				var input envelope.Error
				if !stderrors.As(err, &input) || input.ErrorType != envelope.InputError || input.Message != m.i18n.T("conversation.routing.returnUnavailable") {
					t.Fatal(err)
				}
				t.Logf("serialized rejection: %v", err)
			}
		}
		c, err := m.GetConversation(0, u, "")
		if err != nil {
			t.Fatal(err)
		}
		if c.Status.String != "Closed" || len(c.AssignedUserIDs) != 0 || (c.AssignedTeamID.Int != 0 && c.AssignedTeamID.Int != 203) {
			t.Fatalf("inconsistent concurrent result: %+v", c)
		}
		var attrs map[string]any
		json.Unmarshal(c.CustomAttributes, &attrs)
		if slices.Contains(attrs["visible_users"].([]any), any(float64(101))) {
			t.Fatal("lost concurrent visibility removal")
		}
		if _, ok := attrs["_spam_previous_state"]; !ok {
			t.Fatal("lost Spam snapshot")
		}
	}
}
func TestCustomerRoutingOnlyProtectedMarker(t *testing.T) {
	m := routingFixture(t, "routing_marker")
	actor := umodels.User{ID: 103}
	u := routingTicket(t, m, "Closed", false)
	if handled, err := m.mutateCustomerAssignment(u, customerAssignment{kind: "return"}, actor); handled || err != nil {
		t.Fatalf("noncustomer affected: %v %v", handled, err)
	}
	if err := m.ReturnCustomerToSupport(u, actor); err == nil {
		t.Fatal("noncustomer returned")
	}
	// Both supported historical numeric IDs use the same existing protected name.
	m.db.MustExec(`UPDATE tags SET id=9 WHERE name='🦽Kundenticket'`)
	m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,9 FROM conversations WHERE uuid=$1`, u)
	if err := m.UpdateConversationUserAssignee(u, 101, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, u, 0, []int64{101}, "Closed", true)
	if err := m.SetConversationTags(u, amodels.ActionRemoveTags, []string{"🦽Kundenticket"}, actor); err == nil {
		t.Fatal("removed protected marker")
	}
}

func TestCustomerRoutingWebSocketAuthorization(t *testing.T) {
	m := routingFixture(t, "routing_ws")
	u := routingTicket(t, m, "Open", true)
	clients := map[int]*ws.Client{}
	for _, id := range []int{101, 102, 103, 104, 107} {
		client := &ws.Client{ID: id, Hub: m.wsHub, Send: make(chan wsmodels.WSMessage, 100)}
		m.wsHub.AddClient(client)
		clients[id] = client
		defer m.wsHub.RemoveClient(client)
	}
	actor := umodels.User{ID: 103}
	if err := m.UpdateConversationTeamAssignee(u, 203, actor); err != nil {
		t.Fatal(err)
	}
	// A newly authorized team member has no previous subscription but receives
	// the same membership event as support. An unrelated user gets no payload.
	for id, client := range clients {
		received := false
		for len(client.Send) > 0 {
			msg := <-client.Send
			var event struct {
				Type string         `json:"type"`
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(msg.Data, &event); err != nil {
				t.Fatal(err)
			}
			if event.Type == "conversation_update" && event.Data["routing_changed"] == true {
				received = true
			}
		}
		want := id == 102 || id == 103 || id == 104
		if received != want {
			t.Fatalf("recipient %d: got=%v want=%v", id, received, want)
		}
	}
	m.wsHub.SubscribeListReplace(clients[102], []string{u})
	if err := m.ReturnCustomerToSupport(u, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveVisibleUser(u, 102); err != nil {
		t.Fatal(err)
	}
	for len(clients[102].Send) > 0 {
		<-clients[102].Send
	}
	m.BroadcastConversationUpdate(u, map[string]any{"status": "Open"})
	select {
	case msg := <-clients[102].Send:
		t.Fatalf("revoked subscriber received payload: %s", msg.Data)
	default:
	}
	// Zero authorized recipients must not fall through to broadcast-to-everyone.
	for _, id := range []int{101, 102, 103, 104} {
		m.wsHub.RemoveClient(clients[id])
	}
	m.broadcastTagMembershipUpdate(u, map[string]any{"routing_changed": true})
	select {
	case msg := <-clients[107].Send:
		t.Fatalf("empty recipients leaked: %s", msg.Data)
	default:
	}
}

func TestCustomerRoutingConcurrentAssignmentsAndIncoming(t *testing.T) {
	m := routingFixture(t, "routing_parallel_assignment")
	actor := umodels.User{ID: 103}
	for i := 0; i < 5; i++ {
		u := routingTicket(t, m, "Closed", true)
		m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, u)
		var wg sync.WaitGroup
		errors := make(chan error, 3)
		actions := []func() error{func() error { return m.UpdateConversationTeamAssignee(u, 203, actor) }, func() error { return m.ReturnCustomerToSupport(u, actor) }, func() error { return m.ReOpenConversation(u, actor) }}
		for _, action := range actions {
			wg.Add(1)
			go func(f func() error) { defer wg.Done(); errors <- f() }(action)
		}
		wg.Wait()
		close(errors)
		for err := range errors {
			if err != nil {
				t.Fatal(err)
			}
		}
		c, err := m.GetConversation(0, u, "")
		if err != nil {
			t.Fatal(err)
		}
		if c.Status.String != "Open" || len(c.AssignedUserIDs) != 0 || (c.AssignedTeamID.Int != 0 && c.AssignedTeamID.Int != 203) {
			t.Fatalf("lost atomic routing: %+v", c)
		}
	}
	u := routingTicket(t, m, "Closed", true)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, id := range []int{101, 102} {
		wg.Add(1)
		go func(id int) { defer wg.Done(); errs <- m.UpdateConversationUserAssignee(u, id, actor) }(id)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	c, err := m.GetConversation(0, u, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.AssignedUserIDs) != 1 || c.AssignedUserIDs[0] != int64(c.AssignedUserID.Int) || !c.CustomerRedistributed || c.Status.String != "Closed" {
		t.Fatalf("conflicting assignee projections: %+v", c)
	}
}

func TestCustomerRoutingNonCustomerUpdatesUnchanged(t *testing.T) {
	m := routingFixture(t, "routing_ordinary_ws")
	uuid := routingTicket(t, m, "Open", false)
	client := &ws.Client{ID: 103, Hub: m.wsHub, Send: make(chan wsmodels.WSMessage, 10)}
	m.wsHub.AddClient(client)
	defer m.wsHub.RemoveClient(client)
	m.wsHub.SubscribeListReplace(client, []string{uuid})
	m.BroadcastConversationUpdate(uuid, map[string]any{"status": "Open"})
	select {
	case message := <-client.Send:
		var event struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(message.Data, &event); err != nil {
			t.Fatal(err)
		}
		if len(event.Data) != 2 || event.Data["status"] != "Open" || event.Data["uuid"] != uuid {
			t.Fatalf("noncustomer status event changed: %s", message.Data)
		}
	default:
		t.Fatal("missing ordinary status event")
	}
}

func TestCustomerRoutingWithoutSupportTeam(t *testing.T) {
	m := routingFixture(t, "routing_no_team")
	m.db.MustExec(`DELETE FROM team_members WHERE team_id=201; DELETE FROM teams WHERE id=201`)
	actor := umodels.User{ID: 103}
	for _, start := range []string{"Open", "Closed", "Resolved", "Snoozed"} {
		for _, kind := range []string{"return", "remove"} {
			t.Run(start+"/"+kind, func(t *testing.T) {
				uuid := routingTicket(t, m, start, true)
				if err := m.UpdateConversationTeamAssignee(uuid, 202, actor); err != nil {
					t.Fatal(err)
				}
				if err := m.UpdateConversationUserAssignee(uuid, 101, actor); err != nil {
					t.Fatal(err)
				}
				before, _ := m.GetConversation(0, uuid, "")
				var err error
				if kind == "return" {
					err = m.ReturnCustomerToSupport(uuid, actor)
				} else {
					err = m.SetConversationUserAssignees(uuid, nil, actor)
					if err == nil {
						routingAssert(t, m, uuid, 202, nil, start, true)
						err = m.RemoveConversationAssignee(uuid, "team", actor)
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				after := routingAssert(t, m, uuid, 0, nil, "Open", false)
				var beforeAttrs, afterAttrs map[string]any
				json.Unmarshal(before.CustomAttributes, &beforeAttrs)
				json.Unmarshal(after.CustomAttributes, &afterAttrs)
				if !reflect.DeepEqual(beforeAttrs, afterAttrs) {
					t.Fatalf("return changed visibility/authority: %s -> %s", before.CustomAttributes, after.CustomAttributes)
				}
				filter := `[{"model":"conversation_statuses","field":"name","operator":"equals","value":"Open"}]`
				rows, err := m.GetCustomerConversationsList(103, false, "", "", filter, 1, 100)
				if err != nil {
					t.Fatal(err)
				}
				found := 0
				for _, r := range rows {
					if r.UUID == uuid {
						found++
					}
				}
				if found != 1 {
					t.Fatalf("returned customer not present exactly once: %d", found)
				}
			})
		}
	}
	// Removing a team alone must preserve a still-valid individual owner.
	remaining := routingTicket(t, m, "Closed", true)
	if err := m.UpdateConversationTeamAssignee(remaining, 202, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateConversationUserAssignee(remaining, 101, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveConversationAssignee(remaining, "team", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, remaining, 0, []int64{101}, "Closed", true)
	if err := m.RemoveConversationAssignee(remaining, "user", actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, remaining, 0, nil, "Open", false)
	// A support role without team memberships provides an unambiguous employee
	// context; the same role in an external team must never override that team.
	m.db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Kundensupport','{}'); INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport'`)
	uuid := routingTicket(t, m, "Closed", true)
	if err := m.UpdateConversationUserAssignee(uuid, 101, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateConversationUserAssignee(uuid, 103, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, uuid, 0, []int64{103}, "Open", false)
	if err := m.UpdateConversationTeamAssignee(uuid, 202, actor); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdateConversationUserAssignee(uuid, 103, actor); err != nil {
		t.Fatal(err)
	}
	routingAssert(t, m, uuid, 202, []int64{103}, "Open", true)
}

func TestRedistributedFoldersScopeCountsAndFilters(t *testing.T) {
	m := routingFixture(t, "redistributed_folders")
	actor := umodels.User{ID: 103}
	filter := func(field, value string) json.RawMessage {
		return json.RawMessage(fmt.Sprintf(`[{"model":"conversations","field":%q,"operator":"equals","value":%q}]`, field, value))
	}
	// A lower-priority broad folder overlaps all three Redistributed tickets.
	first, err := m.MutateSidebarFolder(103, "redistributed", 0, 0, "create", SidebarFolder{Name: "Rekla", Filters: json.RawMessage(folderTagFilter)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	high, err := m.MutateSidebarFolder(103, "redistributed", 0, 0, "create", SidebarFolder{Name: "High", Filters: filter("priority_id", "101"), SortOrder: 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(high) != 2 {
		t.Fatal(high)
	}
	uuid := routingTicket(t, m, "Open", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202,priority_id=101 WHERE uuid=$1`, uuid)
	m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,42 FROM conversations WHERE uuid=$1`, uuid)
	rest := routingTicket(t, m, "Snoozed", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=203 WHERE uuid=$1`, rest)
	normal := routingTicket(t, m, "Open", true)
	done := routingTicket(t, m, "Closed", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, done)
	spam := routingTicket(t, m, "Open", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, spam)
	if err := m.SetConversationTags(spam, amodels.ActionAddTags, []string{"🗑Spam"}, actor); err != nil {
		t.Fatal(err)
	}
	nonCustomer := routingTicket(t, m, "Open", false)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, nonCustomer)
	private := routingTicket(t, m, "Open", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202,inbox_id=101,custom_attributes='{"customer_visibility":true,"access_mode":"personal","visible_users":[103]}' WHERE uuid=$1`, private)
	// Verify aggregate counts before pagination and folder exclusivity.
	for page := 1; page <= 2; page++ {
		rows, err := m.GetConversations(103, 103, nil, []string{models.CustomerRedistributedConversations}, "ASC", "conversations.created_at", "[]", page, 1)
		if err != nil || len(rows) != 1 || rows[0].Total != 2 {
			t.Fatalf("aggregate %v %v", rows, err)
		}
	}
	counts, err := m.SidebarFolderCounts(103, "redistributed", 0)
	if err != nil {
		t.Fatal(err)
	}
	if counts[first[0].ID] != 1 || counts[high[1].ID] != 0 || counts[-1] != 1 {
		t.Fatalf("exclusive totals: %v", counts)
	}
	rows, err := m.GetConversations(103, 103, nil, []string{models.CustomerRedistributedUnfiledConversations}, "", "", "[]", 1, 25)
	if err != nil || len(rows) != 1 || rows[0].UUID != rest {
		t.Fatalf("unfiled: %v %v", rows, err)
	}
	rows, err = m.GetSidebarFolderConversations(103, "redistributed", 0, first[0].ID, "", "", 1, 25)
	if err != nil || len(rows) != 1 || rows[0].UUID != uuid {
		t.Fatalf("folder: %v %v", rows, err)
	}
	// Reordering changes the exclusive winner, without changing All or routing.
	if _, err := m.MutateSidebarFolder(103, "redistributed", 0, 0, "reorder", SidebarFolder{}, []int{high[1].ID, first[0].ID}); err != nil {
		t.Fatal(err)
	}
	counts, err = m.SidebarFolderCounts(103, "redistributed", 0)
	if err != nil || counts[high[1].ID] != 1 || counts[first[0].ID] != 0 {
		t.Fatalf("reorder totals: %v %v", counts, err)
	}
	// Test team and canonical multi-employee filters through the existing compiler.
	if err := m.UpdateConversationUserAssignee(uuid, 101, actor); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, field, value string }{{"Team", "assigned_team_id", "202"}, {"Employee", "assigned_user_id", "101"}} {
		folders, err := m.MutateSidebarFolder(103, "redistributed", 0, 0, "create", SidebarFolder{Name: tc.name, Filters: filter(tc.field, tc.value)}, nil)
		if err != nil {
			t.Fatal(err)
		}
		var id int
		for _, f := range folders {
			if f.Name == tc.name {
				id = f.ID
			}
		}
		rows, err = m.GetSidebarFolderConversations(103, "redistributed", 0, id, "", "", 1, 25)
		if err != nil || len(rows) != 1 || rows[0].UUID != uuid {
			t.Fatalf("%s filter: %v %v", tc.name, rows, err)
		}
		if _, err = m.MutateSidebarFolder(103, "redistributed", 0, id, "delete", SidebarFolder{}, nil); err != nil {
			t.Fatal(err)
		}
	}
	// Returning removes the ticket from every redistributed projection.
	if err := m.ReturnCustomerToSupport(uuid, actor); err != nil {
		t.Fatal(err)
	}
	rows, err = m.GetConversations(103, 103, nil, []string{models.CustomerRedistributedConversations}, "", "", "[]", 1, 25)
	if err != nil || len(rows) != 1 || rows[0].UUID != rest {
		t.Fatalf("return aggregate: %v %v", rows, err)
	}
	if _, err = m.ListSidebarFolders(107, "redistributed", 0); err == nil {
		t.Fatal("outsider managed redistributed folders")
	}
	if _, err = m.ListSidebarFolders(103, "redistributed", 202); err == nil {
		t.Fatal("scope ID boundary bypassed")
	}
	// Existing customer definitions stay in their own scope.
	customer, err := m.MutateSidebarFolder(103, "customer", 0, 0, "create", SidebarFolder{Name: "Customer", Filters: json.RawMessage(folderTagFilter)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.GetSidebarFolderConversations(103, "redistributed", 0, customer[0].ID, "", "", 1, 25); err == nil {
		t.Fatal("cross-scope folder accepted")
	}
	_ = normal
	_ = done
}
