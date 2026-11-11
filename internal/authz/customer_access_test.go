package authz

import (
	"encoding/json"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"testing"
)

func TestCustomerAccessExplicitReasonsOnly(t *testing.T) {
	for _, tc := range []struct {
		name           string
		assigned, team int
		canonical      []int64
		attrs          string
		perms          []string
		role           string
		want           bool
	}{
		{name: "read is not global", attrs: `{"customer_visibility":true}`, perms: []string{"conversations:read"}},
		{name: "read all cannot bypass explicit customer ACL", attrs: `{"customer_visibility":true}`, perms: []string{"conversations:read", "conversations:read_all"}},
		{name: "returned unassigned ticket is not public", attrs: `{"customer_visibility":true}`, perms: []string{"conversations:read", "conversations:read_unassigned"}},
		{name: "marker protects missing visibility attribute", attrs: `{}`, perms: []string{"conversations:read", "conversations:read_all"}},
		{name: "current individual", assigned: 1, perms: []string{"conversations:read", "conversations:read_assigned"}, want: true},
		{name: "additional individual", assigned: 2, canonical: []int64{2, 1}, perms: []string{"conversations:read", "conversations:read_assigned"}, want: true},
		{name: "current team", team: 10, perms: []string{"conversations:read", "conversations:read_team_all"}, want: true},
		{name: "team inbox unassigned", team: 10, perms: []string{"conversations:read", "conversations:read_team_inbox"}, want: true},
		{name: "team inbox canonical assignee present", team: 10, canonical: []int64{2}, perms: []string{"conversations:read", "conversations:read_team_inbox"}},
		{name: "explicit visibility", attrs: `{"visible_users":[1]}`, want: true},
		{name: "creator", attrs: `{"creator_id":1}`, want: true},
		{name: "manager", attrs: `{"visibility_managers":[1]}`, want: true},
		{name: "support", role: "Kundensupport", want: true},
		{name: "admin", role: "Admin", want: true},
		{name: "support cannot bypass personal", attrs: `{"access_mode":"personal","visible_users":[]}`, role: "Kundensupport"},
		{name: "support cannot bypass internal", attrs: `{"private":true,"visible_users":[]}`, role: "Kundensupport"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := user(1, tc.perms, 10)
			if tc.role != "" {
				u.Roles = []string{tc.role}
			}
			c := conv(tc.assigned, tc.team)
			c.CustomerTicket = true
			c.AssignedUserIDs = tc.canonical
			if tc.attrs != "" {
				c.CustomAttributes = json.RawMessage(tc.attrs)
			}
			if got := CanReadConversation(u, c); got != tc.want {
				t.Fatalf("got %v want %v: %+v", got, tc.want, c)
			}
		})
	}
	// Public and internal authorization keep the established broad/creator rules.
	public := cmodels.Conversation{}
	if !CanReadConversation(user(1, []string{"conversations:read", "conversations:read_all"}), public) {
		t.Fatal("ordinary public read_all changed")
	}
}
