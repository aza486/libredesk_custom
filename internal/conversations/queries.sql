-- name: get-conversations
SELECT 
    c.updated_at,
    c.uuid,
    ct.first_name as contact_first_name,
    ct.last_name as contact_last_name,
    ct.avatar_url as contact_avatar_url,
    (
        SELECT content
        FROM messages m
        WHERE m.conversation_id = c.id
        ORDER BY m.created_at DESC
        LIMIT 1
    ) AS last_message
FROM conversations c
    JOIN contacts ct ON c.contact_id = ct.id
ORDER BY c.updated_at DESC;

-- name: get-conversation
SELECT
    c.created_at,
    c.updated_at,
    c.closed_at,
    c.resolved_at,
    c.priority,
    c.status,
    c.uuid,
    c.reference_number,
    ct.uuid AS contact_uuid,
    ct.first_name as contact_first_name,
    ct.last_name as contact_last_name,
    ct.email as contact_email,
    ct.phone_number as contact_phone_number,
    ct.avatar_url as contact_avatar_url,
    aa.uuid AS assigned_agent_uuid,
    at.uuid AS assigned_team_uuid,
    (SELECT COALESCE(
        (SELECT json_agg(t.name) 
        FROM tags t 
        INNER JOIN conversation_tags ct ON ct.tag_id = t.id 
        WHERE ct.conversation_id = c.id), 
        '[]'::json
    )) AS tags
FROM conversations c
JOIN contacts ct ON c.contact_id = ct.id
LEFT JOIN agents aa ON aa.id = c.assigned_agent_id
LEFT JOIN teams at ON at.id = c.assigned_team_id
WHERE c.uuid = $1;


-- name: get-messages
WITH conversation_id AS (
    SELECT id
    FROM conversations
    WHERE uuid = $1
)
SELECT
    m.created_at,
    m.updated_at,
    m.status,
    m.type,
    m.content,
    m.uuid,
    CASE 
        WHEN m.contact_id IS NOT NULL THEN ct.first_name
        ELSE aa.first_name
    END as first_name,
    CASE
        WHEN m.agent_id IS NOT NULL THEN aa.last_name
        ELSE ct.last_name
    END AS last_name,
     CASE
        WHEN m.agent_id IS NOT NULL THEN aa.avatar_url
        ELSE ct.avatar_url
    END AS avatar_url
FROM messages m
LEFT JOIN contacts ct ON m.contact_id = ct.id
LEFT JOIN agents aa ON m.agent_id = aa.id
WHERE m.conversation_id = (SELECT id FROM conversation_id)
ORDER BY m.created_at;


-- name: update-assigned-agent
UPDATE conversations
SET assigned_agent_id = (
    SELECT id
    FROM agents
    WHERE uuid = $2
),
updated_at = now()
WHERE uuid = $1;


-- name: update-assigned-team
UPDATE conversations
SET assigned_team_id = (
    SELECT id
    FROM teams
    WHERE uuid = $2
),
updated_at = now()
WHERE uuid = $1;

-- name: update-status
UPDATE conversations
SET status = $2::text,
    resolved_at = CASE WHEN $2::text = 'Resolved' THEN CURRENT_TIMESTAMP ELSE NULL END,
updated_at = now()
WHERE uuid = $1;

-- name: update-priority
UPDATE conversations set priority = $2,
updated_at = now()
where uuid = $1;

