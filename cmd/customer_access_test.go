package main

import (
	"context"
	"encoding/json"
	activitylog "github.com/abhinavxd/libredesk/internal/activity_log"
	"github.com/abhinavxd/libredesk/internal/automation"
	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/media"
	notifier "github.com/abhinavxd/libredesk/internal/notification"
	"github.com/abhinavxd/libredesk/internal/search"
	"github.com/abhinavxd/libredesk/internal/setting"
	"github.com/abhinavxd/libredesk/internal/team"
	"github.com/abhinavxd/libredesk/internal/template"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/abhinavxd/libredesk/internal/ws"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"strconv"
	"strings"
	"sync"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

// Reproduces the exact mismatch: the same identity is rejected by search/list
// authorization but accepted by UUID detail/history through broad scope grants.
func TestCustomerAccessRevokedBroadPermissionsHTTP(t *testing.T) {
	db := testutil.NewPersonalDB(t, "customer_access_http")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	conv, err := conversation.New(nil, lang, nil, nil, nil, nil, users, nil, customerReturnMedia{}, nil, nil, nil, nil, nil, nil, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	enforcer, _ := authz.NewEnforcer(&lo, lang)
	app := &App{ctx: context.Background(), user: users, conversation: conv, authz: enforcer, i18n: lang, lo: &lo}
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Former employee',ARRAY['conversations:read','conversations:read_assigned','conversations:read_all','conversations:read_unassigned','messages:read']);
 INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Former employee';`)
	var uuid string
	if err = db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,102,101,'secret-subject','{"customer_visibility":true,"visible_users":[103],"creator_id":103,"visibility_managers":[103]}') RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, uuid)
	allowed, err := conv.FilterAuthorizedListUUIDs(101, []string{uuid})
	if err != nil || len(allowed) != 0 {
		t.Fatalf("search authorization %v %v", allowed, err)
	}
	for _, handler := range []fastglue.FastRequestHandler{handleGetConversation, handleGetMessages, handleGetConversationParticipants, handleDownloadConversationTranscript, handleGetContactPageVisits} {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue("user", amodels.User{ID: 101})
		ctx.SetUserValue("uuid", uuid)
		ctx.SetUserValue("auth_method", authMethodSession)
		if err := handler(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != 403 {
			t.Fatalf("UUID returned %d instead of 403: %s", ctx.Response.StatusCode(), ctx.Response.Body())
		}
		if strings.Contains(string(ctx.Response.Body()), "secret-subject") {
			t.Fatal("ticket content in denied response")
		}
	}
}

// Tests use real HTTP routing and PostgreSQL managers. Authentication identity is
// injected at the test transport boundary, as it is by production middleware.
type accessTestBlobStore struct{ media.Store }

func (accessTestBlobStore) GetURL(name, _, _ string) string { return "https://storage.invalid/" + name }
func (accessTestBlobStore) Name() string                    { return "s3" }
func (accessTestBlobStore) GetBlob(string) ([]byte, error)  { return []byte("secret-attachment"), nil }

func TestCustomerAccessLifecycleHTTP(t *testing.T) {
	db := testutil.NewPersonalDB(t, "customer_access_lifecycle")
	lang := testutil.NewI18n(t)
	lo := logf.New(logf.Opts{})
	users, err := user.New(lang, user.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	teams, err := team.New(team.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	engine, err := automation.New(automation.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := setting.New(setting.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	files, err := media.New(media.Opts{DB: db, Lo: &lo, I18n: lang, Store: accessTestBlobStore{}, RootURL: func() string { return "https://desk.example" }})
	if err != nil {
		t.Fatal(err)
	}
	notifications, err := notifier.NewUserNotificationManager(notifier.UserNotificationOpts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New(&lo, db, nil, nil, nil, lang)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := notifier.NewDispatcher(notifier.DispatcherOpts{InApp: notifications, Lo: &lo})
	hub := ws.NewHub(&lo, nil)
	conv, err := conversation.New(hub, lang, customerReturnSLA{}, nil, nil, customerReturnInbox{}, users, teams, files, customerReturnSettings{}, nil, engine, tmpl, customerReturnWebhooks{}, dispatcher, conversation.Opts{DB: db, Lo: &lo})
	if err != nil {
		t.Fatal(err)
	}
	hub.SetConversationStore(conv)
	enforcer, _ := authz.NewEnforcer(&lo, lang)
	searcher, err := search.New(search.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	activity, err := activitylog.New(activitylog.Opts{DB: db, Lo: &lo, I18n: lang})
	if err != nil {
		t.Fatal(err)
	}
	notifications.SetConversationAuthorizer(conv.FilterAuthorizedListUUIDs)
	app := &App{ctx: context.Background(), user: users, conversation: conv, authz: enforcer, i18n: lang, lo: &lo, setting: settings, media: files, search: searcher, activityLog: activity, userNotification: notifications}
	app.consts.Store(&constants{UploadProvider: "s3"})
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Access employee',ARRAY['conversations:read','conversations:read_assigned','conversations:read_team_all','conversations:read_all','conversations:read_unassigned','messages:read']),('Kundensupport',ARRAY['conversations:read','messages:read']);
 INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Access employee';
 INSERT INTO user_roles(user_id,role_id) SELECT 102,id FROM roles WHERE name='Access employee';
 INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport';
 INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin';
 INSERT INTO teams(id,name,conversation_assignment_type) VALUES(202,'External','Manual');
 INSERT INTO team_members(team_id,user_id) VALUES(202,101);
 INSERT INTO tags(id,name) VALUES(12,'🗑Spam')`)
	var uuid string
	if err = db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,subject,custom_attributes) VALUES(105,102,101,'unique-customer-secret','{"customer_visibility":true,"creator_id":103,"visibility_managers":[103],"visible_users":[103]}') RETURNING uuid`); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, uuid)
	var msgID int
	var msgUUID string
	if err = db.QueryRow(`INSERT INTO conversation_messages(conversation_id,type,status,sender_id,sender_type,private,content,text_content,content_type) SELECT id,'outgoing','sent',101,'agent',true,'private-ai-secret','private-ai-secret','html' FROM conversations WHERE uuid=$1 RETURNING id,uuid`, uuid).Scan(&msgID, &msgUUID); err != nil {
		t.Fatal(err)
	}
	var fileUUID string
	if err = db.Get(&fileUUID, `INSERT INTO media(store,filename,content_type,model_type,model_id,private,content_id,size) VALUES('s3','secret.png','image/png','messages',$1,false,'',17) RETURNING uuid`, msgID); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversation_drafts(conversation_id,user_id,type,content) SELECT id,101,'reply','draft-secret' FROM conversations WHERE uuid=$1`, uuid)
	db.MustExec(`INSERT INTO user_notifications(user_id,notification_type,title,conversation_id) SELECT 101,'assignment','notification-secret',id FROM conversations WHERE uuid=$1`, uuid)
	g := fastglue.New()
	g.SetContext(app)
	wrap := func(next fastglue.FastRequestHandler) fastglue.FastRequestHandler {
		return func(r *fastglue.Request) error {
			id, _ := strconv.Atoi(string(r.RequestCtx.Request.Header.Peek("X-Test-Agent")))
			if id > 0 {
				r.RequestCtx.SetUserValue("user", amodels.User{ID: id})
				r.RequestCtx.SetUserValue("auth_method", authMethodSession)
			} else {
				r.RequestCtx.SetUserValue("auth_method", authMethodSignedURL)
			}
			return next(r)
		}
	}
	g.GET("/api/v1/conversations/search", wrap(handleSearchConversations))
	g.GET("/api/v1/messages/search", wrap(handleSearchMessages))
	g.GET("/api/v1/conversations/visible", wrap(handleGetVisibleConversations))
	g.GET("/api/v1/conversations/{uuid}", wrap(handleGetConversation))
	g.GET("/api/v1/conversations/{uuid}/messages", wrap(handleGetMessages))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}", wrap(handleGetMessage))
	g.GET("/api/v1/conversations/{uuid}/participants", wrap(handleGetConversationParticipants))
	g.GET("/api/v1/conversations/{uuid}/transcript", wrap(handleDownloadConversationTranscript))
	g.GET("/api/v1/conversations/{uuid}/page-visits", wrap(handleGetContactPageVisits))
	g.GET("/api/v1/conversations/{cuuid}/reply-sources", wrap(handleGetReplySources))
	g.GET("/api/v1/conversations/{cuuid}/messages/{uuid}/manual-feedback-reason", wrap(handleManualReviewReason))
	g.PUT("/api/v1/conversations/{uuid}/remove-visible-user", wrap(handleRemoveVisibleUser))
	g.DELETE("/api/v1/messages/{uuid}", wrap(handleDeletePrivateNote))
	g.DELETE("/api/v1/conversations/{cuuid}/messages/{uuid}", wrap(handleDeleteMessage))
	g.GET("/api/v1/drafts", wrap(handleGetAllDrafts))
	g.GET("/api/v1/contacts/{id}/export", wrap(handleExportContact))
	g.GET("/api/v1/notifications", wrap(handleGetUserNotifications))
	g.GET("/uploads/{uuid}", wrap(handleServeMedia))
	g.GET("/api/v1/context-links/{id}/url", wrap(handleGetContextLinkURL))
	g.GET("/api/v1/ai/copilot/messages", wrap(handleGetCopilotMessages))
	call := func(method, path string, id, want int, body string) []byte {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI(path)
		ctx.Request.Header.SetMethod(method)
		ctx.Request.Header.Set("X-Test-Agent", strconv.Itoa(id))
		ctx.Request.Header.SetContentType("application/json")
		ctx.Request.SetBodyString(body)
		g.Handler()(ctx)
		if ctx.Response.StatusCode() != want {
			t.Fatalf("%s %s as %d: %d want %d: %s", method, path, id, ctx.Response.StatusCode(), want, ctx.Response.Body())
		}
		if want == 403 && (strings.Contains(string(ctx.Response.Body()), "unique-customer-secret") || strings.Contains(string(ctx.Response.Body()), "private-ai-secret")) {
			t.Fatal("denial leaked contents")
		}
		return append([]byte(nil), ctx.Response.Body()...)
	}
	base := "/api/v1/conversations/" + uuid
	actor := umodels.User{ID: 103}
	if err = conv.UpdateConversationUserAssignee(uuid, 101, actor); err != nil {
		t.Fatal(err)
	}
	call("GET", base, 101, 200, "")
	call("PUT", base+"/remove-visible-user", 103, 200, `{"user_id":101}`)
	call("GET", base, 101, 200, "") // assignment remains independently effective
	call("GET", base+"/messages", 101, 200, "")
	body := call("GET", "/uploads/"+fileUUID, 101, 200, "")
	if string(body) != "secret-attachment" {
		t.Fatal("attachment proxy", string(body))
	}
	// Team access remains valid after independent visibility removal.
	if err = conv.UpdateConversationTeamAssignee(uuid, 202, actor); err != nil {
		t.Fatal(err)
	}
	call("PUT", base+"/remove-visible-user", 103, 200, `{"user_id":101}`)
	call("GET", base, 101, 200, "")
	client := &ws.Client{ID: 101, Hub: hub, Send: make(chan wsmodels.WSMessage, 100)}
	hub.AddClient(client)
	defer hub.RemoveClient(client)
	hub.SubscribeOpenConv(client, uuid)
	hub.SubscribeListReplace(client, []string{uuid})
	if err = conv.ReturnCustomerToSupport(uuid, actor); err != nil {
		t.Fatal(err)
	}
	for len(client.Send) > 0 {
		<-client.Send
	}
	call("PUT", base+"/remove-visible-user", 103, 200, `{"user_id":101}`)
	// An already-open subscriber receives only UUID + revocation, no attrs/content.
	revoked := false
	for len(client.Send) > 0 {
		var event struct {
			Data map[string]any `json:"data"`
		}
		json.Unmarshal((<-client.Send).Data, &event)
		if event.Data["access_revoked"] != true || len(event.Data) != 2 {
			t.Fatalf("revoked socket got ticket data: %v", event.Data)
		}
		revoked = true
	}
	if !revoked || len(hub.ListSubscribers(uuid)) != 0 {
		t.Fatal("revoked subscriptions were retained")
	}
	conv.BroadcastMessageUpdate(uuid, msgUUID, map[string]any{"content": "new-secret"})
	conv.BroadcastConversationUpdate(uuid, map[string]any{"subject": "new-secret"})
	conv.BroadcastContactUpdate(105, map[string]any{"email": "new-secret"})
	if len(client.Send) != 0 {
		t.Fatal("protected update after revoke")
	}
	for _, id := range []int{101, 102} {
		for _, path := range []string{base, base + "/messages", base + "/messages/" + msgUUID, base + "/participants", base + "/transcript", base + "/page-visits", base + "/reply-sources", base + "/messages/" + msgUUID + "/manual-feedback-reason", "/uploads/" + fileUUID, "/uploads/" + image.ThumbPrefix + fileUUID} {
			call("GET", path, id, 403, "")
		}
	}
	call("GET", "/api/v1/context-links/1/url?conversation_uuid="+uuid, 101, 403, "")
	call("GET", "/api/v1/ai/copilot/messages?conversation_uuid="+uuid, 101, 403, "")
	call("DELETE", "/api/v1/messages/"+msgUUID, 101, 403, "")
	call("GET", "/uploads/"+fileUUID, 0, 401, "") // a former bearer URL cannot bypass ACL
	for _, path := range []string{"/api/v1/conversations/search?query=unique-customer-secret", "/api/v1/messages/search?query=private-ai-secret", "/api/v1/conversations/visible", "/api/v1/drafts", "/api/v1/contacts/105/export", "/api/v1/notifications"} {
		data := call("GET", path, 101, 200, "")
		for _, secret := range []string{uuid, "unique-customer-secret", "private-ai-secret", "draft-secret", "notification-secret"} {
			if strings.Contains(string(data), secret) {
				t.Fatalf("%s leaked %s", path, secret)
			}
		}
	}
	// Deleting a private note must not turn its file into an unassociated
	// upload which could bypass the conversation ACL.
	call("DELETE", base+"/messages/"+msgUUID, 104, 200, "")
	call("GET", "/uploads/"+fileUUID, 101, 403, "")
	var linkedID int
	if err = db.Get(&linkedID, `SELECT model_id FROM media WHERE uuid=$1`, fileUUID); err != nil || linkedID != msgID {
		t.Fatalf("deleted note lost attachment ACL: %d %v", linkedID, err)
	}
	deleted := call("GET", base+"/messages/"+msgUUID, 103, 200, "")
	if strings.Contains(string(deleted), fileUUID) {
		t.Fatal("deleted note still projects attachments")
	}
	for _, id := range []int{103, 104} {
		call("GET", base, id, 200, "")
	}
	// Live team membership wins over an already-warmed agent cache.
	db.MustExec(`UPDATE conversations SET assigned_team_id=202 WHERE uuid=$1`, uuid)
	call("GET", base, 101, 200, "")
	db.MustExec(`DELETE FROM team_members WHERE team_id=202 AND user_id=101`)
	call("GET", base, 101, 403, "")
	if err = conv.UpdateConversationUserAssignee(uuid, 101, actor); err != nil {
		t.Fatal(err)
	}
	call("GET", base, 101, 200, "")
	call("GET", base+"/messages", 101, 200, "")
	// Revoke/forward operations share the row lock. The former employee must
	// not retain broad-scope access in either serialization order.
	for range 3 {
		db.MustExec(`UPDATE conversations SET assigned_team_id=NULL WHERE uuid=$1`, uuid)
		if err = conv.UpdateConversationUserAssignee(uuid, 101, actor); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		errors := make(chan error, 2)
		wg.Add(2)
		go func() { defer wg.Done(); errors <- conv.RemoveVisibleUser(uuid, 101) }()
		go func() { defer wg.Done(); errors <- conv.UpdateConversationUserAssignee(uuid, 102, actor) }()
		wg.Wait()
		close(errors)
		for err := range errors {
			if err != nil {
				t.Fatal(err)
			}
		}
		call("GET", base, 101, 403, "")
		call("GET", base, 102, 200, "")
	}
	// A private inbox is not opened by customer support or broad employee grants.
	db.MustExec(`UPDATE conversations SET inbox_id=101,custom_attributes='{"access_mode":"personal","visible_users":[101]}' WHERE uuid=$1`, uuid)
	call("GET", base, 102, 403, "")
	call("GET", base, 103, 403, "")
	call("GET", base, 104, 200, "")
}
