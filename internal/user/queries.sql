-- name: get-users
SELECT u.id, u.first_name, u.last_name, u.email, u.disabled, u.team_id, t.name as team_name
FROM users u
LEFT JOIN teams t ON t.id = u.team_id
ORDER BY u.updated_at DESC;

-- name: get-email
SELECT email from users where id = $1;

-- name: get-user-by-email
select id, email, password, avatar_url, first_name, last_name, team_id from users where email = $1;

-- name: get-user
SELECT id, email, avatar_url, first_name, last_name, team_id 
FROM users 
WHERE 
  CASE 
    WHEN $1 > 0 THEN id = $1 
    ELSE uuid = $2 
  END;

-- name: set-user-password
update users set password = $1, updated_at = now() where id = $2;

-- name: create-user
INSERT INTO users
(email, first_name, last_name, "password", team_id, avatar_url)
VALUES($1, $2, $3, $4, $5, $6);

-- name: update-user
UPDATE users
set first_name = $2, last_name = $3, email = $4, team_id = $5, updated_at = now()
where id = $1
