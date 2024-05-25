package tags

import (
	"embed"
	"fmt"
	"time"

	"github.com/abhinavxd/artemis/internal/utils"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

type Tags struct {
	q  queries
	lo *logf.Logger
}

type Tag struct {
	ID        int64     `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Name      string    `db:"name" json:"name"`
}

type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

type queries struct {
	GetAllTags    *sqlx.Stmt `query:"get-all-tags"`
	InsertConvTag *sqlx.Stmt `query:"insert-conversation-tag"`
	DeleteConvTags *sqlx.Stmt `query:"delete-conversation-tags"`
}

func New(opts Opts) (*Tags, error) {
	var q queries

	if err := utils.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}

	return &Tags{
		q:  q,
		lo: opts.Lo,
	}, nil
}

func (t *Tags) GetAllTags() ([]Tag, error) {
	var tt []Tag
	if err := t.q.GetAllTags.Select(&tt); err != nil {
		t.lo.Error("fetching tags", "error", err)
		return tt, fmt.Errorf("error fetching tags")
	}
	return tt, nil
}

func (t *Tags) UpsertConvTag(convUUID string, tagIDs []int) error {
	// First delete tags that've been removed.
	if _, err := t.q.DeleteConvTags.Exec(convUUID, pq.Array(tagIDs)); err != nil {
		t.lo.Error("inserting tag for conversation", "error", err, "converastion_uuid", convUUID, "tag_id", tagIDs)
		return fmt.Errorf("error updating tags")
	}

	// Add new tags one by one.
	for _, tagID := range tagIDs {
		if _, err := t.q.InsertConvTag.Exec(convUUID, tagID); err != nil {
			t.lo.Error("inserting tag for conversation", "error", err, "converastion_uuid", convUUID, "tag_id", tagID)
			return fmt.Errorf("error updating tags")
		}
	}
	return nil
}
