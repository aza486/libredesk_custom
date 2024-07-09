-- name: get-users
SELECT first_name, last_name, uuid, email, disabled from users;

-- name: get-email
SELECT email from users where CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END; 

-- name: get-user-by-email
select id, email, password, avatar_url, first_name, last_name, uuid from users where email = $1;

-- name: get-user
select id, email, avatar_url, first_name, last_name, uuid from users where CASE WHEN $1 > 0 THEN id = $1 ELSE uuid = $2 END;

-- name: set-user-password
update users set password = $1 where id = $2;

-- name: create-user
INSERT INTO users
(email, first_name, last_name, "password", team, avatar_url)
VALUES($1, $2, $3, $4, $5, $6);