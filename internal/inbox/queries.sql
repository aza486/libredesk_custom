-- name: get-active-inboxes
SELECT * from inboxes where disabled is NOT TRUE and soft_delete is false;

-- name: get-all-inboxes
SELECT * from inboxes where soft_delete is false;

-- name: insert-inbox
INSERT INTO inboxes
(channel, config, "name", "from", assign_to_team)
VALUES($1, $2, $3, $4, $5);

-- name: get-by-id
SELECT * from inboxes where id = $1 and soft_delete is false;

-- name: update
UPDATE inboxes
set channel = $2, config = $3, "name" = $4, "from" = $5
where id = $1 and soft_delete is false;

-- name: soft-delete
UPDATE inboxes set soft_delete = true where id = $1;

-- name: toggle
UPDATE inboxes 
SET disabled = NOT disabled, updated_at = NOW() 
WHERE id = $1;