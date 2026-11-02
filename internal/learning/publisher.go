package learning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"time"
)

// A lease and revision fence prevent stale embedding results from being committed.
func (m *Manager) PublishOne(ctx context.Context) (resultErr error) {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if m.Embed == nil {
		return ErrUnavailable
	}
	tx, err := m.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id string
	var raw, checkRaw []byte
	var revision int64
	err = tx.QueryRowContext(ctx, `SELECT r.id::text,to_jsonb(r)-'embedding',v.checks,v.revision FROM rag_items r JOIN learning_reviews v ON v.item_id=r.id WHERE v.decision='approved' AND (v.publication IN ('unpublished','failed') OR (v.publication='publishing' AND v.started_at<now()-interval '2 minutes')) AND v.retry_at<=now() AND NOT r.active ORDER BY v.updated_at FOR UPDATE OF r,v SKIP LOCKED LIMIT 1`).Scan(&id, &raw, &checkRaw, &revision)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	content, err := loadLockedContent(raw)
	if err != nil {
		return err
	}
	var checks Checks
	if err = json.Unmarshal(checkRaw, &checks); err != nil {
		return err
	}
	lease := uuid.NewString()
	_, err = tx.ExecContext(ctx, `UPDATE learning_reviews SET publication='publishing',lease=$2::uuid,started_at=now(),attempts=attempts+1 WHERE item_id=$1::uuid`, id, lease)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	defer func() {
		if resultErr == nil {
			return
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_, _ = m.DB.ExecContext(cleanup, `UPDATE learning_reviews SET publication='failed',lease=NULL,last_error=$3,retry_at=now()+interval '1 minute'*LEAST(30,attempts),updated_at=now() WHERE item_id=$1::uuid AND lease=$2::uuid`, id, lease, errorLabel(resultErr))
	}()
	err = Validate(content, checks, true)
	var vector []float64
	if err == nil {
		vector, err = m.Embed(ctx, content.Retrieval)
		if err == nil && !ValidVector(vector) {
			err = ErrInput
		}
	}
	if err != nil {
		_, e := m.DB.ExecContext(ctx, `UPDATE learning_reviews SET publication='failed',lease=NULL,last_error=$3,retry_at=now()+interval '1 minute'*LEAST(30,attempts),updated_at=now() WHERE item_id=$1::uuid AND lease=$2::uuid`, id, lease, errorLabel(err))
		if e != nil {
			return e
		}
		return err
	}
	final, err := m.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer final.Rollback()
	if _, err = final.ExecContext(ctx, `SELECT set_config('libredesk.review_writer','true',true)`); err != nil {
		return err
	}
	var currentLease string
	var currentRevision int64
	var actor string
	err = final.QueryRowContext(ctx, `SELECT v.lease::text,v.revision,v.reviewer FROM rag_items r JOIN learning_reviews v ON v.item_id=r.id WHERE r.id=$1::uuid AND v.decision='approved' AND v.publication='publishing' FOR UPDATE OF r,v`, id).Scan(&currentLease, &currentRevision, &actor)
	if err != nil {
		return err
	}
	if currentLease != lease || currentRevision != revision {
		return ErrConflict
	}
	_, err = final.ExecContext(ctx, `UPDATE rag_items SET embedding=$2::vector,approved=true,active=true,quality_status='approved',learning_data=learning_data||jsonb_build_object('publish_state','published','approval_method','human_review_v1','embedding_model','text-embedding-3-small','verification_status',CASE WHEN source_type='employee_fact' THEN 'human_verified' ELSE COALESCE(learning_data->>'verification_status','not_applicable') END,'factual_use_approved',source_type='employee_fact','published_at',now()::text),updated_at=now() WHERE id=$1::uuid`, id, vectorLiteral(vector))
	if err != nil {
		return err
	}
	_, err = final.ExecContext(ctx, `UPDATE learning_reviews SET publication='published',revision=revision+1,lease=NULL,last_error=NULL,updated_at=now() WHERE item_id=$1::uuid`, id)
	if err != nil {
		return err
	}
	if err = audit(ctx, final, id, actor, "publish", "Human review and 1536-dimensional embedding confirmed"); err != nil {
		return err
	}
	return final.Commit()
}
