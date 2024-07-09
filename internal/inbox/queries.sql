-- name: get-active-inboxes
SELECT * from inboxes where enabled is TRUE;

-- name: get-all-inboxes
SELECT * from inboxes;

-- name: insert-inbox
INSERT INTO inboxes
(enabled, channel, config, "name", "from", assign_to_team)
VALUES($1, $2, $3, $4, $5, $6);