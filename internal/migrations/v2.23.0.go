package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_23_0 extends existing folders and grants only service-address management.
func V2_23_0(db *sqlx.DB, _ stuffbin.FileSystem, _ *koanf.Koanf) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`ALTER TABLE sidebar_folders
 DROP CONSTRAINT IF EXISTS sidebar_folders_scope_type_check,
 DROP CONSTRAINT IF EXISTS sidebar_folders_scope_id_check,
 ADD CONSTRAINT sidebar_folders_scope_type_check CHECK(scope_type IN ('customer','redistributed','team','personal','my_inbox','service_mail')),
 ADD CONSTRAINT sidebar_folders_scope_id_check CHECK((scope_type IN ('customer','redistributed','service_mail') AND scope_id=0) OR (scope_type IN ('team','personal','my_inbox') AND scope_id>0));
 UPDATE roles SET permissions=array_append(permissions,'service_emails:manage')
 WHERE name IN ('Admin','Kundensupport') AND NOT ('service_emails:manage'=ANY(permissions));`)
	if err != nil {
		return err
	}
	return tx.Commit()
}
