-- name: get-users
SELECT u.id, u.updated_at, u.first_name, u.last_name, u.email, u.disabled
FROM users u
WHERE u.email != 'System' AND u.deleted_at IS NULL AND u.type = 'agent'
ORDER BY u.updated_at DESC;

-- name: soft-delete-user
UPDATE users
SET deleted_at = now()
WHERE id = $1 AND type = 'agent';

-- name: get-users-compact
SELECT u.id, u.first_name, u.last_name, u.disabled
FROM users u
WHERE u.email != 'System' AND u.deleted_at IS NULL AND u.type = 'agent'
ORDER BY u.updated_at DESC;

-- name: get-email
SELECT email
FROM users
WHERE id = $1 AND deleted_at IS NULL AND type = 'agent';

-- name: get-user-by-email
SELECT u.id, u.email, u.password, u.avatar_url, u.first_name, u.last_name, r.permissions
FROM users u
JOIN roles r ON r.name = ANY(u.roles)
WHERE u.email = $1 AND u.deleted_at IS NULL AND u.type = 'agent';

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
    u.id = $1 AND u.deleted_at IS NULL AND u.type = 'agent';

-- name: set-user-password
UPDATE users
SET password = $1, updated_at = now()
WHERE id = $2 AND type = 'agent';

-- name: update-user
UPDATE users
SET first_name = COALESCE($2, first_name),
    last_name = COALESCE($3, last_name),
    email = COALESCE($4, email),
    roles = COALESCE($5, roles),
    avatar_url = COALESCE($6, avatar_url),
    password = COALESCE($7, password),
    updated_at = now()
WHERE id = $1 AND type = 'agent';

-- name: update-avatar
UPDATE users
SET avatar_url = $2, updated_at = now()
WHERE id = $1 AND type = 'agent';

-- name: get-permissions
SELECT unnest(r.permissions)
FROM users u
JOIN roles r ON r.name = ANY(u.roles)
WHERE u.id = $1 AND u.type = 'agent';

-- name: set-reset-password-token
UPDATE users
SET reset_password_token = $2, reset_password_token_expiry = now() + interval '1 day'
WHERE id = $1 AND type = 'agent';

-- name: reset-password
UPDATE users
SET password = $1, reset_password_token = NULL, reset_password_token_expiry = NULL
WHERE reset_password_token = $2 AND reset_password_token_expiry > now() AND type = 'agent';

-- name: insert-agent
INSERT INTO users (email, type, first_name, last_name, "password", avatar_url, roles)
VALUES ($1, 'agent', $2, $3, $4, $5, $6)
ON CONFLICT (email) WHERE email IS NOT NULL 
DO UPDATE SET updated_at = now()
RETURNING id;

-- name: insert-contact
WITH contact AS (
    INSERT INTO users (email, type, first_name, last_name, "password", avatar_url, roles)
    VALUES ($1, 'contact', $2, $3, $4, $5, $6)
    ON CONFLICT (email)
    DO UPDATE SET updated_at = now()
    RETURNING id
)
INSERT INTO contact_channels (contact_id, inbox_id, identifier)
VALUES ((SELECT id FROM contact), $7, $8)
ON CONFLICT (contact_id, inbox_id) DO UPDATE SET updated_at = now()
RETURNING contact_id, id;