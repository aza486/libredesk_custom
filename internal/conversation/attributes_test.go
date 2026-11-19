package conversation

import (
	"encoding/json"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestConversationSystemAttributesPreserveStoredJSON(t *testing.T) {
	m := routingFixture(t, "system_attributes_json")
	uuid := routingTicket(t, m, "Open", false)
	// A generic save must preserve even legacy/null values and integer precision.
	const stored = `{"private":true,"internal":false,"creator_id":9007199254740993,"visible_users":[101,103],"visibility_managers":null,"customer_visibility":false,"access_mode":"public","owner_user_id":null,"_spam_previous_state":{"status_id":101},"old":"remove"}`
	m.db.MustExec(`UPDATE conversations SET custom_attributes=$2 WHERE uuid=$1`, uuid, stored)
	var want map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stored), &want); err != nil {
		t.Fatal(err)
	}
	delete(want, "old")
	for _, input := range []map[string]any{{}, {"ordinary": "new", "private": false, "creator_id": 101, "visible_users": []int{102}, "_spam_previous_state": nil}} {
		if err := m.UpdateConversationCustomAttributes(uuid, input); err != nil {
			t.Fatal(err)
		}
		var raw []byte
		if err := m.db.Get(&raw, `SELECT custom_attributes FROM conversations WHERE uuid=$1`, uuid); err != nil {
			t.Fatal(err)
		}
		var got map[string]json.RawMessage
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		delete(got, "ordinary")
		for key, expected := range want {
			// jsonb spacing is canonicalized by PostgreSQL on both sides.
			var canonical string
			if err := m.db.Get(&canonical, `SELECT $1::jsonb::text`, string(expected)); err != nil {
				t.Fatal(err)
			}
			if string(got[key]) != canonical {
				t.Fatalf("%s changed: got %s want %s", key, got[key], canonical)
			}
		}
	}
}

func TestConversationSystemAttributesConcurrentVisibility(t *testing.T) {
	m := routingFixture(t, "system_attributes_concurrent")
	for _, tc := range []struct {
		name, attrs string
		inbox       int
		customer    bool
	}{
		{"private", `{"private":true,"internal":true,"creator_id":103,"visible_users":[101,103],"visibility_managers":[103],"customer_visibility":false}`, 102, false},
		{"customer", `{"creator_id":103,"visible_users":[101,103],"visibility_managers":[103],"customer_visibility":true}`, 102, true},
		{"personal", `{"access_mode":"personal","owner_user_id":101,"visible_users":[101,103],"visibility_managers":[101],"customer_visibility":false}`, 101, false},
		{"ordinary", `{"visible_users":[101,103]}`, 102, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			removeID := 101
			if tc.name == "personal" {
				removeID = 103
			}
			for range 4 {
				var uuid string
				if err := m.db.Get(&uuid, `INSERT INTO conversations(contact_id,inbox_id,status_id,custom_attributes) VALUES(105,$1,101,$2) RETURNING uuid`, tc.inbox, tc.attrs); err != nil {
					t.Fatal(err)
				}
				if tc.customer {
					m.db.MustExec(`INSERT INTO conversation_tags(conversation_id,tag_id) SELECT id,13 FROM conversations WHERE uuid=$1`, uuid)
				}
				// Make the ordinary save wait behind a real concurrent system change.
				blocker, err := m.db.Beginx()
				if err != nil {
					t.Fatal(err)
				}
				defer blocker.Rollback()
				if _, err = blocker.Exec(`UPDATE conversations SET custom_attributes=custom_attributes || '{"_spam_previous_state":{"status_id":101}}' WHERE uuid=$1`, uuid); err != nil {
					t.Fatal(err)
				}
				start := make(chan struct{})
				results := make(chan error, 3)
				var wg sync.WaitGroup
				for _, mutation := range []func() error{
					func() error {
						return m.UpdateConversationCustomAttributes(uuid, map[string]any{"ordinary": "saved", "visible_users": []int{999}, "private": false, "_spam_previous_state": nil})
					},
					func() error { return m.RemoveVisibleUser(uuid, removeID) },
					func() error { return m.AddVisibleUser(uuid, 102) },
				} {
					wg.Add(1)
					go func(fn func() error) { defer wg.Done(); <-start; results <- fn() }(mutation)
				}
				close(start)
				deadline := time.Now().Add(5 * time.Second)
				for {
					var waiting int
					if err = m.db.Get(&waiting, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'SELECT id FROM conversations WHERE uuid=%FOR UPDATE%'`); err != nil {
						t.Fatal(err)
					}
					if waiting >= 3 {
						break
					}
					if time.Now().After(deadline) {
						t.Fatal("mutations did not all wait on the conversation lock")
					}
					time.Sleep(5 * time.Millisecond)
				}
				if err = blocker.Commit(); err != nil {
					t.Fatal(err)
				}
				wg.Wait()
				close(results)
				for err := range results {
					if err != nil {
						t.Fatal(err)
					}
				}
				var raw []byte
				if err = m.db.Get(&raw, `SELECT custom_attributes FROM conversations WHERE uuid=$1`, uuid); err != nil {
					t.Fatal(err)
				}
				var got, want map[string]any
				json.Unmarshal(raw, &got)
				json.Unmarshal([]byte(tc.attrs), &want)
				visible := []any{float64(103), float64(102)}
				if tc.name == "personal" {
					visible[0] = float64(101)
				}
				actual := got["visible_users"].([]any)
				if len(actual) != len(visible) {
					t.Fatalf("lost visibility: %s", raw)
				}
				for _, id := range visible {
					if !slices.Contains(actual, id) {
						t.Fatalf("lost user %v: %s", id, raw)
					}
				}
				delete(got, "visible_users")
				delete(want, "visible_users")
				want["ordinary"] = "saved"
				want["_spam_previous_state"] = map[string]any{"status_id": float64(101)}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("lost concurrent state: got %s want %v", raw, want)
				}
			}
		})
	}
}
