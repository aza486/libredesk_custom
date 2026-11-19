package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
)

func internalCreateBody(inboxID, agentID, teamID int) string {
	return fmt.Sprintf(`{"inbox_id":%d,"agent_id":%d,"team_id":%d,"subject":"Internal creation","content":"<p>Internal message</p>","initiator":"agent","custom_attributes":{"private":true,"internal":true}}`, inboxID, agentID, teamID)
}

func TestInternalConversationConfiguredInboxHTTP(t *testing.T) {
	app, db := systemAttributesHTTPFixture(t, "f02_configured")
	// An unrelated real inbox may occupy 3; no dummy IDs are needed by the app.
	db.MustExec(`INSERT INTO inboxes(id,name,channel,"from") VALUES
 (3,'Other public inbox','email','other-inbox@example.com'),
 (3087,'Configured internal inbox','email','internal-inbox@example.com');
 UPDATE users SET external_user_id='external-customer' WHERE id=105`)
	if err := db.Get(&app.internalConversation.InboxUUID, `SELECT uuid FROM inboxes WHERE id=3087`); err != nil {
		t.Fatal(err)
	}
	app.internalConversation.ContactEmail = " team-tickets@example.com "
	for _, tc := range []struct {
		name      string
		inbox     int
		agent     int
		team      int
		email     string
		firstName string
	}{
		{"agent", 0, 102, 0, "other@example.com", "Other"},
		{"team", 0, 0, 201, "team-tickets@example.com", "Internal"},
		{"team and agent", 0, 102, 201, "other@example.com", "Other"},
		{"existing client with actual ID", 3087, 0, 201, "team-tickets@example.com", "Internal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var input map[string]any
			json.Unmarshal([]byte(internalCreateBody(tc.inbox, tc.agent, tc.team)), &input)
			// Stale external-mode data and a forged contact name cannot select or
			// overwrite an external customer instead of the internal recipient.
			input["contact_email"], input["first_name"], input["external_user_id"] = "forged@example.com", "Forged", "external-customer"
			body, _ := json.Marshal(input)
			raw := systemAttributesRequest(t, app, 101, "POST", "", string(body), perm(handleCreateConversation, "conversations:write"), 200)
			var response struct {
				Data cmodels.Conversation `json:"data"`
			}
			if err := json.Unmarshal(raw, &response); err != nil {
				t.Fatal(err)
			}
			conv, err := app.conversation.GetConversation(0, response.Data.UUID, "")
			if err != nil {
				t.Fatal(err)
			}
			if conv.InboxID != 3087 || conv.Contact.Email.String != tc.email || conv.Contact.FirstName != tc.firstName {
				t.Fatalf("wrong routing/contact: inbox=%d email=%s name=%s", conv.InboxID, conv.Contact.Email.String, conv.Contact.FirstName)
			}
			if conv.AssignedTeamID.Int != tc.team || conv.AssignedUserID.Int != tc.agent {
				t.Fatalf("assignment changed: team=%v agent=%v", conv.AssignedTeamID, conv.AssignedUserID)
			}
			var attrs map[string]any
			if err := json.Unmarshal(conv.CustomAttributes, &attrs); err != nil {
				t.Fatal(err)
			}
			if attrs["private"] != true || attrs["internal"] != true || attrs["creator_id"] != float64(101) || attrs["access_mode"] != nil || attrs["owner_user_id"] != nil {
				t.Fatalf("security initialization changed: %s", conv.CustomAttributes)
			}
			var to []byte
			if err := db.Get(&to, `SELECT m.meta->'to' FROM conversation_messages m JOIN conversations c ON c.id=m.conversation_id WHERE c.uuid=$1 AND m.type='outgoing'`, conv.UUID); err != nil {
				t.Fatal(err)
			}
			var recipients []string
			if err := json.Unmarshal(to, &recipients); err != nil || len(recipients) != 1 || recipients[0] != tc.email {
				t.Fatalf("email recipient contract changed: %s %v", to, err)
			}
			for id, want := range map[int]int{101: 200, 102: 200, 103: 403, 104: 200} {
				systemAttributesRequest(t, app, id, "GET", conv.UUID, "", perm(handleGetConversation, "conversations:read"), want)
			}
		})
	}
	var untouched string
	if err := db.Get(&untouched, `SELECT first_name FROM users WHERE id=105`); err != nil || untouched != "Contact" {
		t.Fatalf("external contact changed: %s %v", untouched, err)
	}
}

