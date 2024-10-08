-- name: get-users
SELECT u.id, u.updated_at, u.first_name, u.last_name, u.email, u.disabled
FROM users u where u.email != 'System'
ORDER BY u.updated_at DESC;

-- name: get-users-compact
SELECT u.id, u.first_name, u.last_name, u.disabled
FROM users u
ORDER BY u.updated_at DESC;

-- name: get-email
SELECT email from users where id = $1;

-- name: get-user-by-email
SELECT u.id, u.email, u.password, u.avatar_url, u.first_name, u.last_name, r.permissions
FROM users u
JOIN roles r ON r.name = ANY(u.roles)
WHERE u.email = $1;

-- name: get-user
SELECT
    u.id,
    u.email,
    u.avatar_url,
    u.first_name,
    u.last_name,
    u.roles,
    COALESCE(
        (SELECT json_agg(json_build_object('id', t.id, 'name', t.name))
         FROM team_members tm
         JOIN teams t ON tm.team_id = t.id
         WHERE tm.user_id = u.id),
        '[]'
    ) AS teams,
    COALESCE(
        ARRAY(
            SELECT DISTINCT unnest(r.permissions)
            FROM roles r
            WHERE r.name = ANY(u.roles)
        ),
        ARRAY[]::text[]
    ) AS permissions
FROM
    users u
WHERE
    u.id = $1;


-- name: set-user-password
update users set password = $1, updated_at = now() where id = $2;

-- name: create-user
INSERT INTO users
(email, first_name, last_name, "password", avatar_url, roles)
VALUES($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: update-user
UPDATE users
SET first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    email = COALESCE($4, email),
    roles = COALESCE($5, roles),
    avatar_url = COALESCE($6, avatar_url),
    password = COALESCE($7, password),
    updated_at = now()
WHERE id = $1

-- name: update-avatar
UPDATE users
SET avatar_url = $2 WHERE id = $1;

-- name: get-permissions
SELECT unnest(r.permissions)
FROM users u
JOIN roles r ON r.name = ANY(u.roles)
WHERE u.id = $1