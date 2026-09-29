package media

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

type personalTestStore struct{ Store }

func (personalTestStore) GetURL(name, disposition, filename string) string {
	return "https://storage.example/" + name
}
func (personalTestStore) Name() string { return "s3" }

func TestPersonalMediaNeverReturnsStorageBearerURL(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_media")
	db.MustExec(`INSERT INTO conversations(id,contact_id,inbox_id,status_id,custom_attributes) VALUES(101,105,101,101,'{"access_mode":"personal","visible_users":[101]}');
 INSERT INTO conversation_messages(id,conversation_id,type,status,sender_id,sender_type) VALUES(101,101,'incoming','received',105,'contact')`)
	var uuid string
	if err := db.QueryRow(`INSERT INTO media(store,filename,content_type,model_type,model_id) VALUES('s3','image.png','image/png','messages',101) RETURNING uuid`).Scan(&uuid); err != nil {
		t.Fatal(err)
	}
	lo := logf.New(logf.Opts{})
	m, err := New(Opts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t), Store: personalTestStore{}, RootURL: func() string { return "https://desk.example" }})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://desk.example/uploads/" + uuid
	if got := m.GetURL(uuid, "image/png", "image.png"); got != want {
		t.Fatal(got)
	}
	if got := m.GetSignedURL(uuid); got != want {
		t.Fatal(got)
	}
	if got := m.GetThumbnailURL(uuid); got != "https://desk.example/uploads/"+image.ThumbPrefix+uuid {
		t.Fatal(got)
	}
	if got := m.GetURLForDownload(uuid, "image.png"); got != want+"?download=1" {
		t.Fatal(got)
	}
	db.MustExec(`UPDATE conversations SET custom_attributes='{}' WHERE id=101`)
	if got := m.GetURL(uuid, "image/png", "image.png"); got != "https://storage.example/"+uuid {
		t.Fatal("public changed", got)
	}
}
