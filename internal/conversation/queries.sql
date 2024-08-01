-- name: insert-conversation
INSERT INTO conversations
(reference_number, contact_id, status, inbox_id, meta)
VALUES($1, $2, $3, $4, $5)
returning id, uuid;

-- name: get-conversations
SELECT
    c.updated_at,
    c.uuid,
    c.assignee_last_seen_at,
    ct.first_name,
    ct.last_name,
    ct.avatar_url,
    inb.channel as inbox_channel,
    inb.name as inbox_name,
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

-- name: get-conversations-uuids
SELECT
    c.uuid
FROM conversations c
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
    c.assigned_user_id,
    c.assigned_team_id,
    c.meta->>'subject' as subject,
    ct.id as contact_id,
    ct.uuid AS contact_uuid,
    ct.first_name as first_name,
    ct.last_name as last_name,
    ct.email as email,
    ct.phone_number as phone_number,
    ct.avatar_url as avatar_url,
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

-- name: get-recent-conversations
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
WHERE c.created_at > $1 and c.uuid = 'e2f69c9f-17f5-4d09-9aae-12c2a79046a2';


-- name: get-conversation-id
SELECT id from conversations where uuid = $1;

-- name: get-conversation-uuid
SELECT uuid from conversations where id = $1;

-- name: update-conversation-assigned-user
UPDATE conversations
SET assigned_user_id = $2,
updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-assigned-team
UPDATE conversations
SET assigned_team_id = $2,
assigned_user_id = NULL,
updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-status
UPDATE conversations
SET status = $2::text,
    resolved_at = CASE WHEN $2::text = 'Resolved' THEN CURRENT_TIMESTAMP ELSE NULL END,
updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-priority
UPDATE conversations set priority = $2,
updated_at = now()
where uuid = $1;

-- name: update-conversation-assignee-last-seen
UPDATE conversations set assignee_last_seen_at = now(),
updated_at = now()
where uuid = $1;

-- name: update-conversation-meta
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


-- name: get-unassigned-conversations
SELECT
    c.updated_at,
    c.uuid,
    c.assignee_last_seen_at,
    c.assigned_team_id,
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
    JOIN inboxes inb on c.inbox_id = inb.id where assigned_user_id is NULL and assigned_team_id is not null;

-- name: get-dashboard-counts
SELECT json_build_object(
    'resolved_count', COUNT(CASE WHEN status IN ('Resolved') THEN 1 END),
    'unresolved_count', COUNT(CASE WHEN status NOT IN ('Resolved', 'Closed') THEN 1 END),
    'awaiting_response_count', COUNT(CASE WHEN first_reply_at IS NULL THEN 1 END),
    'total_count', COUNT(*)
)
FROM conversations
WHERE 1=1 %s;

-- name: get-dashboard-charts
WITH new_conversations AS (
    SELECT json_agg(row_to_json(agg)) AS data
    FROM (
        SELECT
            TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date,
            COUNT(*) AS new_conversations
        FROM
            conversations
        WHERE 1=1 %s
        GROUP BY
            date
        ORDER BY
            date
    ) agg
),
status_summary AS (
    SELECT json_agg(row_to_json(agg)) AS data
    FROM (
        SELECT 
            status,
            COUNT(*) FILTER (WHERE priority = 'Low') AS "Low",
            COUNT(*) FILTER (WHERE priority = 'Medium') AS "Medium",
            COUNT(*) FILTER (WHERE priority = 'High') AS "High"
        FROM 
            conversations
        WHERE 1=1 %s
        GROUP BY 
            status
    ) agg
)
SELECT json_build_object(
    'new_conversations', (SELECT data FROM new_conversations),
    'status_summary', (SELECT data FROM status_summary)
) AS result;


-- name: update-conversation-first-reply-at
UPDATE conversations
SET first_reply_at = $2
WHERE first_reply_at IS NULL AND id = $1;

-- name: add-conversation-tag
INSERT INTO conversation_tags (conversation_id, tag_id)
VALUES(
    (
        SELECT id
        from conversations
        where uuid = $1
    ),
    $2
) ON CONFLICT DO NOTHING

