package conversation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/template"
	tmodels "github.com/abhinavxd/libredesk/internal/template/models"
	"github.com/abhinavxd/libredesk/internal/user"
)

func TestReplyAuthorWorkPhoneRendering(t *testing.T) {
	m := emailThreadFixture(t, "reply_author_phone")
	users, err := user.New(m.i18n, user.Opts{DB: m.db, Lo: m.lo})
	if err != nil {
		t.Fatal(err)
	}
	m.userStore = users
	m.template, err = template.New(m.lo, m.db, nil, nil, nil, m.i18n)
	if err != nil {
		t.Fatal(err)
	}
	signature, err := m.template.Create(tmodels.Template{Name: "Phone signature", Type: "email_outgoing", Body: `{{ template "content" . }}<p>{{ .Author.FullName }}</p><table><tr><td>T</td><td>{{ if .Author.WorkPhone }}{{ .Author.WorkPhone }}{{ else }}0351 40424-0{{ end }}</td></tr><tr><td>F</td><td>0351 40424-55</td></tr></table><img src="https://example.com/logo.png">`})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := m.template.Create(tmodels.Template{Name: "No phone", Type: "email_outgoing", Body: `{{ template "content" . }}<p>Unchanged</p>`})
	if err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(map[string]any{"template_id": signature.ID})
	m.db.MustExec(`UPDATE users SET first_name='Reply',last_name='Author' WHERE id=102`)
	// Creator/assignee differs from the actual message sender, in both inbox types.
	for _, inboxID := range []int{101, 102} {
		var uuid string
		if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,assigned_user_id) VALUES(105,$1,101,101) RETURNING uuid`, inboxID); err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			id          int
			phone, want string
		}{
			{102, "0351 40424-23", "0351 40424-23"}, {101, "0351 40424-99", "0351 40424-99"},
			{102, "+49 351 12345678", "+49 351 12345678"}, {102, "", "0351 40424-0"},
		} {
			if err := users.UpdateWorkPhone(tc.id, tc.phone); err != nil {
				t.Fatal(err)
			}
			msg := models.Message{ConversationUUID: uuid, SenderID: tc.id, Content: "<p>Body</p>", Meta: meta}
			if err := m.RenderMessageInTemplate("email", &msg); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(msg.Content, "<td>"+tc.want+"</td>") || strings.Count(msg.Content, "<p>Body</p>") != 1 || !strings.Contains(msg.Content, `<td>0351 40424-55</td>`) || !strings.Contains(msg.Content, `<img src="https://example.com/logo.png">`) {
				t.Fatal(msg.Content)
			}
			if tc.id == 102 && !strings.Contains(msg.Content, "Reply Author") {
				t.Fatal("FullName broken")
			}
		}
		// Defense in depth for legacy/direct DB values; text/template must not emit HTML.
		m.db.MustExec(`UPDATE users SET work_phone_number='<img src=x>&"' WHERE id=102`)
		msg := models.Message{ConversationUUID: uuid, SenderID: 102, Content: "Body", Meta: meta}
		if err := m.RenderMessageInTemplate("email", &msg); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(msg.Content, "<img src=x>") || !strings.Contains(msg.Content, "&lt;img src=x&gt;&amp;&#34;") {
			t.Fatal("unescaped phone", msg.Content)
		}
		// System author has no personal identity even if a DB value exists.
		m.db.MustExec(`UPDATE users SET email='System' WHERE id=102`)
		if err := m.RenderMessageInTemplate("email", &models.Message{ConversationUUID: uuid, SenderID: 102, Content: "Body", Meta: meta}); err != nil {
			t.Fatal(err)
		}
		data, err := m.BuildTemplateData(uuid, 102)
		if err != nil || data["Author"].(map[string]any)["WorkPhone"] != "" {
			t.Fatal("system phone", err)
		}
		rendered, err := m.template.RenderEmailWithTemplateID(data, "Body", plain.ID)
		if err != nil || rendered != "Body<p>Unchanged</p>" {
			t.Fatal("plain signature changed", rendered, err)
		}
		m.db.MustExec(`UPDATE users SET email='other@example.com' WHERE id=102`)
	}
}