func TestInternalConversationConfigurationRejectsHTTP(t *testing.T) {
	for index, tc := range []struct {
		name, config, sql  string
		inbox, agent, team int
		actor, status      int
		message            string
	}{
		{name: "missing configuration", config: "missing", status: 400, message: "conversation.internal_inbox_uuid"},
		{name: "invalid UUID", config: "invalid", status: 400, message: "UUID"},
		{name: "nonexistent inbox", config: "nonexistent", status: 400, message: "does not exist"},
		{name: "deleted inbox", sql: `UPDATE inboxes SET deleted_at=now() WHERE id=102`, status: 400, message: "does not exist"},
		{name: "disabled inbox", sql: `UPDATE inboxes SET enabled=false WHERE id=102`, status: 400, message: "disabled"},
		{name: "wrong channel", sql: `UPDATE inboxes SET channel='livechat' WHERE id=102`, status: 400, message: "public email inbox"},
		{name: "personal inbox", config: "personal", status: 400, message: "public email inbox"},
		{name: "conflicting ID 3", inbox: 3, sql: `INSERT INTO inboxes(id,name,channel) VALUES(3,'Unrelated','email')`, status: 400, message: "configured inbox"},
		{name: "negative inbox ID", inbox: -1, status: 400, message: "configured inbox"},
		{name: "missing team contact", config: "no contact", team: 201, status: 400, message: "conversation.internal_contact_email"},
		{name: "invalid team contact", config: "invalid contact", team: 201, status: 400, message: "email"},
		{name: "no create permission", actor: 102, agent: 102, status: 403},
		{name: "no agent assignment permission", agent: 102, sql: `UPDATE roles SET permissions=array_remove(permissions,'conversations:update_user_assignee') WHERE name='Security creator'`, status: 403},
		{name: "no team assignment permission", team: 201, sql: `UPDATE roles SET permissions=array_remove(permissions,'conversations:update_team_assignee') WHERE name='Security creator'`, status: 403},
		{name: "disabled recipient", agent: 102, sql: `UPDATE users SET enabled=false WHERE id=102`, status: 400},
		{name: "unknown agent", agent: 999999, status: 404},
		{name: "unknown team", team: 999999, status: 400},
		{name: "no inboxes after installation", config: "missing", sql: `DELETE FROM inboxes`, status: 400, message: "conversation.internal_inbox_uuid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app, db := systemAttributesHTTPFixture(t, fmt.Sprintf("f02_reject_%d", index))
			switch tc.config {
			case "missing":
				app.internalConversation.InboxUUID = ""
			case "invalid":
				app.internalConversation.InboxUUID = "3"
			case "nonexistent":
				app.internalConversation.InboxUUID = "135434a1-ad1f-4213-8770-1a51ea910505"
			case "personal":
				if err := db.Get(&app.internalConversation.InboxUUID, `SELECT uuid FROM inboxes WHERE id=101`); err != nil {
					t.Fatal(err)
				}
			case "no contact":
				app.internalConversation.ContactEmail = ""
			case "invalid contact":
				app.internalConversation.ContactEmail = "invalid"
			}
			if tc.sql != "" {
				db.MustExec(tc.sql)
			}
			actor := tc.actor
			if actor == 0 {
				actor = 101
			}
			raw := systemAttributesRequest(t, app, actor, "POST", "", internalCreateBody(tc.inbox, tc.agent, tc.team), perm(handleCreateConversation, "conversations:write"), tc.status)
			if tc.message != "" && !strings.Contains(strings.ToLower(string(raw)), strings.ToLower(tc.message)) {
				t.Fatalf("missing setup error %q: %s", tc.message, raw)
			}
			for _, table := range []string{"conversations", "conversation_messages"} {
				var count int
				if err := db.Get(&count, `SELECT count(*) FROM `+table); err != nil || count != 0 {
					t.Fatalf("rejected request wrote %s: %d %v", table, count, err)
				}
			}
		})
	}
}

func TestInternalConversationConfigurationDoesNotAffectOtherCreation(t *testing.T) {
	app, _ := systemAttributesHTTPFixture(t, "f02_optional_config")
	configuredUUID := app.internalConversation.InboxUUID
	app.internalConversation = internalConversationConfig{}
	for _, tc := range []struct {
		name, attrs string
		inbox       int
	}{
		{"public", `{}`, 102},
		{"personal", `{}`, 101},
		{"private non-internal", `{"private":true}`, 102},
	} {
		t.Run(tc.name, func(t *testing.T) {
			systemAttributesRequest(t, app, 101, "POST", "", systemAttributesCreateBody(tc.attrs, tc.inbox, 0, 0), perm(handleCreateConversation, "conversations:write"), 200)
		})
	}
	// A valid employee recipient does not need the team contact setting.
	app.internalConversation.InboxUUID = configuredUUID
	systemAttributesRequest(t, app, 101, "POST", "", internalCreateBody(0, 102, 0), perm(handleCreateConversation, "conversations:write"), 200)
}
