-- name: get-all-tags
select id,
    created_at,
    name
from tags;

-- name: insert-tag
INSERT into tags (name) values ($1);

-- name: delete-tag
DELETE from tags where id = $1;