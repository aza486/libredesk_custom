package dbutil

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	goyesqlx "github.com/knadh/goyesql/v2/sqlx"
	"github.com/lib/pq"
)

var (
	ErrUniqueViolation = errors.New("unique constraint violation")
	ErrEmailExists     = errors.New("email already exists")
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

// HandlePGError checks for common Postgres errors like unique constraint violations.
func HandlePGError(err error) error {
	if pgErr, ok := err.(*pq.Error); ok {
		switch pgErr.Code {
		case "23505":
			// Unique violation
			switch pgErr.Constraint {
			case "users_email_unique":
				return fmt.Errorf("%w: %s", ErrEmailExists, pgErr.Detail)
			default:
				return fmt.Errorf("%w: %s", ErrUniqueViolation, pgErr.Detail)
			}
		}
	}
	return err
}
