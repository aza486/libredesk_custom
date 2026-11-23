package main

import (
	"encoding/json"
	"testing"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestSpamDetectionNoteHTTP(t *testing.T) {
	f := newTagSecurityHTTP(t, "spam_detection_note_http")
	f.app.aiNoteAgentIDs = []int{771}
	uuid := f.ticket(t, `{}`)
	path := "/api/v1/conversations/" + uuid + "/messages"
	note := `{"private":true,"sender_type":"agent","message":"Spam detected","purpose":"spam_detection"}`
	ordinary := `{"private":true,"sender_type":"agent","message":"Spam detected"}`
	f.db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Spam note integration',ARRAY['messages:write_spam_detection_note']);
 INSERT INTO user_roles(user_id,role_id) SELECT 771,id FROM roles WHERE name='Spam note integration'`)
	// Purpose grants no capability on ordinary customer conversations.
	f.request(t, 771, "POST", path, note, 403)
	f.request(t, 771, "POST", path, ordinary, 200)
	f.request(t, 771, "POST", path, `{"sender_type":"agent","message":"Reply","to":["contact@example.com"]}`, 200)
	f.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,12 FROM conversations WHERE uuid=$1`, uuid)
	var before string
	if err := f.db.Get(&before, `SELECT jsonb_build_array(status_id,closed_at,resolved_at,custom_attributes,ARRAY(SELECT tag_id FROM conversation_tags WHERE conversation_id=conversations.id ORDER BY tag_id))::text FROM conversations WHERE uuid=$1`, uuid); err != nil {
		t.Fatal(err)
	}
	// The grant can be assigned through the existing role API, with no default grant.
	f.request(t, 104, "POST", "/api/v1/roles", `{"name":"Selected spam note writer","permissions":["messages:write_spam_detection_note"]}`, 200)
	for _, actor := range []int{101, 104} { // Includes Admin; no implicit grant.
		f.request(t, actor, "POST", path, note, 403)
	}
	f.request(t, 771, "POST", path, ordinary, 403)
	f.request(t, 771, "POST", path, `{"sender_type":"agent","message":"Reply","to":["contact@example.com"]}`, 403)
	f.request(t, 771, "POST", path, `{"private":true,"sender_type":"agent","message":"Spam detected","purpose":"invented"}`, 403)
	f.request(t, 771, "POST", path, `{"private":true,"sender_type":"agent","message":"Spam detected","meta":{"security_note":{"purpose":"spam_detection"}}}`, 403)
	for _, body := range []string{
		`{"purpose":"spam_detection","sender_type":"agent","message":"x"}`,
		`{"purpose":"spam_detection","private":true,"sender_type":"contact","message":"x"}`,
		`{"purpose":"spam_detection","private":true,"sender_type":"agent","message":"x","to":["contact@example.com"]}`,
		`{"purpose":"spam_detection","private":true,"sender_type":"agent","message":"x","template_id":1}`,
		`{"purpose":"spam_detection","private":true,"sender_type":"agent","message":"x","reply_to_message_uuid":"forged"}`,
		`{"purpose":"spam_detection","private":true,"sender_type":"agent","message":"x","attachments":[1]}`,
		`{"purpose":"spam_detection","private":true,"sender_type":"agent","message":"x","mentions":[{"user_id":101}]}`,
	} {
		f.request(t, 771, "POST", path, body, 400)
	}
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBodyString(note)
	ctx.SetUserValue("auth_method", authMethodSession)
	ctx.SetUserValue("user", amodels.User{ID: 771})
	ctx.SetUserValue("cuuid", uuid)
	if err := handleSendMessage(&fastglue.Request{RequestCtx: ctx, Context: f.app}); err != nil || ctx.Response.StatusCode() != 403 {
		t.Fatalf("session used security purpose: %d %v %s", ctx.Response.StatusCode(), err, ctx.Response.Body())
	}
	raw := f.request(t, 771, "POST", path, note, 200)
	var result struct{ Data cmodels.Message }
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	msg := result.Data
	var meta map[string]any
	if err := json.Unmarshal(msg.Meta, &meta); err != nil || meta["security_note"] == nil || meta["automatic_email_ai"] != nil || !msg.Private || msg.Status != cmodels.MessageStatusSent {
		t.Fatalf("invalid stored security note: %s", raw)
	}
	var after string
	if err := f.db.Get(&after, `SELECT jsonb_build_array(status_id,closed_at,resolved_at,custom_attributes,ARRAY(SELECT tag_id FROM conversation_tags WHERE conversation_id=conversations.id ORDER BY tag_id))::text FROM conversations WHERE uuid=$1`, uuid); err != nil || before != after {
		t.Fatalf("spam state changed: %v", err)
	}
	hidden := f.ticket(t, `{"private":true,"internal":true,"creator_id":101,"visible_users":[101],"visibility_managers":[101]}`)
	f.request(t, 771, "POST", "/api/v1/conversations/"+hidden+"/messages", note, 403)
	// Service-mail remains protected, including for a purpose-authorized integration.
	service := f.ticket(t, `{}`)
	f.db.MustExec(`INSERT INTO service_email_addresses(address) VALUES('contact@example.com')`)
	f.request(t, 771, "POST", "/api/v1/conversations/"+service+"/messages", ordinary, 403)
	f.request(t, 771, "POST", "/api/v1/conversations/"+service+"/messages", note, 403)
	f.request(t, 771, "POST", path, note, 200)     // Spam+service still permits only the safety record.
	f.request(t, 101, "POST", path, ordinary, 200) // Manual employee behavior preserved.
	f.db.MustExec(`UPDATE roles SET permissions=array_remove(permissions,'messages:write') WHERE name='Tag operator'`)
	f.request(t, 771, "POST", path, note, 403) // Purpose grant does not replace messages:write.
}
