-- name: get-all-statuses
select id, 
    created_at,
    name
from status;

-- name: insert-status
INSERT into status(name) values ($1);

-- name: delete-status
DELETE from status where id = $1;

-- name: update-status
UPDATE status set name = $2 where id = $1;