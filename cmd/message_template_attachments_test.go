package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/authz"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/media"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/messagetemplate"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/user"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
	"github.com/zerodha/logf"
)

type templateHTTPStore struct {
	media.Store
	data map[string][]byte
}

func (s *templateHTTPStore) Name() string { return "s3" }
func (s *templateHTTPStore) Put(name, typ string, r io.ReadSeeker) (string, error) {
	b, e := io.ReadAll(r)
	s.data[name] = b
	return name, e
}
func (s *templateHTTPStore) GetBlob(name string) ([]byte, error) { return s.data[name], nil }
func (s *templateHTTPStore) Delete(name string) error            { delete(s.data, name); return nil }
func (s *templateHTTPStore) GetURL(name, d, f string) string     { return "https://bearer.invalid/" + name }
func TestTemplateAttachmentHTTPSecurity(t *testing.T) {
	db := testutil.NewPersonalDB(t, "template_attachment_http")
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
	store := &templateHTTPStore{data: map[string][]byte{}}
	files, err := media.New(media.Opts{DB: db, Lo: &lo, I18n: lang, Store: store, RootURL: func() string { return "https://desk.test" }})
	if err != nil {
		t.Fatal(err)
	}
	app := &App{user: users, conversation: conv, authz: enforcer, i18n: lang, lo: &lo, media: files, messageTemplates: messagetemplate.New(db)}
	app.consts.Store(&constants{UploadProvider: "s3", MaxFileUploadSizeMB: 1, AllowedUploadFileExtensions: []string{"pdf", "png", "txt", "docx", "xlsx"}})
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('File writer',ARRAY['messages:write','messages:read','conversations:read','conversations:read_all']),('Kundensupport',ARRAY['message_templates:customer_support']);INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN(101,102,103,104) AND r.name='File writer';INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport';INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(101,103)`)
	var public, private string
	db.Get(&public, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`)
	db.Get(&private, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,101,101,'{"access_mode":"personal"}') RETURNING uuid`)
	request := func(actor int, method, uuid, context string, handler fastglue.FastRequestHandler, body []byte, contentType string, status int) *fasthttp.RequestCtx {
		t.Helper()
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(method)
		ctx.Request.Header.SetContentType(contentType)
		ctx.SetUserValue("uuid", uuid)
		ctx.SetUserValue("file_uuid", uuid)
		ctx.Request.URI().QueryArgs().Set("conversation_uuid", context)
		ctx.Request.SetBody(body)
		var fn fastglue.FastRequestHandler = handler
		if actor > 0 {
			key, secret, e := users.GenerateAPIKey(actor)
			if e != nil {
				t.Fatal(e)
			}
			ctx.Request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(key+":"+secret)))
			fn = perm(handler, "messages:write")
		} else {
			ctx.SetUserValue("auth_method", authMethodSignedURL)
		}
		if err := fn(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
			t.Fatal(err)
		}
		if ctx.Response.StatusCode() != status {
			t.Fatalf("actor=%d %s expected %d got %d: %s", actor, method, status, ctx.Response.StatusCode(), ctx.Response.Body())
		}
		return ctx
	}
	upload := func(name, body string, status int) mmodels.Media {
		t.Helper()
		buf := &bytes.Buffer{}
		writer := multipart.NewWriter(buf)
		headers := textproto.MIMEHeader{}
		headers.Set("Content-Disposition", `form-data; name="files"; filename="`+name+`"`)
		headers.Set("Content-Type", "image/png")
		part, _ := writer.CreatePart(headers)
		part.Write([]byte(body))
		writer.Close()
		ctx := request(101, "POST", "", private, handleTemplateAttachmentUpload, buf.Bytes(), writer.FormDataContentType(), status)
		var resp struct {
			Data mmodels.Media `json:"data"`
		}
		json.Unmarshal(ctx.Response.Body(), &resp)
		return resp.Data
	}
	pdf := upload("confirmation.pdf", "%PDF-1.4\nfixture", 200)
	if pdf.ContentType != "application/pdf" {
		t.Fatal("trusted spoofed MIME", pdf.ContentType)
	}
	for _, name := range []string{"notes.txt", "photo.png", "document.docx", "prices.xlsx"} {
		f := upload(name, "file content", 200)
		if f.ID == 0 {
			t.Fatal("supported extension rejected", name)
		}
	}
	path := upload("../../confirmation.pdf", "%PDF-1.4\nfixture", 200)
	if strings.ContainsAny(path.Filename, "/\\") {
		t.Fatal("unsafe filename", path.Filename)
	}
	active := upload("active.txt", "<html><script>alert(1)</script></html>", 200)
	activeResponse := request(101, "GET", active.UUID, private, handleServeMedia, nil, "", 200)
	if !strings.HasPrefix(string(activeResponse.Response.Header.Peek("Content-Disposition")), "attachment") || string(activeResponse.Response.Header.Peek("Content-Security-Policy")) != "sandbox" {
		t.Fatal("unsafe content serving")
	}
	upload("danger.exe", "bad", 400)
	upload("empty.pdf", "", 400)
	upload("too-large.pdf", string(bytes.Repeat([]byte("a"), 1024*1024+1)), 400)
	for _, actor := range []int{102, 103, 104} {
		request(actor, "GET", pdf.UUID, public, handleServeMedia, nil, "", 404)
	}
	request(0, "GET", pdf.UUID, "", handleServeMedia, nil, "", 401)
	own := request(101, "GET", pdf.UUID, private, handleServeMedia, nil, "", 200)
	if string(own.Response.Body()) != "%PDF-1.4\nfixture" || string(own.Response.Header.Peek("Cache-Control")) != "private, no-store" || len(own.Response.Header.Peek("Location")) != 0 {
		t.Fatal("unsafe file delivery")
	}
	tpl, err := app.messageTemplates.Create(messagetemplate.Access{UserID: 101, ConversationUUID: private}, messagetemplate.Input{Name: "File", Scope: "personal", AttachmentIDs: []int{pdf.ID}})
	if err != nil {
		t.Fatal(err)
	}
	// Public adoption needs no inbox manager or SMTP connection at all. This
	// also covers unavailable transports and transports without a SIZE extension.
	// Keep accepting the legacy content field without estimating a MIME size.
	publicApply := request(101, "POST", tpl.UUID, public, handleApplyMessageTemplate,
		[]byte(`{"draft_type":"reply","attachment_ids":[],"content":"legacy client"}`), "application/json", 200)
	var publicResult struct {
		Data messagetemplate.Applied `json:"data"`
	}
	if err := json.Unmarshal(publicApply.Response.Body(), &publicResult); err != nil {
		t.Fatal(err)
	}
	if len(publicResult.Data.Attachments) != 1 || publicResult.Data.Attachments[0].UUID == pdf.UUID {
		t.Fatal("public reply lost independent PDF reference", publicResult.Data)
	}
	request(101, "POST", tpl.UUID, public, handleApplyMessageTemplate,
		[]byte(`{"draft_type":"reply","revision":999,"attachment_ids":[]}`), "application/json", 409)
	raw := []byte(`{"draft_type":"private_note","attachment_ids":[],"content":""}`)
	stored := store.data[pdf.UUID]
	delete(store.data, pdf.UUID)
	request(101, "POST", tpl.UUID, private, handleApplyMessageTemplate, raw, "application/json", 400)
	store.data[pdf.UUID] = stored
	applied := request(101, "POST", tpl.UUID, private, handleApplyMessageTemplate, raw, "application/json", 200)
	var result struct {
		Data messagetemplate.Applied `json:"data"`
	}
	json.Unmarshal(applied.Response.Body(), &result)
	if len(result.Data.Attachments) != 1 {
		t.Fatal("missing adopted file")
	}
	clone := result.Data.Attachments[0]
	if err := app.messageTemplates.Delete(messagetemplate.Access{UserID: 101, ConversationUUID: private}, tpl.UUID, tpl.Revision); err != nil {
		t.Fatal(err)
	}
	request(101, "GET", pdf.UUID, private, handleServeMedia, nil, "", 404)
	request(101, "GET", clone.UUID, private, handleServeMedia, nil, "", 200)
	request(102, "GET", clone.UUID, public, handleServeMedia, nil, "", 404)
	// A known protected source cannot bypass adoption through the generic send resolver.
	if _, err = getUnassociatedMedia(app, []int{pdf.ID}, 102, public); err == nil {
		t.Fatal("source bypass")
	}
	// Force a pending draft reference through another context: denied before bytes.
	if _, err = getUnassociatedMedia(app, []int{clone.ID}, 101, public); err == nil {
		t.Fatal("cross-conversation reference")
	}
	db.MustExec(`UPDATE roles SET permissions=array_remove(permissions,'messages:write') WHERE name='File writer'`)
	// Injecting an old authenticated identity still fails the new live file check.
	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue("uuid", clone.UUID)
	ctx.SetUserValue("user", amodels.User{ID: 101})
	ctx.Request.URI().QueryArgs().Set("conversation_uuid", private)
	if err = handleServeMedia(&fastglue.Request{RequestCtx: ctx, Context: app}); err != nil {
		t.Fatal(err)
	}
	if ctx.Response.StatusCode() != 403 {
		t.Fatal("revoked permission served bytes", ctx.Response.StatusCode())
	}
}
