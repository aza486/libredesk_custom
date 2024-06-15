-- name: get-pending-messages
SELECT
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

-- name: get-to-address
SELECT cm.source_id from conversations c inner join contact_methods cm on cm.contact_id = c.contact_id where c.id = $1 and cm.source = $2;

-- name: get-in-reply-to
SELECT source_id
FROM messages
WHERE conversation_id = $1 and status = 'received'
ORDER BY id DESC
LIMIT 1;

-- name: get-messages
WITH conversation_id AS (
    SELECT id
    FROM conversations
    WHERE uuid = $1
    LIMIT 1
),
attachments AS (
    SELECT 
        message_id,
        json_agg(
            json_build_object(
                'name', filename,
                'content_type', content_type,
                'uuid', uuid,
                'size', size
            ) ORDER BY filename
        ) AS attachment_details
    FROM attachments
    WHERE content_disposition = 'attachment'
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
WHERE m.conversation_id = (SELECT id FROM conversation_id)
ORDER BY m.created_at;


-- name: insert-message-by-id
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
VALUES(
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11
    )
returning id,
    uuid;


-- name: insert-message-by-uuid
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
VALUES(
        $1,
        $2,
        (SELECT id from conversations where uuid = $3),
        $4,
        $5,
        $6,
        $7,
        $8,
        $9,
        $10,
        $11
    )
returning id,
    uuid;

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
