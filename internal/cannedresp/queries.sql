-- name: get-all
SELECT id, title, content, created_at, updated_at FROM canned_responses order by updated_at desc;

-- name: create
INSERT INTO canned_responses (title, content)
VALUES ($1, $2);

-- name: update
UPDATE canned_responses
SET title = $2, content = $3, updated_at = now() where id = $1;

-- name: delete
DELETE FROM canned_responses WHERE id = $1;