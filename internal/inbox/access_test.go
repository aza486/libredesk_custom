package inbox

import (
	"errors"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestPersonalInboxValidationAndModeLock(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_inbox")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, testutil.NewI18n(t), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []imodels.Inbox{
		{AccessMode: "invalid"}, {AccessMode: "public", OwnerUserID: null.IntFrom(101)},
		{AccessMode: "personal", Channel: "email"}, {AccessMode: "personal", Channel: "email", OwnerUserID: null.IntFrom(105)},
		{AccessMode: "personal", Channel: "livechat", OwnerUserID: null.IntFrom(101)},
	} {
		if err := m.ValidateAccess(in); err == nil {
			t.Errorf("accepted invalid inbox %+v", in)
		}
	}
	db.MustExec(`UPDATE users SET enabled=false WHERE id=102`)
	if err := m.ValidateAccess(imodels.Inbox{AccessMode: "personal", Channel: "email", OwnerUserID: null.IntFrom(102)}); err == nil {
		t.Fatal("inactive owner accepted")
	}
	if _, err := db.Exec(`UPDATE inboxes SET access_mode='personal',owner_user_id=101 WHERE id=102`); err != nil {
		t.Fatal("empty mode switch", err)
	}
	db.MustExec(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,101,101)`)
	_, err = db.Exec(`UPDATE inboxes SET access_mode='public',owner_user_id=NULL WHERE id=101`)
	var env envelope.Error
	if !errors.As(accessConstraintError(err), &env) || env.ErrorType != envelope.ConflictError {
		t.Fatalf("mode lock: %v", err)
	}
	record, err := m.GetDBRecord(101)
	if err != nil {
		t.Fatal(err)
	}
	record.Config = []byte(`{"imap":[{}],"smtp":[{}]}`)
	record.AccessMode = "public"
	record.OwnerUserID = null.Int{}
	_, err = m.Update(101, record)
	if !errors.As(err, &env) || env.Code != 409 {
		t.Fatalf("API mode lock must be HTTP 409: %v", err)
	}
	for _, query := range []string{`UPDATE users SET deleted_at=now() WHERE id=101`, `UPDATE users SET enabled=false WHERE id=101`, `DELETE FROM users WHERE id=101`} {
		if _, err := db.Exec(query); err == nil {
			t.Fatalf("owner protection missing: %s", query)
		}
	}
	db.MustExec(`UPDATE users SET enabled=true WHERE id=102`)
	if _, err := db.Exec(`UPDATE inboxes SET owner_user_id=102 WHERE id=101`); err != nil {
		t.Fatal("owner change", err)
	}
}

func TestPersonalInboxOwnerMigrationBackfillsLegacyOwner(t *testing.T) {
	db := testutil.NewDB(t, "personal_owner_backfill")
	if err := migrations.V2_12_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO users(id,type,email,first_name) VALUES(101,'agent','owner@example.com','Owner')`)
	if _, err := db.Exec(`INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id) VALUES(101,'Personal','email','personal',101)`); err != nil {
		t.Fatal(err)
	}
	if err := migrations.V2_13_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := migrations.V2_13_0(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	var ownerIDs []int64
	if err := db.Select(&ownerIDs, `SELECT user_id FROM personal_inbox_owners WHERE inbox_id=101`); err != nil {
		t.Fatal(err)
	}
	if len(ownerIDs) != 1 || ownerIDs[0] != 101 {
		t.Fatalf("backfilled owner IDs: %v", ownerIDs)
	}
}