-- name: delete-conversation-tags
DELETE FROM conversation_tags
WHERE conversation_id = (
    SELECT id
    from conversations
    where uuid = $1
) AND tag_id NOT IN (SELECT unnest($2::int[]));


-- name: get-to-address
SELECT cm.source_id from conversations c inner join contact_methods cm on cm.contact_id = c.contact_id where c.id = $1 and cm.source = $2;

-- name: get-conversation-uuid-from-message-uuid
SELECT c.uuid AS conversation_uuid
FROM messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.uuid = $1;

-- MESSAGE queries.
-- name: get-latest-received-message-source-id
SELECT source_id
FROM messages
WHERE conversation_id = $1 and status = 'received'
ORDER BY id DESC
LIMIT 1;

-- name: get-pending-messages
SELECT
    m.created_at,
    m.id,
    m.uuid,
    m.sender_id,
    m.type,
    m.status,
    m.content,
    m.conversation_id,
    m.content_type,
    m.source_id,
    c.inbox_id,
    c.uuid as conversation_uuid,
    COALESCE(c.meta->>'subject', '') as subject
FROM messages m
INNER JOIN conversations c ON c.id = m.conversation_id
WHERE m.status = 'pending'
AND NOT(m.id = ANY($1::INT[]))

-- name: get-message
SELECT
    m.created_at,
    m.updated_at,
    m.status,
    m.type,
    m.content,
    m.uuid,
    m.private,
    m.sender_type,
    u.uuid as sender_uuid,
    COALESCE(
        json_agg(
            json_build_object(
                'name', a.filename,
                'content_type', a.content_type,
                'uuid', a.uuid,
                'size', a.size
            ) ORDER BY a.filename
        ) FILTER (WHERE a.message_id IS NOT NULL), 
        '[]'::json
    ) AS attachments
FROM messages m
LEFT JOIN attachments a 
    ON a.message_id = m.id 
    AND a.content_disposition = 'attachment'
LEFT JOIN users u on u.id = m.sender_id
WHERE m.uuid = $1
GROUP BY 
    m.id, m.created_at, m.updated_at, m.status, m.type, m.content, m.uuid, m.private, m.sender_type, u.uuid
ORDER BY m.created_at;

-- name: get-messages
WITH conversation_id AS (
    SELECT id
    FROM conversations
    WHERE uuid = $1
    LIMIT 1
),
attachments AS (
    SELECT 
        model_id as message_id,
        json_agg(
            json_build_object(
                'name', filename,
                'content_type', content_type,
                'uuid', uuid,
                'size', size
            ) ORDER BY filename
        ) AS attachment_details
    FROM media
    WHERE model_type = 'messages'
    GROUP BY message_id
)
SELECT
    m.created_at,
    m.updated_at,
    m.status,
    m.type,
    m.content,
    m.uuid,
    m.private,
    m.sender_id,
    m.sender_type,
    u.uuid as sender_uuid,
    COALESCE(a.attachment_details, '[]'::json) AS attachments
FROM messages m
LEFT JOIN attachments a ON a.message_id = m.id
LEFT JOIN users u on u.id = m.sender_id
WHERE m.conversation_id = (SELECT id FROM conversation_id) ORDER BY m.created_at DESC
%s;

-- name: insert-message
WITH conversation_id AS (
    SELECT id 
    FROM conversations 
    WHERE CASE 
        WHEN $3 > 0 THEN id = $3 
        ELSE uuid = $4 
    END
)
INSERT INTO messages (
    "type",
    status,
    conversation_id,
    "content",
    sender_id,
    sender_type,
    private,
    content_type,
    source_id,
    inbox_id,
    meta
)
VALUES (
    $1,
    $2,
    (SELECT id FROM conversation_id),
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12
)
RETURNING id, uuid, created_at;

-- name: message-exists
SELECT conversation_id
FROM messages
WHERE source_id = ANY($1::text []);

-- name: update-message-content
update messages
set content = $1
where id = $2;

-- name: update-message-status
update messages set status = $1 where uuid = $2;
