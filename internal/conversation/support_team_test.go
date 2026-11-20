package conversation

import (
	"fmt"
	"testing"

	"github.com/abhinavxd/libredesk/internal/authz"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
)

func TestCustomerSupportOptionalTeam(t *testing.T) {
	for i, setup := range []string{
		`DELETE FROM teams WHERE id=201`,
		`UPDATE teams SET name='Renamed support' WHERE id=201`,
		`UPDATE teams SET id=761 WHERE id=201`,
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			m := routingFixture(t, fmt.Sprintf("support_team_optional_%d", i))
			m.db.MustExec(setup)
			actor := umodels.User{ID: 103}
			uuid := routingTicket(t, m, "Closed", true)
			if err := m.UpdateConversationTeamAssignee(uuid, 202, actor); err != nil {
				t.Fatal(err)
			}
			if err := m.UpdateConversationUserAssignee(uuid, 106, actor); err != nil {
				t.Fatal(err)
			}
			c := routingAssert(t, m, uuid, 202, []int64{106}, "Closed", true)
			// Multiple memberships, including support, never override explicit external responsibility.
			if !c.CustomerRedistributed {
				t.Fatal("external context lost")
			}
			if err := m.ReturnCustomerToSupport(uuid, actor); err != nil {
				t.Fatal(err)
			}
			c = routingAssert(t, m, uuid, 0, nil, "Open", false)
			if authz.CanReadConversation(umodels.User{ID: 107, Permissions: []string{"conversations:read", "conversations:read_all"}}, c) {
				t.Fatal("fallback grants outsider access")
			}
			if i == 2 {
				if err := m.UpdateConversationTeamAssignee(uuid, 761, actor); err != nil {
					t.Fatal(err)
				}
				routingAssert(t, m, uuid, 761, nil, "Open", false)
			}
			if i == 1 {
				if err := m.UpdateConversationTeamAssignee(uuid, 201, actor); err != nil {
					t.Fatal(err)
				}
				routingAssert(t, m, uuid, 201, nil, "Open", true)
			}
		})
	}
}

func TestCustomerSupportAmbiguousTeamRejected(t *testing.T) {
	m := routingFixture(t, "support_team_ambiguous")
	// Supported schemas reject duplicate names. Simulate an inconsistent legacy
	// database to verify routing also fails closed instead of choosing min(id).
	if _, err := m.db.Exec(`INSERT INTO teams(name,conversation_assignment_type) VALUES('Kundensupport','Manual')`); err == nil {
		t.Fatal("schema allowed duplicate")
	}
	m.db.MustExec(`ALTER TABLE teams DROP CONSTRAINT constraint_teams_on_name_unique; INSERT INTO teams(id,name,conversation_assignment_type) VALUES(761,'Kundensupport','Manual')`)
	uuid := routingTicket(t, m, "Closed", true)
	m.db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, uuid)
	var before string
	if err := m.db.Get(&before, `SELECT to_jsonb(c)::text FROM conversations c WHERE uuid=$1`, uuid); err != nil {
		t.Fatal(err)
	}
	actor := umodels.User{ID: 103}
	for _, change := range []func() error{
		func() error { return m.UpdateConversationTeamAssignee(uuid, 201, actor) },
		func() error { return m.UpdateConversationUserAssignee(uuid, 106, actor) },
		func() error { return m.ReturnCustomerToSupport(uuid, actor) },
	} {
		if err := change(); err == nil {
			t.Fatal("ambiguous support context accepted")
		}
	}
	var after string
	if err := m.db.Get(&after, `SELECT to_jsonb(c)::text FROM conversations c WHERE uuid=$1`, uuid); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatal("rejected assignment changed conversation")
	}
}
