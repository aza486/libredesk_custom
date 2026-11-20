// Package tag handles the management of tags.
package tag

import (
	"database/sql"
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/tag/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	db   *sqlx.DB
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllTags *sqlx.Stmt `query:"get-all-tags"`
	InsertTag  *sqlx.Stmt `query:"insert-tag"`
	DeleteTag  *sqlx.Stmt `query:"delete-tag"`
	UpdateTag  *sqlx.Stmt `query:"update-tag"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		db:   opts.DB,
		q:    q,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// GetAll retrieves all tags.
func (t *Manager) GetAll() ([]models.Tag, error) {
	var tags = make([]models.Tag, 0)
	if err := t.q.GetAllTags.Select(&tags); err != nil {
		t.lo.Error("error fetching tags", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	filtered := make([]models.Tag, 0, len(tags))

	for _, tag := range tags {
		if IsClassificationTag(tag.ID, tag.Name) {
			continue
		}

		filtered = append(filtered, tag)
	}

	return filtered, nil
}

// Create creates a new tag.
func (t *Manager) Create(name string) (models.Tag, error) {
	var tag models.Tag
	if IsReservedDefinition(0, name) {
		return tag, envelope.NewError(envelope.InputError, "System tag names are reserved.", nil)
	}
	if err := t.q.InsertTag.Get(&tag, name); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return tag, envelope.NewError(envelope.ConflictError, t.i18n.T("errors.alreadyExistsTag"), nil)
		}
		t.lo.Error("error inserting tag", "error", err)
		return tag, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tag, nil
}

// Delete deletes a tag by ID.
func (t *Manager) Delete(id int) error {
	tx, err := t.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`LOCK TABLE tags IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	var name string
	if err := tx.Get(&name, `SELECT name FROM tags WHERE id=$1 FOR UPDATE`, id); err != nil && err != sql.ErrNoRows {
		return err
	}
	if IsReservedDefinition(id, name) {
		return envelope.NewError(
			envelope.InputError,
			"System tags cannot be deleted.",
			nil,
		)
	}
	if _, err := tx.Stmtx(t.q.DeleteTag).Exec(id); err != nil {
		t.lo.Error("error deleting tag", "error", err)
		return envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tx.Commit()
}

// Update updates a tag by id.
func (t *Manager) Update(id int, name string) (models.Tag, error) {
	tx, err := t.db.Beginx()
	if err != nil {
		return models.Tag{}, err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`LOCK TABLE tags IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return models.Tag{}, err
	}
	var existing string
	if err := tx.Get(&existing, `SELECT name FROM tags WHERE id=$1 FOR UPDATE`, id); err != nil && err != sql.ErrNoRows {
		return models.Tag{}, err
	}
	if IsReservedDefinition(id, existing) || IsReservedDefinition(0, name) {
		return models.Tag{},
			envelope.NewError(
				envelope.InputError,
				"System tags cannot be modified.",
				nil,
			)
	}
	var tag models.Tag
	if err := tx.Stmtx(t.q.UpdateTag).Get(&tag, id, name); err != nil {
		t.lo.Error("error updating tag", "error", err)
		return tag, envelope.NewError(envelope.GeneralError, t.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return tag, tx.Commit()
}
