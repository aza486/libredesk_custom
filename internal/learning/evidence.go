package learning

import (
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
)

// Bound the *complete source*, separately from the short RAG candidate. PostgreSQL
// checks octet_length before returning the value, so oversized bodies are not loaded.
const maxEvidenceSourceBytes = 1 << 20

func verifyEvidence(ctx context.Context, tx *sqlx.Tx, id string, c Content) error {
	if c.Source != "human_answer" && c.Source != "feedback_reason" {
		return invalid("evidence_source_invalid")
	}
	var size sql.NullInt64
	var source sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT octet_length(s.body),CASE WHEN octet_length(s.body)<=$3 THEN s.body END
 FROM public.rag_items r JOIN public.ai_feedback f ON f.id=r.feedback_id
 CROSS JOIN LATERAL (SELECT CASE WHEN $2='human_answer' THEN f.human_answer ELSE f.feedback_reason END AS body) s
 WHERE r.id=$1::uuid`, id, c.Source, maxEvidenceSourceBytes).Scan(&size, &source)
	if err == sql.ErrNoRows {
		return invalid("evidence_source_missing")
	}
	if err != nil {
		return err
	}
	if size.Valid && size.Int64 > maxEvidenceSourceBytes {
		return invalid("evidence_source_too_large")
	}
	if !source.Valid {
		return invalid("evidence_source_missing")
	}
	return evidenceInSource(source.String, c.Source, c.Evidence)
}

func evidenceInSource(source, kind, quote string) error {
	if kind != "human_answer" && kind != "feedback_reason" {
		return invalid("evidence_source_invalid")
	}
	if len(source) > maxEvidenceSourceBytes {
		return invalid("evidence_source_too_large")
	}
	if !utf8.ValidString(source) || strings.ContainsRune(source, 0) || !text(quote, 6000) {
		return invalid("evidence_invalid_text")
	}
	// Raw source may contain signatures/PII. Never validate or copy it as reusable
	// knowledge; only the separately validated candidate can enter RAG/embedding.
	if kind == "human_answer" {
		source = stringutil.HTML2Text(source)
	}
	if len(source) > 2*maxEvidenceSourceBytes {
		return invalid("evidence_source_too_large")
	}
	source = strings.Join(strings.Fields(source), " ")
	quote = strings.Join(strings.Fields(quote), " ")
	if quote == "" || !strings.Contains(source, quote) {
		return invalid("evidence_quote_not_found")
	}
	return nil
}