func TestGetOwnPersonalInboxes(t *testing.T) {
	db := testutil.NewPersonalDB(t, "own_personal_inboxes")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, testutil.NewI18n(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id) VALUES(103,'Second','email','personal',101)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id,enabled,deleted_at) VALUES(104,'Deleted','email','personal',101,false,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id,enabled) VALUES(105,'Disabled','email','personal',101,false)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO personal_inbox_owners(inbox_id,user_id) VALUES(103,102)`); err != nil {
		t.Fatal(err)
	}
	ownerInboxes, err := m.GetOwnPersonalInboxes(101)
	if err != nil {
		t.Fatalf("owner inboxes: %v", err)
	}
	if len(ownerInboxes) != 2 {
		t.Fatalf("owner inboxes: got %d, want 2: %#v", len(ownerInboxes), ownerInboxes)
	}
	for _, inbox := range ownerInboxes {
		if inbox.Name == "Deleted" || inbox.Name == "Disabled" {
			t.Fatalf("deleted or disabled inbox leaked: %#v", ownerInboxes)
		}
	}
	otherInboxes, err := m.GetOwnPersonalInboxes(102)
	if err != nil || len(otherInboxes) != 1 || otherInboxes[0].Name != "Second" {
		t.Fatalf("other user inboxes: %v %v", otherInboxes, err)
	}
}

func TestUpdatePersonalInboxOwners(t *testing.T) {
	db := testutil.NewPersonalDB(t, "update_personal_inbox_owners")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, testutil.NewI18n(t), "")
	if err != nil {
		t.Fatal(err)
	}
	inbox, err := m.GetDBRecord(101)
	if err != nil {
		t.Fatal(err)
	}
	inbox.Config = []byte(`{"imap":[{}],"smtp":[{}]}`)
	inbox.OwnerUserIDs = []int64{101, 102}
	if _, err := m.Update(101, inbox); err != nil {
		t.Fatal(err)
	}
	for _, userID := range []int{101, 102} {
		owned, err := m.GetOwnPersonalInboxes(userID)
		if err != nil || len(owned) != 1 || owned[0].ID != 101 {
			t.Fatalf("owner %d inboxes: %v %v", userID, owned, err)
		}
	}
	if _, err := db.Exec(`UPDATE users SET enabled=false WHERE id=102`); err == nil {
		t.Fatal("disabling a secondary owner was allowed")
	}
	if _, err := m.GetOwnPersonalInboxes(103); err != nil {
		t.Fatal(err)
	} else if owned, _ := m.GetOwnPersonalInboxes(103); len(owned) != 0 {
		t.Fatalf("unassigned user inboxes: %v", owned)
	}

	inbox.OwnerUserIDs = []int64{101}
	if _, err := m.Update(101, inbox); err != nil {
		t.Fatal(err)
	}
	if owned, err := m.GetOwnPersonalInboxes(102); err != nil || len(owned) != 0 {
		t.Fatalf("removed owner inboxes: %v %v", owned, err)
	}
	if _, err := db.Exec(`DELETE FROM personal_inbox_owners WHERE inbox_id=101 AND user_id=101`); err == nil {
		t.Fatal("removing the final owner was allowed")
	}
	if _, err := m.GetDBRecord(101); err != nil {
		t.Fatalf("inbox removed with owner: %v", err)
	}
}

func TestPersonalInboxCoOwnerCanManageOwners(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_inbox_co_owner_manage")
	lo := logf.New(logf.Opts{})
	m, err := New(&lo, db, testutil.NewI18n(t), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.UpdatePersonalInboxOwners(101, 101, []int64{101, 102}, false); err != nil {
		t.Fatal(err)
	}
	if err := m.UpdatePersonalInboxOwners(101, 102, []int64{102}, false); err != nil {
		t.Fatalf("secondary owner could not remove original owner: %v", err)
	}
	if err := m.UpdatePersonalInboxOwners(101, 103, []int64{102, 103}, false); err == nil {
		t.Fatal("non-owner managed personal inbox owners")
	}
	if err := m.UpdatePersonalInboxOwners(101, 102, nil, false); err == nil {
		t.Fatal("last owner was removed")
	}
	ownerInboxes, err := m.GetOwnPersonalInboxes(102)
	if err != nil || len(ownerInboxes) != 1 || len(ownerInboxes[0].OwnerUserIDs) != 1 || ownerInboxes[0].OwnerUserIDs[0] != 102 {
		t.Fatalf("remaining owner data: %#v, %v", ownerInboxes, err)
	}
}

func TestModeChangeWaitsForConversationCreation(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_mode_race")
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`SELECT id FROM inboxes WHERE id=102 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := db.Exec(`UPDATE inboxes SET access_mode='personal',owner_user_id=101 WHERE id=102`)
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("mode change did not wait: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if _, err := tx.Exec(`INSERT INTO conversations(contact_id,inbox_id,status_id) VALUES(105,102,101)`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if accessConstraintError(err) == nil {
			t.Fatalf("racing mode switch allowed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("mode switch stuck")
	}
}
