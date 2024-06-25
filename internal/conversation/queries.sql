-- name: insert-conversation
INSERT INTO conversations
(reference_number, contact_id, status, inbox_id, meta)
VALUES($1, $2, $3, $4, $5)
returning id;


-- name: get-conversations
SELECT
    c.updated_at,
    c.uuid,
    c.assignee_last_seen_at,
    inb.channel as inbox_channel,
    inb.name as inbox_name,
    ct.first_name,
    ct.last_name,
    ct.avatar_url,
    COALESCE(c.meta->>'subject', '') as subject,
    COALESCE(c.meta->>'last_message', '') as last_message,
    COALESCE((c.meta->>'last_message_at')::timestamp, '1970-01-01 00:00:00'::timestamp) as last_message_at,
    (
        SELECT COUNT(*)
        FROM messages m
        WHERE m.conversation_id = c.id AND m.created_at > c.assignee_last_seen_at
    ) AS unread_message_count
FROM conversations c
    JOIN contacts ct ON c.contact_id = ct.id
    JOIN inboxes inb on c.inbox_id = inb.id
WHERE 1=1 %s

-- name: get-assigned-conversations
SELECT uuid from conversations where assigned_user_id = $1;

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
    c.first_reply_at,
    ct.uuid AS contact_uuid,
    ct.first_name as first_name,
    ct.last_name as last_name,
    ct.email as email,
    ct.phone_number as phone_number,
    ct.avatar_url as avatar_url,
    u.uuid AS assigned_user_uuid,
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
LEFT JOIN users u ON u.id = c.assigned_user_id
LEFT JOIN teams at ON at.id = c.assigned_team_id
WHERE c.uuid = $1;

-- name: get-id
SELECT id from conversations where uuid = $1;

-- name: get-uuid
SELECT uuid from conversations where id = $1;

-- name: get-inbox-id
select inbox_id from conversations where uuid = $1;

-- name: update-assigned-user
UPDATE conversations
SET assigned_user_id = (
    SELECT id
    FROM users
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
assigned_user_id = NULL,
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

-- name: update-assignee-last-seen
UPDATE conversations set assignee_last_seen_at = now(),
updated_at = now()
where uuid = $1;

-- name: update-meta
UPDATE conversations set meta = meta || $3 where CASE WHEN $1 > 0 then id = $1 else uuid = $2 end;

-- name: get-conversation-participants
select users.uuid as uuid, first_name, last_name, avatar_url from conversation_participants 
inner join users on users.id = conversation_participants.user_id
where conversation_id = 
(
    select id from conversations where uuid = $1
);

-- name: insert-conversation-participant
INSERT INTO conversation_participants
(user_id, conversation_id)
VALUES($1, (select id from conversations where uuid = $2));

-- name: get-assigned-uuids
select uuids from conversations where assigned_user_id = $1;

-- name: get-unassigned
SELECT id, uuid, assigned_team_id from conversations where assigned_user_id is NULL and assigned_team_id is not null;

-- name: get-assignee-stats
SELECT 
    COUNT(*) AS total_assigned,
    COUNT(CASE WHEN status NOT IN ('Resolved', 'Closed') THEN 1 END) AS unresolved_count,
    COUNT(CASE WHEN first_reply_at IS NULL THEN 1 END) AS awaiting_response_count,
    COUNT(CASE WHEN created_at::date = now()::date THEN 1 END) AS created_today_count
FROM 
    conversations 
WHERE 
    assigned_user_id = $1;


-- name: update-first-reply-at
UPDATE conversations
SET first_reply_at = $2
WHERE first_reply_at IS NULL AND id = $1;