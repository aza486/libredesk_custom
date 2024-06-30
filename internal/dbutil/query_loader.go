package dbutil

import (
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
)

// ScanSQLFile scans a goyesql .sql file from the given fs to the given struct.
func ScanSQLFile(path string, o interface{}, db *sqlx.DB, f fs.FS) error {
	b, err := fs.ReadFile(f, path)
	if err != nil {
		return err
	}

	q, err := goyesql.ParseBytes(b)
	if err != nil {
		return err
	}
	// Prepare queries.
	if err := goyesqlx.ScanToStruct(o, q, db.Unsafe()); err != nil {
		return err
	}
	return nil
}
