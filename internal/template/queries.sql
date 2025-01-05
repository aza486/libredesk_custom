-- name: insert
INSERT INTO templates ("name", body, is_default, subject, type)
VALUES ($1, $2, $3, $4, $5);

-- name: update
WITH u AS (
    UPDATE templates
    SET 
        name = CASE WHEN $6::template_type = 'email_outgoing' THEN $2 ELSE name END,
        body = $3,
        is_default = $4,
        subject = $5,
        type = $6::template_type,
        updated_at = NOW()
    WHERE id = $1
    RETURNING id
)
UPDATE templates
SET is_default = FALSE
WHERE id != $1 AND $4 = TRUE;

-- name: get-default
SELECT id, name, body, subject FROM templates WHERE is_default is TRUE;

-- name: get-all
SELECT id, name, is_default, updated_at FROM templates WHERE type = $1 ORDER BY updated_at DESC;

-- name: get-template
SELECT id, name, body, subject, is_default, type FROM templates WHERE id = $1;

-- name: delete
DELETE FROM templates WHERE id = $1;

-- name: get-by-name
SELECT id, name, body, subject, is_default, type FROM templates WHERE name = $1;

-- name: is-builtin
SELECT EXISTS(SELECT 1 FROM templates WHERE id = $1 AND is_builtin is TRUE);