package conversation

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

type personalUsers struct{ userStore }

func (personalUsers) GetEnabledAgentIDsByRoleName(string) ([]int, error) { return []int{103}, nil }

func (personalUsers) GetSystemUser() (umodels.User, error) {
	return umodels.User{}, fmt.Errorf("no system actor in test")
}

func (personalUsers) GetAgentCachedOrLoad(id int) (umodels.User, error) {
	u := umodels.User{ID: id, Enabled: true}
	if id == 103 {
		u.Roles = []string{"Kundensupport"}
	}
	if id == 104 {
		u.Roles = []string{"Admin"}
	}
	return u, nil
}

type personalSettings struct{ settingsStore }

func (personalSettings) Get(string) (types.JSONText, error) { return types.JSONText(`"UTC"`), nil }

func TestPersonalIncomingAndVisibility(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_conversation")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), userStore: personalUsers{}, settingsStore: personalSettings{}, wsHub: ws.NewHub(&lo, nil)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	engine, err := automation.New(automation.Opts{DB: db, Lo: &lo, I18n: m.i18n})
	if err != nil {
		t.Fatal(err)
	}
	m.automation = engine
	// Service-mail classification must lose to personal access.
	db.MustExec(`INSERT INTO service_email_addresses(address) VALUES('contact@example.com')`)
	in := models.IncomingMessage{InboxID: 101, Subject: "Personal", Contact: models.IncomingContact{ID: 105, Email: null.StringFrom("contact@example.com")}}
	id, uuid, created, err := m.findOrCreateConversation(in)
	if err != nil || !created {
		t.Fatalf("create: %v %v", created, err)
	}
	conv, err := m.GetConversation(id, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var attrs map[string]any
	if err := json.Unmarshal(conv.CustomAttributes, &attrs); err != nil {
		t.Fatal(err)
	}
	if attrs["access_mode"] != "personal" || attrs["customer_visibility"] != false || attrs["creator_id"] != float64(101) || attrs["owner_user_id"] != float64(101) {
		t.Fatalf("attributes: %v", attrs)
	}
	for _, key := range []string{"visible_users", "visibility_managers"} {
		ids := attrs[key].([]any)
		if len(ids) != 1 || ids[0] != float64(101) {
			t.Fatalf("%s: %v", key, ids)
		}
	}
	var assigned []int
	if err := db.Select(&assigned, `SELECT user_id FROM conversation_assignees WHERE conversation_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if len(assigned) != 1 || assigned[0] != 101 || conv.AssignedUserID.Int != 101 {
		t.Fatalf("assignment: %v %+v", assigned, conv.AssignedUserID)
	}
	var tags int
	db.Get(&tags, `SELECT count(*) FROM conversation_tags WHERE conversation_id=$1`, id)
	if tags != 0 {
		t.Fatalf("personal tags: %d", tags)
	}
	viewers := []struct {
		id    int
		roles []string
		want  bool
	}{{101, nil, true}, {102, nil, false}, {103, []string{"Kundensupport"}, false}, {104, []string{"Admin"}, true}}
	for _, v := range viewers {
		u := umodels.User{ID: v.id, Roles: v.roles, Permissions: []string{"conversations:read", "conversations:read_all", "conversations:read_team_all"}}
		if got := authz.CanReadConversation(u, conv); got != v.want {
			t.Errorf("visibility %d: %v", v.id, got)
		}
		var allowed []string
		if err := m.q.FilterAuthorizedListUUIDs.Select(&allowed, pq.Array([]string{uuid}), v.id, pq.Array([]int{}), true, true, true, true, true, true, v.id == 104, v.id == 103); err != nil {
			t.Fatal(err)
		}
		if (len(allowed) > 0) != v.want {
			t.Errorf("SQL visibility %d: %v", v.id, allowed)
		}
	}
	db.MustExec(`INSERT INTO teams(id,name,conversation_assignment_type) VALUES(1,'Team','Manual'); INSERT INTO team_members(team_id,user_id) VALUES(1,102)`)
	db.MustExec(`UPDATE conversations SET assigned_team_id=1,priority_id=101 WHERE id=$1`, id)
	// Execute each real list query: catches missing switch cases and SQL errors.
	for _, list := range []string{models.CustomerConversations, models.CustomerHighPriorityConversations, models.ServiceMailConversations, models.UnassignedConversations, models.TeamUnassignedConversations, models.TeamAllConversations, models.TeamHighPriorityConversations, models.VisibleInternalConversations, models.AssignedConversations, models.VisibleConversations, models.AllConversations} {
		for _, viewer := range []int{101, 102, 104} {
			query, args, err := m.makeConversationsListQuery(viewer, viewer, []int{1}, []string{list}, m.q.GetConversations, "", "", 1, 20, "", viewer == 104, false)
			if err != nil {
				t.Fatal(list, err)
			}
			var rows []models.ConversationListItem
			if err := db.Select(&rows, query, args...); err != nil {
				t.Fatal(list, err)
			}
			want := (list == models.AssignedConversations || list == models.VisibleConversations || list == models.VisibleInternalConversations) && viewer == 101 || (list == models.AllConversations || list == models.TeamAllConversations || list == models.TeamHighPriorityConversations) && (viewer == 101 || viewer == 104)
			if (len(rows) > 0) != want {
				t.Errorf("list %s viewer %d: %d rows, want present %v", list, viewer, len(rows), want)
			}
		}
	}
	// Even an old subscription is rechecked before personal events are delivered.
	clients := []*ws.Client{}
	for _, id := range []int{101, 102, 103, 104} {
		client := &ws.Client{ID: id, Hub: m.wsHub, Send: make(chan wsmodels.WSMessage, 20)}
		m.wsHub.AddClient(client)
		m.wsHub.SubscribeListReplace(client, []string{uuid})
		clients = append(clients, client)
	}
	if got := m.authorizedConversationSubscribers(uuid); len(got) != 2 {
		t.Fatalf("event recipients: %v", got)
	}
	if err := m.AddVisibleUser(uuid, 102); err != nil {
		t.Fatal(err)
	}
	shared, _ := m.GetConversation(id, "", "")
	if !authz.CanReadConversation(umodels.User{ID: 102}, shared) {
		t.Fatal("explicit sharing failed")
	}
	if got := m.authorizedConversationSubscribers(uuid); len(got) != 3 {
		t.Fatal("sharing did not authorize events")
	}
	if err := m.RemoveVisibleUser(uuid, 102); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveVisibleUser(uuid, 101); err != nil {
		t.Fatal(err)
	}
	protected, _ := m.GetConversation(id, "", "")
	if !authz.CanReadConversation(umodels.User{ID: 101}, protected) {
		t.Fatal("creator removed")
	}
	if got := m.authorizedConversationSubscribers(uuid); len(got) != 2 {
		t.Fatal("revoked subscriber still receives events")
	}
	for _, client := range clients {
		m.wsHub.RemoveClient(client)
	}
	// Changing the inbox owner leaves the original conversation and replies untouched.
	db.MustExec(`UPDATE inboxes SET owner_user_id=102 WHERE id=101`)
	db.MustExec(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,source_id) VALUES($1,'incoming','received',105,'contact','First','personal-source')`, id)
	var messageID int
	if err := db.Get(&messageID, `SELECT id FROM conversation_messages WHERE conversation_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	mediaConv, err := m.GetConversationByMessageID(messageID)
	if err != nil {
		t.Fatal(err)
	}
	if !isPersonalConversation(mediaConv.CustomAttributes) || authz.CanReadConversation(umodels.User{ID: 102}, mediaConv) || !authz.CanReadConversation(umodels.User{ID: 101}, mediaConv) {
		t.Fatal("media lookup lost personal visibility")
	}
	in.InReplyTo = "personal-source"
	again, _, isNew, err := m.findOrCreateConversation(in)
	if err != nil || isNew || again != id {
		t.Fatalf("reply: %d %v %v", again, isNew, err)
	}
	if err := m.ProcessIncomingMessageHooks(uuid, false); err != nil {
		t.Fatal(err)
	}
	db.Get(&tags, `SELECT count(*) FROM conversation_tags WHERE conversation_id=$1`, id)
	if tags != 0 {
		t.Fatal("reply added system tags")
	}
	updated, _ := m.GetConversation(id, "", "")
	if string(updated.CustomAttributes) != string(conv.CustomAttributes) {
		t.Fatal("reply changed visibility")
	}
	in.InReplyTo = ""
	_, newUUID, _, err := m.findOrCreateConversation(in)
	if err != nil {
		t.Fatal(err)
	}
	newer, _ := m.GetConversation(0, newUUID, "")
	if newer.AssignedUserID.Int != 102 {
		t.Fatal("new owner not applied")
	}
	// The same source ID is not deduplicated or threaded across a personal boundary.
	if exists, err := m.MessageExists("personal-source", 102); err != nil || exists {
		t.Fatalf("cross inbox dedup: %v %v", exists, err)
	}
	in.InboxID = 102
	in.InReplyTo = "personal-source"
	publicID, publicUUID, isNew, err := m.findOrCreateConversation(in)
	if err != nil || !isNew || publicID == id {
		t.Fatalf("cross inbox threading: %v", err)
	}
	public, _ := m.GetConversation(0, publicUUID, "")
	var publicAttrs map[string]any
	json.Unmarshal(public.CustomAttributes, &publicAttrs)
	if publicAttrs["customer_visibility"] != true {
		t.Fatal("public visibility changed")
	}
	var tagNames []string
	db.Select(&tagNames, `SELECT t.name FROM tags t JOIN conversation_tags ct ON ct.tag_id=t.id WHERE ct.conversation_id=$1`, publicID)
	if len(tagNames) != 1 || tagNames[0] != "🧷Service-Mail" {
		t.Fatalf("public service tags: %v", tagNames)
	}
	db.MustExec(`DELETE FROM service_email_addresses`)
	in.InReplyTo = ""
	customerID, _, _, err := m.findOrCreateConversation(in)
	if err != nil {
		t.Fatal(err)
	}
	db.Select(&tagNames, `SELECT t.name FROM tags t JOIN conversation_tags ct ON ct.tag_id=t.id WHERE ct.conversation_id=$1`, customerID)
	if len(tagNames) != 1 || tagNames[0] != "🦽Kundenticket" {
		t.Fatalf("public customer tags: %v", tagNames)
	}
}
