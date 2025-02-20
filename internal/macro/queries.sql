-- name: get
SELECT
    id,
    name,
    message_content,
    created_at,
    updated_at,
    visibility,
    user_id,
    team_id,
    actions,
    usage_count
FROM
    macros
WHERE
    id = $1;

-- name: get-all
SELECT
    id,
    name,
    message_content,
    created_at,
    updated_at,
    visibility,
    user_id,
    team_id,
    actions,
    usage_count
FROM
    macros
ORDER BY
    updated_at DESC;

-- name: create
INSERT INTO
    macros (name, message_content, user_id, team_id, visibility, actions)
VALUES
    ($1, $2, $3, $4, $5, $6);

-- name: update
UPDATE
    macros
SET
    name = $2,
    message_content = $3,
    user_id = $4,
    team_id = $5,
    visibility = $6,
    actions = $7,
    updated_at = NOW()
WHERE
    id = $1;

-- name: delete
DELETE FROM
    macros
WHERE
    id = $1;

-- name: increment-usage-count
UPDATE
    macros
SET
    usage_count = usage_count + 1
WHERE
    id = $1;