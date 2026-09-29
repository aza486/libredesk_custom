package notifier

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestPersonalNotificationVisibilityAndBadges(t *testing.T) {
	db := testutil.NewPersonalDB(t, "personal_notification")
	db.MustExec(`INSERT INTO conversations(id,contact_id,inbox_id,status_id,custom_attributes) VALUES(101,105,101,101,'{"access_mode":"personal","visible_users":[101]}');
 INSERT INTO user_roles(user_id,role_id) SELECT 104,id FROM roles WHERE name='Admin';
 INSERT INTO user_notifications(user_id,notification_type,title,conversation_id) SELECT id,'assignment','Personal subject',101 FROM users WHERE id BETWEEN 101 AND 104;`)
	lo := logf.New(logf.Opts{})
	m, err := NewUserNotificationManager(UserNotificationOpts{DB: db, Lo: &lo, I18n: testutil.NewI18n(t)})
	if err != nil {
		t.Fatal(err)
	}
	dispatcher := NewDispatcher(DispatcherOpts{InApp: m, Lo: &lo})
	n := Notification{ConversationID: null.IntFrom(101), Type: models.NotificationTypeAssignment}
	for _, id := range []int{101, 102, 103, 104} {
		want := id == 101 || id == 104
		if got := dispatcher.canReceive(id, n); got != want {
			t.Fatalf("dispatch %d: %v", id, got)
		}
		notifications, err := m.GetAll(id, 20, 0)
		if err != nil {
			t.Fatal(err)
		}
		if (len(notifications) > 0) != want {
			t.Fatalf("notifications %d: %v", id, notifications)
		}
		stats, err := m.GetStats(id)
		if err != nil {
			t.Fatal(err)
		}
		if (stats.UnreadCount > 0) != want || (stats.TotalCount > 0) != want {
			t.Fatalf("stats %d: %+v", id, stats)
		}
	}
	db.MustExec(`UPDATE conversations SET custom_attributes=jsonb_set(custom_attributes,'{visible_users}','[101,102]') WHERE id=101`)
	if !dispatcher.canReceive(102, n) {
		t.Fatal("sharing did not enable notification")
	}
	db.MustExec(`UPDATE conversations SET custom_attributes=jsonb_set(custom_attributes,'{visible_users}','[101]') WHERE id=101`)
	if dispatcher.canReceive(102, n) {
		t.Fatal("revoked recipient still receives notification")
	}
}
