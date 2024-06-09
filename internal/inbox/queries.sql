-- name: get-active-inboxes
SELECT * from inboxes where enabled is TRUE;