package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/importer"
	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/tag"
	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type tagSecurityHTTP struct {
	app    *App
	db     *sqlx.DB
	client *http.Client
	keys   map[int][2]string
}

func newTagSecurityHTTP(t *testing.T, name string) *tagSecurityHTTP {
	t.Helper()
	app, db := systemAttributesHTTPFixture(t, name)
	db.MustExec(`DELETE FROM tags;
 INSERT INTO users(id,type,email,first_name,last_name) VALUES(41,'agent','regular41@example.test','Regular',''),(771,'agent','integration@example.test','Integration','');
 INSERT INTO roles(name,permissions) VALUES('Tag operator',ARRAY['conversations:read','conversations:read_all','conversations:update_tags','messages:write','tags:manage']);
 INSERT INTO user_roles(user_id,role_id) SELECT u,r.id FROM roles r CROSS JOIN unnest(ARRAY[41,101,104,771]) u WHERE r.name='Tag operator';
 INSERT INTO roles(name,permissions) VALUES('Explicit classifier',ARRAY['conversations:add_system_tags']);`)
	if err := tag.EnsureSystemTags(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var err error
	app.tag, err = tag.New(tag.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.setting, err = setting.New(setting.Opts{DB: db, Lo: app.lo})
	if err != nil {
		t.Fatal(err)
	}
	app.role, err = role.New(role.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	app.importer = importer.New(importer.Opts{Lo: app.lo, I18n: app.i18n})
	t.Cleanup(app.importer.Close)
	g := fastglue.NewGlue()
	g.SetContext(app)
	g.POST("/api/v1/conversations/{uuid}/tags", perm(handleUpdateConversationtags, "conversations:update_tags"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	g.POST("/api/v1/tags", perm(handleCreateTag, "tags:manage"))
	g.PUT("/api/v1/tags/{id}", perm(handleUpdateTag, "tags:manage"))
	g.DELETE("/api/v1/tags/{id}", perm(handleDeleteTag, "tags:manage"))
	g.POST("/api/v1/tags/import", perm(handleImportTags, "tags:manage"))
	g.POST("/api/v1/roles", perm(handleCreateRole, "roles:manage"))
	g.POST("/api/v1/conversations/{cuuid}/messages", perm(handleSendMessage, "messages:write"))
	g.POST("/api/v1/conversations/{cuuid}/ai-analysis-notes", perm(handleCreateAIAnalysisNote, "messages:write"))
	dir, err := os.MkdirTemp("/tmp", "ld-tags-http-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	socket := filepath.Join(dir, "http.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &fasthttp.Server{Handler: g.Handler()}
	go func() { _ = server.Serve(listener) }()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}
	t.Cleanup(func() { transport.CloseIdleConnections(); _ = server.Shutdown(); _ = listener.Close() })
	f := &tagSecurityHTTP{app: app, db: db, client: &http.Client{Transport: transport, Timeout: 10 * time.Second}, keys: map[int][2]string{}}
	for _, id := range []int{41, 101, 104, 771} {
		key, secret, err := app.user.GenerateAPIKey(id)
		if err != nil {
			t.Fatal(err)
		}
		f.keys[id] = [2]string{key, secret}
	}
	return f
}

func (f *tagSecurityHTTP) request(t *testing.T, actor int, method, path, body string, want int) []byte {
	t.Helper()
	req, err := http.NewRequest(method, "http://test"+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key, ok := f.keys[actor]; ok {
		req.SetBasicAuth(key[0], key[1])
	}
	response, err := f.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != want {
		t.Fatalf("%s %s as %d: %d want %d: %s %v", method, path, actor, response.StatusCode, want, raw, err)
	}
	return raw
}

func (f *tagSecurityHTTP) ticket(t *testing.T, attributes string) string {
	t.Helper()
	var uuid string
	if err := f.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,102,101,$1) RETURNING uuid`, attributes); err != nil {
		t.Fatal(err)
	}
	return uuid
}

func TestSystemTagSecurityHTTPAssignments(t *testing.T) {
	f := newTagSecurityHTTP(t, "tag_security_http")
	f.app.aiNoteAgentIDs = []int{771} // This allowlist must never grant tag rights.
	uuid := f.ticket(t, `{}`)
	path := "/api/v1/conversations/" + uuid + "/tags"
	for _, name := range []string{"🏢Intern", "🦽Kundenticket", "😎Mensch erforderlich", "🧷Service-Mail"} {
		t.Run(name, func(t *testing.T) {
			for _, action := range []string{"add_tags", "set_tags"} {
				for _, actor := range []int{41, 104, 771} {
					payload, _ := json.Marshal(map[string]any{"tags": []string{name}, "action": action, "actor_id": 771, "user_id": 771, "internal": true, "system": true})
					f.request(t, actor, "POST", path+"?actor_id=771&internal=true", string(payload), 400)
				}
			}
		})
	}
	f.request(t, 0, "POST", path, `{"tags":["🏢Intern"],"action":"add_tags"}`, 403)
	f.db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT 771,id FROM roles WHERE name='Explicit classifier'`)
	f.app.aiNoteAgentIDs = nil
	f.request(t, 771, "POST", path, `{"tags":["🏢Intern"],"action":"add_tags"}`, 200)
	f.request(t, 771, "POST", path, `{"tags":["🏢Intern","😎Mensch erforderlich"],"action":"set_tags"}`, 200)
	for _, actor := range []int{41, 104, 771} {
		f.request(t, actor, "POST", path, `{"tags":[],"action":"set_tags"}`, 400)
		f.request(t, actor, "POST", path, `{"tags":["🏢Intern"],"action":"remove_tags"}`, 400)
	}
	f.request(t, 41, "POST", "/api/v1/tags", `{"id":8,"name":"Ordinary"}`, 200)
	f.request(t, 41, "POST", path, `{"tags":["🏢Intern","😎Mensch erforderlich","Ordinary"],"action":"set_tags"}`, 200)
	f.request(t, 41, "POST", path, `{"tags":["Ordinary"],"action":"remove_tags"}`, 200)
	f.request(t, 41, "POST", path, `{"tags":["🗑Spam"],"action":"add_tags"}`, 200)
	f.request(t, 41, "POST", path, `{"tags":["🗑Spam"],"action":"remove_tags"}`, 200)
	f.db.MustExec(`DELETE FROM user_roles WHERE user_id=771 AND role_id=(SELECT id FROM roles WHERE name='Explicit classifier')`)
	f.request(t, 771, "POST", path, `{"tags":["🧷Service-Mail"],"action":"add_tags"}`, 400)
	f.db.MustExec(`UPDATE users SET enabled=false WHERE id=771`)
	f.request(t, 771, "POST", path, `{"tags":["🧷Service-Mail"],"action":"add_tags"}`, 401)
}

func TestSystemTagSecurityHTTPDefinitionsAndImport(t *testing.T) {
	f := newTagSecurityHTTP(t, "tag_definitions_http")
	for _, tag := range []struct {
		id   int
		name string
	}{{8, "🏢Intern"}, {9, "🦽Kundenticket"}, {11, "😎Mensch erforderlich"}, {12, "🗑Spam"}, {14, "🧷Service-Mail"}} {
		path := fmt.Sprintf("/api/v1/tags/%d", tag.id)
		for _, actor := range []int{41, 104} {
			f.request(t, actor, "POST", "/api/v1/tags", fmt.Sprintf(`{"name":%q}`, tag.name), 400)
			f.request(t, actor, "PUT", path, `{"name":"Forged"}`, 400)
			f.request(t, actor, "DELETE", path, "", 400)
		}
	}
	f.request(t, 41, "POST", "/api/v1/tags", `{"name":"Editable"}`, 200)
	var id int
	if err := f.db.Get(&id, `SELECT id FROM tags WHERE name='Editable'`); err != nil {
		t.Fatal(err)
	}
	f.request(t, 104, "PUT", fmt.Sprintf("/api/v1/tags/%d", id), `{"name":"🗑️Spam "}`, 400)
	f.request(t, 41, "PUT", fmt.Sprintf("/api/v1/tags/%d", id), `{"name":"Renamed"}`, 200)
	f.request(t, 41, "DELETE", fmt.Sprintf("/api/v1/tags/%d", id), "", 200)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "tags.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(part, "name\n🏢Intern\n🗑️Spam\n🧷service-mail\nImported ordinary\n")
	_ = writer.Close()
	req, _ := http.NewRequest("POST", "http://test/api/v1/tags/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	key := f.keys[104]
	req.SetBasicAuth(key[0], key[1])
	response, err := f.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("import status %d", response.StatusCode)
	}
	// Observe completion through the database, without reading the importer's
	// mutable job pointer while its worker is still updating counters.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var count int
		if err := f.db.Get(&count, `SELECT count(*) FROM tags WHERE name='Imported ordinary'`); err != nil {
			t.Fatal(err)
		}
		if count == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("ordinary import did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
	var count int
	if err := f.db.Get(&count, `SELECT count(*) FROM tags`); err != nil || count != 6 {
		t.Fatalf("reserved import names were created: %d %v", count, err)
	}
	if err := tag.CheckSystemTags(t.Context(), f.db); err != nil {
		t.Fatal(err)
	}
}

func TestSystemTagSecurityHTTPAIAndPrivateAccess(t *testing.T) {
	f := newTagSecurityHTTP(t, "tag_identity_ai_http")
	uuid := f.ticket(t, `{}`)
	private := f.ticket(t, `{"private":true,"internal":true,"creator_id":101,"visible_users":[101],"visibility_managers":[101]}`)
	personal := f.ticket(t, `{"access_mode":"personal","owner_user_id":101,"visible_users":[101],"visibility_managers":[101]}`)
	f.db.MustExec(`INSERT INTO user_roles(user_id,role_id) SELECT 771,id FROM roles WHERE name='Explicit classifier'`)
	for _, hidden := range []string{private, personal} {
		for _, actor := range []int{41, 771} {
			f.request(t, actor, "GET", "/api/v1/conversations/"+hidden, "", 403)
			f.request(t, actor, "POST", "/api/v1/conversations/"+hidden+"/tags", `{"tags":["🏢Intern"],"action":"add_tags"}`, 403)
		}
	}
	var original string
	if err := f.db.Get(&original, `INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,content,text_content,content_type) SELECT id,'incoming','received',105,'contact','Question','Question','text' FROM conversations WHERE uuid=$1 RETURNING uuid`, uuid); err != nil {
		t.Fatal(err)
	}
	analysis := strings.Replace(validAnalysisRequest, "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", original, 1)
	analysisPath := "/api/v1/conversations/" + uuid + "/ai-analysis-notes"
	f.request(t, 771, "POST", analysisPath, analysis, 403) // Role grant alone is not the AI allowlist.
	f.app.aiNoteAgentIDs = []int{771}
	f.request(t, 41, "POST", analysisPath, analysis, 403)
	f.request(t, 104, "POST", analysisPath, analysis, 403)
	f.request(t, 771, "POST", analysisPath, analysis, 200)
	f.request(t, 771, "POST", "/api/v1/conversations/"+private+"/ai-analysis-notes", analysis, 403)
	f.request(t, 771, "POST", "/api/v1/conversations/"+uuid+"/messages", `{"private":true,"sender_type":"agent","message":"Automatic note"}`, 200)
	f.request(t, 41, "POST", "/api/v1/conversations/"+uuid+"/messages", `{"private":true,"sender_type":"agent","message":"Ordinary note","meta":{"automatic_email_ai":true},"sender_id":771,"actor_id":771}`, 200)
	var automatic, sentPublic int
	if err := f.db.Get(&automatic, `SELECT count(*) FROM conversation_messages WHERE sender_id=771 AND meta->>'automatic_email_ai'='true'`); err != nil || automatic != 1 {
		t.Fatalf("automatic detection: %d %v", automatic, err)
	}
	if err := f.db.Get(&automatic, `SELECT count(*) FROM conversation_messages WHERE sender_id=41 AND meta->>'automatic_email_ai'='true'`); err != nil || automatic != 0 {
		t.Fatalf("forged automatic metadata: %d %v", automatic, err)
	}
	if err := f.db.Get(&sentPublic, `SELECT count(*) FROM conversation_messages WHERE type='outgoing' AND private=false`); err != nil || sentPublic != 0 {
		t.Fatalf("analysis sent without human approval: %d %v", sentPublic, err)
	}
	f.db.MustExec(`INSERT INTO service_email_addresses(address) VALUES('contact@example.com')`)
	f.request(t, 771, "POST", "/api/v1/conversations/"+uuid+"/messages", `{"private":true,"sender_type":"agent","message":"Blocked automatic note"}`, 403)
	f.db.MustExec(`UPDATE users SET enabled=false WHERE id=771`)
	f.request(t, 771, "POST", analysisPath, analysis, 401)
}

func TestSystemTagSecurityRoleGrantHTTP(t *testing.T) {
	f := newTagSecurityHTTP(t, "tag_role_grant_http")
	f.request(t, 104, "POST", "/api/v1/roles", `{"name":"Selected API classifier","description":"Explicit limited grant","permissions":["conversations:add_system_tags"]}`, 200)
	var granted, defaults int
	if err := f.db.Get(&granted, `SELECT count(*) FROM roles WHERE name='Selected API classifier' AND permissions=ARRAY['conversations:add_system_tags']::text[]`); err != nil || granted != 1 {
		t.Fatalf("permission was dropped by role validation: %d %v", granted, err)
	}
	if err := f.db.Get(&defaults, `SELECT count(*) FROM roles WHERE name IN ('Admin','Tag operator') AND 'conversations:add_system_tags'=ANY(permissions)`); err != nil || defaults != 0 {
		t.Fatalf("default role was expanded: %d %v", defaults, err)
	}
}
