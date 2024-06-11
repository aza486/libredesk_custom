package tag

import (
	"embed"

	"github.com/abhinavxd/artemis/internal/dbutils"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q  queries
	lo *logf.Logger
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	AddTag     *sqlx.Stmt `query:"add-tag"`
	DeleteTags *sqlx.Stmt `query:"delete-tags"`
}

func New(opts Opts) (*Manager, error) {
	var q queries

	if err := dbutils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (t *Manager) AddTags(convUUID string, tagIDs []int) error {
	// Delete tags that have been removed.
	if _, err := t.q.DeleteTags.Exec(convUUID, pq.Array(tagIDs)); err != nil {
		t.lo.Error("inserting tag for conversation", "error", err, "converastion_uuid", convUUID, "tag_id", tagIDs)
	}

	// Add new tags one by one.
	for _, tagID := range tagIDs {
		if _, err := t.q.AddTag.Exec(convUUID, tagID); err != nil {
			t.lo.Error("inserting tag for conversation", "error", err, "converastion_uuid", convUUID, "tag_id", tagID)
		}
	}
	return nil
}
