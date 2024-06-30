-- name: insert-template
INSERT INTO templates
("name", subject, body, is_default)
VALUES($1, $2, $3, $4);

-- name: get-template
select id, name, subject, body from templates where name = $1;

-- name: get-default-template
select id, name, subject, body from templates where is_default is true;