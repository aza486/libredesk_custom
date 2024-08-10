-- name: insert
INSERT INTO templates ("name", body, is_default)
VALUES ($1, $2, $3);

-- name: update
WITH u AS (
    UPDATE templates 
    SET name = $2, body = $3, is_default = $4, updated_at = now()  
    WHERE id = $1
)
UPDATE templates 
SET is_default = false 
WHERE id != $1 AND $4 = true;

-- name: get-default
select id, name, body from templates where is_default is true;

-- name: get-all
select * from templates order by updated_at desc;

-- name: get-template
SELECT * from templates where id = $1;