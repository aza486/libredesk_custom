package messagetemplate

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func code(err error) int {
	if e, ok := err.(envelope.Error); ok {
		return e.Code
	}
	return 0
}
func TestValidationAndHTMLSafety(t *testing.T) {
	for _, in := range []Input{
		{Name: " ", Content: "text", Scope: "personal"},
		{Name: strings.Repeat("x", 161), Content: "text", Scope: "personal"},
		{Name: "name\n", Content: "text", Scope: "unknown"},
		{Name: "name", Content: "<p><br></p>", Scope: "personal"},
		{Name: "name", Content: "<script>alert(1)</script>", Scope: "personal"},
		{Name: "name", Content: "<p>&nbsp;\u200b</p>", Scope: "personal"},
		{Name: "name", Content: strings.Repeat("x", MaxContentBytes+1), Scope: "personal"},
	} {
		if _, err := Validate(in); code(err) != 400 {
			t.Errorf("invalid input accepted: %.20q: %v", in.Content, err)
		}
	}
	safe, err := Validate(Input{Name: " Geschäftskundenkonditionen ", Scope: "personal", Content: `<p>Sehr geehrte/r [Name],<br>Danke.</p><ul><li><strong>Vorteile</strong> <em>Kursiv</em> <u>Unterstrichen</u></li></ul><ol start="2"><li>Nummeriert</li></ol><p><a href="https://www.pixelfoto-express.de/scan-und-film/filmentwicklung" onclick="evil()">Preise</a><a href="javascript:evil()">Bad</a><img src="https://example.com/x" onerror="evil()"><script>evil()</script><iframe src="https://evil.test"></iframe>{{ inert }}</p>`})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[Name]", "<br>", "<strong>", "<em>", "<u>", "<ul>", "<ol", `start="2"`, `href="https://www.pixelfoto-express.de/scan-und-film/filmentwicklung"`, "{{ inert }}"} {
		if !strings.Contains(safe.Content, want) {
			t.Errorf("lost %s in %s", want, safe.Content)
		}
	}
	for _, bad := range []string{"onclick", "onerror", "javascript:", "<script", "<iframe", "<img"} {
		if strings.Contains(safe.Content, bad) {
			t.Errorf("unsafe %s", safe.Content)
		}
	}
}
func TestTemplateLifecycleIsolationAndRevocation(t *testing.T) {
	db := testutil.NewPersonalDB(t, "message_templates")
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Writer',ARRAY['messages:write','conversations:read','conversations:read_all']),('Kundensupport',ARRAY['messages:write','conversations:read']);
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN(101,102,103,104) AND r.name='Writer';
 INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN(102,103) AND r.name='Kundensupport';
 INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(101,102);`)
	if err := migrations.V2_25_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := migrations.V2_25_0(db, nil, nil); err != nil {
		t.Fatal("migration repetition", err)
	}
	var public, private string
	db.Get(&public, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject) VALUES(105,102,101,'Public') RETURNING uuid`)
	db.Get(&private, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject) VALUES(105,101,101,'Private') RETURNING uuid`)
	m := New(db)
	owner := Access{101, private}
	other := Access{102, private}
	support := Access{103, public}
	support2 := Access{102, public}
	create := func(a Access, name, scope string) Template {
		t.Helper()
		v, err := m.Create(a, Input{Name: name, Content: "<p>Original <strong>Text</strong></p>", Scope: scope})
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	personal := create(owner, "Persönlich", "personal")
	if personal.OwnerID.Int != 101 || personal.CreatedBy.Int != 101 {
		t.Fatal(personal)
	}
	create(other, "Andere persönliche Vorlage", "personal")
	for _, a := range []Access{other, support, {104, public}} {
		if _, err := m.Get(a, personal.UUID); code(err) != 404 {
			t.Fatalf("foreign personal get: %v", err)
		}
		if _, err := m.Update(a, personal.UUID, Input{Name: "Stolen", Content: "Text", Scope: "personal", Revision: 1}); code(err) != 404 {
			t.Fatalf("foreign personal update: %v", err)
		}
		if err := m.Delete(a, personal.UUID, 1); code(err) != 404 {
			t.Fatalf("foreign personal delete: %v", err)
		}
	}
	for _, a := range []Access{owner, other, {101, public}, {103, private}, {103, ""}} {
		result, err := m.List(a)
		if err != nil {
			t.Fatal(err)
		}
		if result.CanSupport {
			t.Fatalf("global scope exposed: %+v", a)
		}
		if _, err := m.Create(a, Input{Name: "Forged", Content: "text", Scope: "customer_support"}); code(err) != 403 {
			t.Fatalf("global create allowed: %+v %v", a, err)
		}
	}
	if _, err := m.Update(owner, personal.UUID, Input{Name: "Convert", Content: "text", Scope: "customer_support", Revision: 1}); code(err) != 400 {
		t.Fatal("scope conversion", err)
	}
	global := create(support, "Gelangensbestätigung", "customer_support")
	if global.OwnerID.Valid || global.CreatedBy.Int != 103 {
		t.Fatal(global)
	}
	if _, err := m.Get(support2, global.UUID); err != nil {
		t.Fatal("shared read", err)
	}
	for _, a := range []Access{owner, other, {101, public}, {103, private}, {103, ""}} {
		if _, err := m.Get(a, global.UUID); code(err) != 404 {
			t.Fatalf("global UUID leaked: %+v %v", a, err)
		}
		if err := m.Delete(a, global.UUID, 1); code(err) != 404 {
			t.Fatal("global delete", err)
		}
	}
	// Two employees editing exactly the same revision: one succeeds, one conflicts.
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, a := range []Access{support, support2} {
		wg.Add(1)
		go func(a Access) {
			defer wg.Done()
			_, err := m.Update(a, global.UUID, Input{Name: "Updated", Content: "<p>Edited</p>", Scope: "customer_support", Revision: 1})
			results <- err
		}(a)
	}
	wg.Wait()
	close(results)
	ok, conflicts := 0, 0
	for err := range results {
		if err == nil {
			ok++
		} else if code(err) == 409 {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflicts != 1 {
		t.Fatal("lost update", ok, conflicts)
	}
	if err := m.Delete(support, global.UUID, 1); code(err) != 409 {
		t.Fatal("stale delete", err)
	}
	db.MustExec(`DELETE FROM user_roles WHERE user_id=103 AND role_id=(SELECT id FROM roles WHERE name='Kundensupport')`)
	if _, err := m.Get(support, global.UUID); code(err) != 404 {
		t.Fatal("revoked role", err)
	}
	db.MustExec(`UPDATE roles SET permissions=array_remove(permissions,'message_templates:customer_support') WHERE name='Kundensupport'`)
	if _, err := m.Get(support2, global.UUID); code(err) != 404 {
		t.Fatal("revoked capability", err)
	}
	db.MustExec(`UPDATE roles SET permissions=array_append(permissions,'message_templates:customer_support') WHERE name='Kundensupport'; UPDATE inboxes SET enabled=false WHERE id=102`)
	if _, err := m.Get(support2, global.UUID); code(err) != 404 {
		t.Fatal("disabled inbox", err)
	}
	db.MustExec(`UPDATE inboxes SET enabled=true WHERE id=102`)
	if err := m.Delete(support2, global.UUID, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Get(support2, global.UUID); code(err) != 404 {
		t.Fatal("deleted still visible", err)
	}
	updated, err := m.Update(owner, personal.UUID, Input{Name: "Neu", Content: "<p>Persönlich</p>", Scope: "personal", Revision: 1})
	if err != nil || updated.Revision != 2 {
		t.Fatal(updated, err)
	}
	if err := m.Delete(owner, personal.UUID, 2); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`UPDATE roles SET permissions=array_remove(permissions,'messages:write') WHERE name='Writer'; UPDATE roles SET permissions=array_remove(permissions,'messages:write') WHERE name='Kundensupport'`)
	if _, err := m.List(other); code(err) != 403 {
		t.Fatal("revoked messages permission", err)
	}
	db.MustExec(`UPDATE users SET enabled=false WHERE id=104`)
	if _, err := m.Create(Access{104, ""}, Input{Name: "Disabled", Content: "text", Scope: "personal"}); code(err) != 403 {
		t.Fatal("disabled actor", err)
	}
}

func TestDevelopmentExamples(t *testing.T) {
	raw, err := os.ReadFile("testdata/examples.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples []Input
	if err := json.Unmarshal(raw, &examples); err != nil {
		t.Fatal(err)
	}
	for _, in := range examples {
		in.Scope = "personal"
		result, err := Validate(in)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Content) < 500 {
			t.Fatal("long example truncated")
		}
		if in.Name == "Geschäftskundenkonditionen" && (strings.Count(result.Content, "<strong>") != 7 || !strings.Contains(result.Content, "[Name]")) {
			t.Fatal("lost formatting or placeholder")
		}
	}
}
