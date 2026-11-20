package conversation

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	azmodels "github.com/abhinavxd/libredesk/internal/authz/models"
	amodels "github.com/abhinavxd/libredesk/internal/automation/models"
	"github.com/abhinavxd/libredesk/internal/tag"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
)

func TestSystemTagMembershipAuthority(t *testing.T) {
	m := emailThreadFixture(t, "tag_membership_authority")
	m.db.MustExec(`DELETE FROM tags; INSERT INTO users(id,type,email,first_name) VALUES(771,'agent','bot@example.test','Bot')`)
	if err := tag.EnsureSystemTags(t.Context(), m.db); err != nil {
		t.Fatal(err)
	}
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(42,'Ordinary')`)
	hooks := &bootstrapWebhooks{}
	m.webhookStore = hooks
	actor := umodels.User{ID: 101, FirstName: "Owner"}
	bot := umodels.User{ID: 771, Enabled: true, Type: umodels.UserTypeAgent, Permissions: []string{azmodels.PermConversationsAddSystemTags}}
	for _, name := range []string{"🏢Intern", "🦽Kundenticket", "😎Mensch erforderlich", "🧷Service-Mail"} {
		t.Run(name, func(t *testing.T) {
			var uuid string
			if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`); err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{amodels.ActionAddTags, amodels.ActionSetTags} {
				for _, denied := range []umodels.User{actor, {ID: 41}, {ID: 104, Roles: []string{"Admin"}}, bot} {
					if err := m.SetConversationTags(uuid, action, []string{name}, denied); err == nil {
						t.Fatalf("ordinary authority bypass: %d %s", denied.ID, action)
					}
				}
			}
			for _, invalid := range []umodels.User{{ID: 41}, {ID: 104, Enabled: true, Type: "agent", Roles: []string{"Admin"}}, {ID: 771, Type: "agent", Permissions: bot.Permissions}, {ID: 771, Enabled: true, Type: "contact", Permissions: bot.Permissions}} {
				if err := m.SetIntegrationConversationTags(uuid, amodels.ActionAddTags, []string{name}, invalid); err == nil {
					t.Fatalf("invalid integration authority accepted: %+v", invalid)
				}
			}
			if err := m.SetIntegrationConversationTags(uuid, amodels.ActionSetTags, []string{name}, bot); err != nil {
				t.Fatal(err)
			}
			for _, action := range []string{amodels.ActionRemoveTags, amodels.ActionSetTags} {
				input := []string{name}
				if action == amodels.ActionSetTags {
					input = []string{}
				}
				if err := m.SetConversationTags(uuid, action, input, actor); err == nil {
					t.Fatal("classification removed")
				}
				if err := m.SetIntegrationConversationTags(uuid, action, input, bot); err == nil {
					t.Fatal("integration removed classification")
				}
			}
			if err := m.SetConversationTags(uuid, amodels.ActionSetTags, []string{name, "Ordinary"}, actor); err != nil {
				t.Fatal("ordinary edits preserving classification failed", err)
			}
			if err := m.SetConversationTags(uuid, amodels.ActionRemoveTags, []string{"Ordinary"}, actor); err != nil {
				t.Fatal(err)
			}
			before := len(hooks.events)
			if err := m.AddSystemTags(uuid, []string{name}); err != nil {
				t.Fatal(err)
			}
			if before != len(hooks.events) {
				t.Fatal("internal system classification emitted webhook")
			}
		})
	}
	if !slices.Contains(hooks.events, wmodels.EventConversationTagsChanged) {
		t.Fatal("normal/integration tag webhook suppressed unexpectedly")
	}
}

func TestSystemTagMembershipLegacyIDsAndConcurrentChanges(t *testing.T) {
	m := emailThreadFixture(t, "tag_legacy_and_concurrent")
	// The legacy fixture has customer under ID 13: protect that name too.
	m.db.MustExec(`INSERT INTO tags(id,name) VALUES(8,'Unexpected definition'),(42,'Rekla')`)
	var uuid string
	if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	actor := umodels.User{ID: 101}
	for _, name := range []string{"Unexpected definition", "🦽Kundenticket"} {
		if err := m.SetConversationTags(uuid, amodels.ActionSetTags, []string{name}, actor); err == nil {
			t.Fatalf("legacy ID/name bypass: %q", name)
		}
	}
	// No webhook/activity writes are needed for this concurrent row-lock test.
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for index := range 12 {
		wg.Go(func() {
			if index%2 == 0 {
				errs <- m.AddSystemTags(uuid, []string{"🦽Kundenticket"})
			} else {
				_, _, _, err := m.mutateTagsWithSpam(uuid, amodels.ActionAddTags, []string{"Rekla"}, actor)
				errs <- err
			}
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	tags, err := m.getConversationTags(uuid)
	if err != nil || len(tags) != 2 || !slices.Contains(tags, "🦽Kundenticket") || !slices.Contains(tags, "Rekla") {
		t.Fatal(fmt.Sprintf("parallel classification lost: %v %v", tags, err))
	}
}
