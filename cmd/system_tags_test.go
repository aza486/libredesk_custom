package main

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/tag"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/knadh/stuffbin"
)

func TestSystemTagInstallUpgradeLifecycle(t *testing.T) {
	db := testutil.NewDB(t, "system_tag_install_upgrade")
	// Force the real installer down its fresh-schema path in this disposable DB.
	db.MustExec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`)
	schema, err := filepath.Abs("../schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	fs, err := stuffbin.NewLocalFS("/", schema+":/schema.sql")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LIBREDESK_SYSTEM_USER_PASSWORD", "Test-only-install-39!Password")
	if err := install(t.Context(), db, fs, true, false); err != nil {
		t.Fatal(err)
	}
	if err := tag.CheckSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	upgrade(db, fs, false)
	last, pending, err := getPendingMigrations(db)
	if err != nil || len(pending) != 0 || last != migList[len(migList)-1].version {
		t.Fatalf("incomplete upgrade: %q %d %v", last, len(pending), err)
	}
	// Existing schema is skipped, but missing required tags must still be seeded.
	db.MustExec(`DELETE FROM tags WHERE id=12`)
	if err := install(t.Context(), db, fs, true, false); err != nil {
		t.Fatal(err)
	}
	if err := tag.CheckSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	// --upgrade must also seed when there are no pending migrations.
	db.MustExec(`DELETE FROM tags WHERE id=11`)
	upgrade(db, fs, false)
	if err := tag.CheckSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var inboxes int
	if err := db.Get(&inboxes, `SELECT count(*) FROM inboxes`); err != nil || inboxes != 0 {
		t.Fatalf("installation unexpectedly requires/creates an inbox: %d %v", inboxes, err)
	}
}

func TestSystemTagBootstrapInternalCreationHTTP(t *testing.T) {
	app, db := systemAttributesHTTPFixture(t, "system_tag_bootstrap_internal")
	// This legacy fixture normally seeds customer tag 13 and internal tag 8.
	// Here the application bootstrap, rather than the fixture, supplies all tags.
	db.MustExec(`DELETE FROM tags`)
	if err := tag.EnsureSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	raw := systemAttributesRequest(t, app, 101, "POST", "", internalCreateBody(0, 102, 201), perm(handleCreateConversation, "conversations:write"), 200)
	var response struct {
		Data cmodels.Conversation `json:"data"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	var tags []string
	if err := db.Select(&tags, `SELECT t.name FROM tags t JOIN conversation_tags ct ON ct.tag_id=t.id JOIN conversations c ON c.id=ct.conversation_id WHERE c.uuid=$1`, response.Data.UUID); err != nil || !slices.Contains(tags, "🏢Intern") {
		t.Fatalf("missing initialized internal tag: %v %v", tags, err)
	}
	for actor, want := range map[int]int{101: 200, 102: 200, 103: 403, 104: 200} {
		systemAttributesRequest(t, app, actor, "GET", response.Data.UUID, "", perm(handleGetConversation, "conversations:read"), want)
	}
}
