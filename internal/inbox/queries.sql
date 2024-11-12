-- name: get-active-inboxes
SELECT * from inboxes where disabled is NOT TRUE and deleted_at is NULL;

-- name: get-all-inboxes
SELECT id, name, channel, disabled, updated_at from inboxes where deleted_at is NULL;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from")
VALUES($1, $2, $3, $4);

-- name: get-by-id
SELECT * from inboxes where id = $1 and deleted_at is NULL;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5, updated_at = now()
where id = $1 and deleted_at is NULL;

-- name: soft-delete
UPDATE inboxes set deleted_at = now(), config = '{}' where id = $1 and deleted_at is NULL;

-- name: toggle
UPDATE inboxes 
SET disabled = NOT disabled, updated_at = NOW() 
WHERE id = $1;