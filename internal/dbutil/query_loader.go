// Package dbutil provides utility functions for database operations.
package dbutil

import (
	"errors"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
)

var (
	// ErrUniqueViolation indicates a unique constraint violation.
	ErrUniqueViolation = errors.New("unique constraint violation")
	// ErrEmailExists indicates that an email already exists.
	ErrEmailExists = errors.New("email already exists")
)

// ScanSQLFile scans a goyesql .sql file from the given fs and prepares the queries in the given struct.
func ScanSQLFile(path string, o interface{}, db *sqlx.DB, f fs.FS) error {
	// Read the SQL file from the embedded filesystem.
	b, err := fs.ReadFile(f, path)
	if err != nil {
		return err
	}

	// Parse the SQL file.
	q, err := goyesql.ParseBytes(b)
	if err != nil {
		return err
	}

	// Scan the parsed queries into the provided struct and prepare them.
	if err := goyesqlx.ScanToStruct(o, q, db.Unsafe()); err != nil {
		return err
	}
	return nil
}
