-- name: insert
INSERT INTO templates ("name", body, is_default)
VALUES ($1, $2, $3);

-- name: update
UPDATE templates set name = $2, body = $3, is_default = $4  where id = $1;

-- name: get-default
select id, name, body from templates where is_default is true;

-- name: get-all
select * from templates;

-- name: get-template
SELECT * from templates where id = $1;