-- name: get-view
SELECT id, created_at, updated_at, name, filters, inbox_type, user_id
FROM views WHERE id = $1;

-- name: get-user-views
SELECT id, created_at, updated_at, name, filters, inbox_type, user_id
FROM views WHERE user_id = $1;

-- name: insert-view
INSERT INTO views (name, filters, inbox_type, user_id)
VALUES ($1, $2, $3, $4);

-- name: delete-view
DELETE FROM views
WHERE id = $1;

-- name: update-view
UPDATE views
SET name = $2, filters = $3, inbox_type = $4, updated_at = NOW()
WHERE id = $1
