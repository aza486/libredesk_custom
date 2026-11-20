package tag

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// This is the existing database/API contract, not a list of protected tags.
// In particular, Spam's user-editing policy remains unchanged (F06).
var requiredSystemTags = [...]struct {
	ID   int
	Name string
}{
	{SystemTagInternal, "🏢Intern"},
	{SystemTagCustomer, SystemTagCustomerName},
	{SystemTagHuman, "😎Mensch erforderlich"},
	{12, "🗑Spam"},
	{SystemTagServiceMail, "🧷Service-Mail"},
}

// CheckSystemTags is a read-only preflight. Missing tags and conflicting
// identities are errors; it never repairs data or advances the ID sequence.
func CheckSystemTags(ctx context.Context, db *sqlx.DB) error {
	missing, err := inspectSystemTags(ctx, db)
	if err != nil {
		return err
	}
	var problems []error
	for _, id := range missing {
		for _, required := range requiredSystemTags {
			if required.ID == id {
				problems = append(problems, fmt.Errorf("missing system tag: ID %d must be %q", id, required.Name))
			}
		}
	}
	return errors.Join(problems...)
}

// EnsureSystemTags inserts only unambiguously missing canonical rows. A table
// lock serializes bootstraps and ordinary tag writes. All conflict checks happen
// before any insertion. No conversation associations, activities or webhooks are
// touched. The sequence repair uses transactional RESTART, never setval.
func EnsureSystemTags(ctx context.Context, db *sqlx.DB) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `LOCK TABLE tags IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("locking system tags: %w", err)
	}
	missing, err := inspectSystemTags(ctx, tx)
	if err != nil {
		return err
	}
	for _, required := range requiredSystemTags {
		for _, id := range missing {
			if required.ID == id {
				if _, err := tx.ExecContext(ctx, `INSERT INTO tags(id,name) VALUES($1,$2)`, id, required.Name); err != nil {
					return fmt.Errorf("creating system tag ID %d: %w", id, err)
				}
			}
		}
	}
	if err := advanceTagSequence(ctx, tx, len(missing) > 0); err != nil {
		return fmt.Errorf("checking tag ID sequence: %w", err)
	}
	return tx.Commit()
}

func inspectSystemTags(ctx context.Context, db sqlx.QueryerContext) ([]int, error) {
	var ids []int
	var names []string
	for _, required := range requiredSystemTags {
		ids = append(ids, required.ID)
		names = append(names, required.Name)
	}
	var rows []struct {
		ID   int
		Name string
	}
	if err := sqlx.SelectContext(ctx, db, &rows, `SELECT id,name FROM tags WHERE id=ANY($1) OR name=ANY($2) ORDER BY id`, pq.Array(ids), pq.Array(names)); err != nil {
		return nil, fmt.Errorf("reading system tags (install/upgrade the schema first): %w", err)
	}
	var missing []int
	var conflicts []error
	for _, required := range requiredSystemTags {
		found := false
		for _, row := range rows {
			if row.ID == required.ID {
				found = true
				if row.Name != required.Name {
					conflicts = append(conflicts, fmt.Errorf("system tag conflict: ID %d is %q, expected %q", row.ID, row.Name, required.Name))
				}
			}
			if row.Name == required.Name && row.ID != required.ID {
				conflicts = append(conflicts, fmt.Errorf("system tag conflict: %q has ID %d, expected %d", row.Name, row.ID, required.ID))
			}
		}
		if !found {
			missing = append(missing, required.ID)
		}
	}
	if err := errors.Join(conflicts...); err != nil {
		return nil, fmt.Errorf("%w; no automatic renaming or renumbering is permitted", err)
	}
	return missing, nil
}

func advanceTagSequence(ctx context.Context, tx *sqlx.Tx, inserted bool) error {
	var schema, name string
	var cacheSize int64
	if err := tx.QueryRowContext(ctx, `SELECT n.nspname,c.relname,s.seqcache FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_sequence s ON s.seqrelid=c.oid WHERE c.oid=pg_get_serial_sequence('tags','id')::regclass`).Scan(&schema, &name, &cacheSize); err != nil {
		return err
	}
	sequence := pq.QuoteIdentifier(schema) + "." + pq.QuoteIdentifier(name)
	var last, maxID int64
	var called bool
	if err := tx.QueryRowContext(ctx, `SELECT last_value,is_called FROM `+sequence).Scan(&last, &called); err != nil {
		return err
	}
	if err := tx.GetContext(ctx, &maxID, `SELECT COALESCE(max(id),0) FROM tags`); err != nil {
		return err
	}
	next := last
	if called {
		next++
	}
	if next <= maxID {
		next = maxID + 1
	}
	// RESTART also invalidates other sessions' cached IDs. Newly inserted
	// canonical IDs may overlap those caches even when last_value is ahead.
	// Never rewind past the sequence's current allocation boundary.
	if inserted || cacheSize > 1 || last < maxID || (last == maxID && !called) {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER SEQUENCE %s RESTART WITH %d`, sequence, next))
		return err
	}
	return nil
}
