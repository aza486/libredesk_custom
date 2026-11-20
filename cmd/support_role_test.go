package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/role"
	"github.com/abhinavxd/libredesk/internal/role/models"
	"github.com/abhinavxd/libredesk/internal/tag"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func newSupportRoleHTTP(t *testing.T) *tagSecurityHTTP {
	t.Helper()
	app, db := systemAttributesHTTPFixture(t, "support_role_http")
	db.MustExec(`DELETE FROM roles WHERE name='Kundensupport'; DELETE FROM tags; DELETE FROM user_roles WHERE user_id=103`)
	if err := ensureSystemData(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	var err error
	app.role, err = role.New(role.Opts{DB: db, Lo: app.lo, I18n: app.i18n})
	if err != nil {
		t.Fatal(err)
	}
	g := fastglue.NewGlue()
	g.SetContext(app)
	g.POST("/api/v1/roles", perm(handleCreateRole, "roles:manage"))
	g.PUT("/api/v1/roles/{id}", perm(handleUpdateRole, "roles:manage"))
	g.DELETE("/api/v1/roles/{id}", perm(handleDeleteRole, "roles:manage"))
	g.GET("/api/v1/conversations/{uuid}", perm(handleGetConversation, "conversations:read"))
	g.PUT("/api/v1/conversations/{uuid}/add-visible-user", auth(handleAddVisibleUser))
	g.PUT("/api/v1/conversations/{uuid}/remove-visible-user", auth(handleRemoveVisibleUser))
	g.GET("/api/v1/service-email-addresses", serviceEmailManagement(handleGetServiceEmailAddresses))
	dir, err := os.MkdirTemp("/tmp", "ld-support-http-")
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
	for _, id := range []int{101, 102, 103, 104} {
		key, secret, err := app.user.GenerateAPIKey(id)
		if err != nil {
			t.Fatal(err)
		}
		f.keys[id] = [2]string{key, secret}
	}
	return f
}

func TestCustomerSupportRoleHTTP(t *testing.T) {
	f := newSupportRoleHTTP(t)
	var id, members int
	if err := f.db.Get(&id, `SELECT id FROM roles WHERE name=$1`, models.RoleCustomerSupport); err != nil {
		t.Fatal(err)
	}
	if err := f.db.Get(&members, `SELECT count(*) FROM user_roles WHERE role_id=$1`, id); err != nil || members != 0 {
		t.Fatal("automatic membership", members, err)
	}
	customer := f.ticket(t, `{"customer_visibility":true,"visible_users":[],"visibility_managers":[]}`)
	f.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,$2 FROM conversations WHERE uuid=$1`, customer, tag.SystemTagCustomer)
	path := "/api/v1/conversations/" + customer
	f.request(t, 103, "GET", path, "", 403)
	// Explicit assignment only, after asserting no bootstrap assignment.
	f.db.MustExec(`INSERT INTO user_roles(user_id,role_id) VALUES(103,$1)`, id)
	f.app.user.InvalidateAllAgentCache()
	f.request(t, 103, "GET", path, "", 200)
	f.request(t, 103, "PUT", path+"/add-visible-user", `{"user_id":102}`, 200)
	f.request(t, 102, "GET", path, "", 200)
	f.request(t, 103, "PUT", path+"/remove-visible-user", `{"user_id":102}`, 200)
	f.request(t, 102, "GET", path, "", 403)
	f.request(t, 103, "GET", "/api/v1/service-email-addresses", "", 200)
	private := f.ticket(t, `{"private":true,"internal":true,"creator_id":101,"visible_users":[101]}`)
	f.request(t, 103, "GET", "/api/v1/conversations/"+private, "", 403)
	personal := f.ticket(t, `{"access_mode":"personal","owner_user_id":101,"visible_users":[]}`)
	f.request(t, 103, "GET", "/api/v1/conversations/"+personal, "", 403)
	rolePath := fmt.Sprintf("/api/v1/roles/%d", id)
	body := func(name string, perms []string) string {
		raw, err := json.Marshal(map[string]any{"name": name, "description": "Test", "permissions": perms})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	required := []string{"conversations:read", "messages:read", "service_emails:manage"}
	f.request(t, 104, "DELETE", rolePath, "", 400)
	f.request(t, 104, "PUT", rolePath, body("Renamed", required), 400)
	f.request(t, 104, "PUT", rolePath, body(models.RoleCustomerSupport, []string{"conversations:read"}), 400)
	f.request(t, 104, "POST", "/api/v1/roles", body(models.RoleCustomerSupport, required), 400)
	f.request(t, 103, "PUT", rolePath, body(models.RoleCustomerSupport, required), 403)
	if err := role.CheckCustomerSupportRole(t.Context(), f.db); err != nil {
		t.Fatal(err)
	}
}
