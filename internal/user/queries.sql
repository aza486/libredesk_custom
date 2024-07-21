-- name: get-users
SELECT u.id, u.first_name, u.last_name, u.email, u.disabled, u.team_id, t.name as team_name
FROM users u
LEFT JOIN teams t ON t.id = u.team_id
ORDER BY u.updated_at DESC;

-- name: get-email
SELECT email from users where id = $1;

-- name: get-user-by-email
SELECT u.id, u.email, u.password, u.avatar_url, u.first_name, u.last_name, u.team_id, r.permissions
FROM users u
JOIN roles r ON r.name = ANY(u.roles)
WHERE u.email = $1;

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
(email, first_name, last_name, "password", team_id, avatar_url, roles)
VALUES($1, $2, $3, $4, $5, $6, $7);

-- name: update-user
UPDATE users
set first_name = $2, last_name = $3, email = $4, team_id = $5, roles = $6, updated_at = now()
where id = $1
