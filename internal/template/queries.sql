-- name: insert
INSERT INTO templates ("name", body, is_default)
VALUES ($1, $2, $3);

-- name: update
WITH u AS (
    UPDATE templates 
    SET name = $2, body = $3, is_default = $4, updated_at = NOW()  
    WHERE id = $1
)
UPDATE templates 
SET is_default = FALSE 
WHERE id != $1 AND $4 = TRUE;

-- name: get-default
SELECT id, name, body FROM templates WHERE is_default = TRUE;

-- name: get-all
SELECT * FROM templates ORDER BY updated_at DESC;

-- name: get-template
SELECT * FROM templates WHERE id = $1;

-- name: delete
DELETE FROM templates WHERE id = $1;
