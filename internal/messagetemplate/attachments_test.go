package messagetemplate

import (
	"sync"
	"testing"

	"github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestTemplateAttachmentsLifecycle(t *testing.T) {
	db := testutil.NewPersonalDB(t, "template_files")
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Writer',ARRAY['messages:write']),('Kundensupport',ARRAY['message_templates:customer_support']); INSERT INTO user_roles(user_id,role_id) SELECT u.id,r.id FROM users u CROSS JOIN roles r WHERE u.id IN(101,102,103,104) AND r.name='Writer';INSERT INTO user_roles(user_id,role_id) SELECT 103,id FROM roles WHERE name='Kundensupport'`)
	var public, private string
	db.Get(&public, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`)
	db.Get(&private, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,101,101) RETURNING uuid`)
	m := New(db)
	owner := Access{101, private}
	other := Access{102, private}
	support := Access{103, public}
	upload := func(a Access, name string, size int) models.Media {
		t.Helper()
		var f models.Media
		if err := db.Get(&f, `INSERT INTO media(store,filename,content_type,size,model_type,disposition,content_id) VALUES('fs',$1,'application/pdf',$2,'message_templates','attachment','') RETURNING *`, name, size); err != nil {
			t.Fatal(err)
		}
		if err := m.Stage(a, f.ID, 10000); err != nil {
			t.Fatal(err)
		}
		return f
	}
	pdf := upload(owner, "Gelangensbestätigung.pdf", 100)
	if err := m.AuthorizeFile(other, pdf.UUID); code(err) != 404 {
		t.Fatal("foreign staging", err)
	}
	if err := m.AuthorizeFile(owner, pdf.UUID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Create(other, Input{Name: "Forged", Scope: "personal", AttachmentIDs: []int{pdf.ID}}); code(err) != 404 {
		t.Fatal("stolen reference", err)
	}
	tmpl, err := m.Create(owner, Input{Name: "Nur Anhänge", Scope: "personal", AttachmentIDs: []int{pdf.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if tmpl.Content != "" || tmpl.AttachmentCount != 1 {
		t.Fatal(tmpl)
	}
	listed, err := m.List(owner)
	if err != nil || len(listed.Templates) != 1 || listed.Templates[0].AttachmentCount != 1 || len(listed.Templates[0].Attachments) != 0 {
		t.Fatal("list must contain only metadata", listed, err)
	}
	if err := m.DiscardUpload(owner, pdf.UUID); code(err) != 404 {
		t.Fatal("cannot discard saved file", err)
	}
	adopt := func(a Access, ids []int) Applied {
		t.Helper()
		v, err := m.Apply(a, tmpl.UUID, ApplyInput{DraftType: "private_note", AttachmentIDs: ids}, 10000)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	first := adopt(owner, nil)
	if len(first.Attachments) != 1 || first.Attachments[0].UUID == pdf.UUID {
		t.Fatal("independent metadata required", first)
	}
	twice := adopt(owner, []int{first.Attachments[0].ID})
	if len(twice.Attachments) != 1 || twice.Attachments[0].ID != first.Attachments[0].ID {
		t.Fatal("duplicate adoption", twice)
	}
	removedThenReadopt := adopt(owner, nil)
	if removedThenReadopt.Attachments[0].ID != first.Attachments[0].ID {
		t.Fatal("unnecessary reference duplicate")
	}
	if _, err := m.Apply(other, tmpl.UUID, ApplyInput{DraftType: "private_note"}, 10000); code(err) != 404 {
		t.Fatal("foreign apply", err)
	}
	if _, err := m.Apply(owner, tmpl.UUID, ApplyInput{DraftType: "private_note"}, 10); code(err) != 400 {
		t.Fatal("current per-file limit", err)
	}
	replacement := upload(owner, "Preisliste.pdf", 120)
	updated, err := m.Update(owner, tmpl.UUID, Input{Name: tmpl.Name, Scope: "personal", AttachmentIDs: []int{replacement.ID}, Revision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Update(owner, tmpl.UUID, Input{Name: tmpl.Name, Scope: "personal", AttachmentIDs: []int{pdf.ID}, Revision: 1}); code(err) != 409 {
		t.Fatal("attachment conflict", err)
	}
	if err = m.AuthorizeFile(owner, pdf.UUID); code(err) != 404 {
		t.Fatal("retired source downloadable", err)
	}
	if _, err = m.Update(owner, tmpl.UUID, Input{Name: tmpl.Name, Scope: "personal", AttachmentIDs: []int{}, Revision: updated.Revision}); code(err) != 400 {
		t.Fatal("empty file-only template", err)
	}
	if err = m.Delete(owner, tmpl.UUID, updated.Revision); err != nil {
		t.Fatal(err)
	}
	var refs int
	db.Get(&refs, `SELECT count(*) FROM media_blob_references WHERE media_id=$1`, first.Attachments[0].ID)
	if refs != 1 {
		t.Fatal("template delete damaged draft reference")
	}
	global := upload(support, "Global.pdf", 150)
	shared, err := m.Create(support, Input{Name: "Global", Scope: "customer_support", Content: "<p>Text</p>", AttachmentIDs: []int{global.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err = m.AuthorizeFile(Access{103, private}, global.UUID); code(err) != 404 {
		t.Fatal("private global access", err)
	}
	if _, err = m.Apply(support, shared.UUID, ApplyInput{DraftType: "reply", AttachmentIDs: []int{first.Attachments[0].ID}}, 10000); code(err) != 403 {
		t.Fatal("foreign draft clone", err)
	}
	db.MustExec(`DELETE FROM user_roles WHERE user_id=103 AND role_id=(SELECT id FROM roles WHERE name='Kundensupport')`)
	if err = m.AuthorizeFile(support, global.UUID); code(err) != 404 {
		t.Fatal("revoked file access", err)
	}
}
func TestTemplateAttachmentConcurrentApply(t *testing.T) {
	db := testutil.NewPersonalDB(t, "template_files_concurrent")
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Writer',ARRAY['messages:write']);INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Writer'`)
	var context string
	db.Get(&context, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`)
	var file models.Media
	db.Get(&file, `INSERT INTO media(store,filename,content_type,size,model_type,content_id) VALUES('fs','one.pdf','application/pdf',100,'message_templates','') RETURNING *`)
	m := New(db)
	a := Access{101, context}
	if err := m.Stage(a, file.ID, 1000); err != nil {
		t.Fatal(err)
	}
	template, err := m.Create(a, Input{Name: "PDF", Scope: "personal", AttachmentIDs: []int{file.ID}})
	if err != nil {
		t.Fatal(err)
	}
	ids := make(chan int, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, e := m.Apply(a, template.UUID, ApplyInput{DraftType: "reply"}, 10000)
			if e != nil {
				errs <- e
				return
			}
			ids <- v.Attachments[0].ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		t.Fatal(e)
	}
	var first int
	for id := range ids {
		if first != 0 && id != first {
			t.Fatal("concurrent duplicate", first, id)
		}
		first = id
	}
}

func TestTemplateMultipleAttachmentKinds(t *testing.T) {
	db := testutil.NewPersonalDB(t, "template_multiple_files")
	db.MustExec(`INSERT INTO roles(name,permissions) VALUES('Writer',ARRAY['messages:write']);INSERT INTO user_roles(user_id,role_id) SELECT 101,id FROM roles WHERE name='Writer'`)
	var context string
	db.Get(&context, `INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101) RETURNING uuid`)
	m := New(db)
	a := Access{101, context}
	var ids []int
	for _, kind := range []struct{ name, mime string }{{"confirmation.pdf", "application/pdf"}, {"prices.pdf", "application/pdf"}, {"photo.png", "image/png"}, {"terms.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"}, {"prices.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}, {"info.txt", "text/plain"}} {
		var f models.Media
		db.Get(&f, `INSERT INTO media(store,filename,content_type,size,model_type,content_id) VALUES('fs',$1,$2,100,'message_templates','') RETURNING *`, kind.name, kind.mime)
		if err := m.Stage(a, f.ID, 100000); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, f.ID)
	}
	template, err := m.Create(a, Input{Name: "Files", Scope: "personal", AttachmentIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	// A reduced regular per-file limit still rejects adoption atomically.
	if _, err := m.Apply(a, template.UUID, ApplyInput{DraftType: "reply"}, 99); code(err) != 400 {
		t.Fatal("current upload limit not enforced", err)
	}
	var references int
	if err := db.Get(&references, `SELECT count(*) FROM media_blob_references`); err != nil || references != 0 {
		t.Fatal("failed adoption left references", references, err)
	}
	var existing models.Media
	if err := db.Get(&existing, `INSERT INTO media(store,filename,content_type,size,model_type,content_id) VALUES('fs','existing.pdf','application/pdf',100,'messages','') RETURNING *`); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"reply", "private_note"} {
		t.Run(mode, func(t *testing.T) {
			// Every file meets the upload cap. There is no separate template
			// aggregate byte limit or MIME estimate, even with existing files.
			first, err := m.Apply(a, template.UUID, ApplyInput{DraftType: mode, AttachmentIDs: []int{existing.ID}}, 100)
			if err != nil || len(first.Attachments) != 7 || first.Attachments[0].ID != existing.ID {
				t.Fatal("existing or template files lost", first, err)
			}
			var adoptedIDs []int
			for _, f := range first.Attachments {
				adoptedIDs = append(adoptedIDs, f.ID)
			}
			repeated, err := m.Apply(a, template.UUID, ApplyInput{DraftType: mode, AttachmentIDs: adoptedIDs}, 100)
			if err != nil || len(repeated.Attachments) != 7 {
				t.Fatal("duplicate selection", repeated, err)
			}
			for i, f := range repeated.Attachments {
				if f.ID != first.Attachments[i].ID {
					t.Fatal("adoption replaced an existing reference")
				}
			}
		})
	}
}
