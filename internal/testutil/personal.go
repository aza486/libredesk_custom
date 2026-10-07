package testutil

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/migrations"
	"github.com/jmoiron/sqlx"
)

// NewPersonalDB uses a disposable test database, never the application's DB.
func NewPersonalDB(t *testing.T, name string) *sqlx.DB {
	t.Helper()
	db := NewDB(t, name)
	for _, migrate := range []func(*sqlx.DB) error{
		func(db *sqlx.DB) error { return migrations.V2_9_0(db, nil, nil) },
		func(db *sqlx.DB) error { return migrations.V2_11_0(db, nil, nil) },
		func(db *sqlx.DB) error { return migrations.V2_12_0(db, nil, nil) },
		func(db *sqlx.DB) error { return migrations.V2_13_0(db, nil, nil) },
	} {
		if err := migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	db.MustExec(`INSERT INTO users (id,type,email,first_name) VALUES
 (101,'agent','owner@example.com','Owner'), (102,'agent','other@example.com','Other'),
 (103,'agent','support@example.com','Support'), (104,'agent','admin@example.com','Admin'),
 (105,'contact','contact@example.com','Contact');
 UPDATE users SET last_name='';
 UPDATE conversation_statuses SET id=101 WHERE name='Open';
 UPDATE conversation_priorities SET id=101 WHERE name='High';
 INSERT INTO tags(id,name) VALUES(13,'🦽Kundenticket'),(14,'🧷Service-Mail');
 INSERT INTO inboxes(id,name,channel,access_mode,owner_user_id) VALUES
 (101,'Personal','email','personal',101),(102,'Public','email','public',NULL);
 UPDATE inboxes SET "from"='inbox@example.com';
 `)
	return db
}
