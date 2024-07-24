// Package tag handles the management of tags in the system.
package tag

import (
	"embed"
	"fmt"

	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/tag/models"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager handles tag-related operations.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllTags *sqlx.Stmt `query:"get-all-tags"`
	InsertTag  *sqlx.Stmt `query:"insert-tag"`
	DeleteTag  *sqlx.Stmt `query:"delete-tag"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

// GetAll retrieves all tags.
func (t *Manager) GetAll() ([]models.Tag, error) {
	var tags []models.Tag
	if err := t.q.GetAllTags.Select(&tags); err != nil {
		t.lo.Error("fetching tags", "error", err)
		return tags, fmt.Errorf("error fetching tags")
	}
	return tags, nil
}

// AddTag adds a new tag.
func (t *Manager) AddTag(name string) error {
	if _, err := t.q.InsertTag.Exec(name); err != nil {
		t.lo.Error("inserting tag", "error", err)
		return fmt.Errorf("error inserting tag")
	}
	return nil
}

// DeleteTag deletes a tag by ID.
func (t *Manager) DeleteTag(id int) error {
	if _, err := t.q.DeleteTag.Exec(id); err != nil {
		t.lo.Error("deleting tag", "error", err)
		return fmt.Errorf("error deleting tag")
	}
	return nil
}
