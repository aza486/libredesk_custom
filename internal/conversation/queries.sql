-- name: unsnooze-all
UPDATE conversations
SET snoozed_until = NULL, status_id = (SELECT id FROM conversation_statuses WHERE name = 'Open')
WHERE snoozed_until <= now();

-- name: insert-conversation
INSERT INTO conversations
(contact_id, contact_channel_id, status_id, inbox_id, last_message, last_message_at, subject)
VALUES($1, $2, (SELECT id FROM conversation_statuses WHERE name = $3), $4, $5, $6, $7)
RETURNING id, uuid;

-- name: get-conversations
SELECT
COUNT(*) OVER() as total,
conversations.created_at,
conversations.updated_at,
conversations.uuid,
conversations.assignee_last_seen_at,
users.first_name as "contact.first_name",
users.last_name as "contact.last_name",
users.avatar_url as "contact.avatar_url", 
inboxes.channel as inbox_channel,
inboxes.name as inbox_name,
conversations.sla_policy_id,
conversations.first_reply_at,
conversations.resolved_at,
conversations.subject,
conversations.last_message,
conversations.last_message_at,
conversations.next_sla_deadline_at,
conversations.priority_id,
(
  SELECT CASE WHEN COUNT(*) > 9 THEN 10 ELSE COUNT(*) END
  FROM (
    SELECT 1 FROM conversation_messages 
    WHERE conversation_id = conversations.id 
      AND created_at > conversations.assignee_last_seen_at
    LIMIT 10
  ) t
) as unread_message_count,
conversation_statuses.name as status,
conversation_priorities.name as priority
FROM conversations
JOIN users ON contact_id = users.id
JOIN inboxes ON inbox_id = inboxes.id  
LEFT JOIN conversation_statuses ON status_id = conversation_statuses.id
LEFT JOIN conversation_priorities ON priority_id = conversation_priorities.id
WHERE 1=1 %s

-- name: get-conversation
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.closed_at,
    c.resolved_at,
    c.inbox_id,
    c.status_id,
    c.priority_id,
    p.name as priority,
    s.name as status,
    c.uuid,
    c.reference_number,
    c.first_reply_at,
    c.assigned_user_id,
    c.assigned_team_id,
    c.subject,
    c.contact_id,
    c.sla_policy_id,
    c.last_message,
    sla.name as sla_policy_name,
    (SELECT COALESCE(
        (SELECT json_agg(t.name)
        FROM tags t
        INNER JOIN conversation_tags ct ON ct.tag_id = t.id
        WHERE ct.conversation_id = c.id),
        '[]'::json
    )) AS tags,
    ct.first_name as "contact.first_name",
    ct.last_name as "contact.last_name",
    ct.email as "contact.email",
    ct.avatar_url as "contact.avatar_url",
    ct.phone_number as "contact.phone_number"
FROM conversations c
JOIN users ct ON c.contact_id = ct.id
LEFT JOIN sla_policies sla ON c.sla_policy_id = sla.id
LEFT JOIN teams at ON at.id = c.assigned_team_id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
LEFT JOIN conversation_priorities p ON c.priority_id = p.id
WHERE ($1 > 0 AND c.id = $1)
   OR ($2 != '' AND c.uuid = $2::uuid);

-- name: get-conversations-created-after
SELECT
    c.id,
    c.created_at,
    c.updated_at,
    c.closed_at,
    c.resolved_at,
    p.name as priority,
    s.name as status,
    c.uuid,
    c.reference_number,
    c.first_reply_at,
    u.first_name as first_name,
    u.last_name as last_name,
    u.email as email,
    u.avatar_url as avatar_url,
    (SELECT COALESCE(
        (SELECT json_agg(t.name)
        FROM tags t
        INNER JOIN conversation_tags ct ON ct.tag_id = t.id
        WHERE ct.conversation_id = c.id),
        '[]'::json
    )) AS tags
FROM conversations c
JOIN users u ON c.contact_id = u.id
LEFT JOIN conversation_statuses s ON c.status_id = s.id
LEFT JOIN conversation_priorities p ON c.priority_id = p.id
WHERE c.created_at > $1;

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
updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-status
UPDATE conversations
SET status_id = (SELECT id FROM conversation_statuses WHERE name = $2),
    resolved_at = CASE 
                    WHEN $2 = 'Resolved' THEN 
                        COALESCE(resolved_at, CURRENT_TIMESTAMP)
                    WHEN $2 != 'Resolved' THEN 
                        resolved_at
                  END,
    closed_at = CASE 
                  WHEN $2 = 'Closed' THEN 
                      COALESCE(closed_at, CURRENT_TIMESTAMP)
                  ELSE 
                      closed_at
                END,
    snoozed_until = CASE 
                      WHEN $2 = 'Snoozed' THEN 
                          $3::timestamptz
                      ELSE 
                          NULL
                    END,
    updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-priority
UPDATE conversations 
SET priority_id = (SELECT id FROM conversation_priorities WHERE name = $2),
    updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-assignee-last-seen
UPDATE conversations 
SET assignee_last_seen_at = now(),
    updated_at = now()
WHERE uuid = $1;

-- name: update-conversation-last-message
UPDATE conversations SET last_message = $3, last_message_at = $4 WHERE CASE 
    WHEN $1 > 0 THEN id = $1
    ELSE uuid = $2
END

-- name: get-conversation-participants
SELECT users.id as id, first_name, last_name, avatar_url 
FROM conversation_participants
INNER JOIN users ON users.id = conversation_participants.user_id
WHERE conversation_id =
(
    SELECT id FROM conversations WHERE uuid = $1
);

-- name: insert-conversation-participant
INSERT INTO conversation_participants
(user_id, conversation_id)
VALUES($1, (SELECT id FROM conversations WHERE uuid = $2));

-- name: get-unassigned-conversations
SELECT
    c.created_at,
    c.updated_at,
    c.uuid,
    c.assigned_team_id,
    inb.channel as inbox_channel,
    inb.name as inbox_name
FROM conversations c
    JOIN inboxes inb ON c.inbox_id = inb.id 
WHERE assigned_user_id IS NULL AND assigned_team_id IS NOT NULL;

-- name: get-dashboard-counts
SELECT json_build_object(
    'open', COUNT(*),
    'awaiting_response', COUNT(CASE WHEN c.first_reply_at IS NULL THEN 1 END),
    'unassigned', COUNT(CASE WHEN c.assigned_user_id IS NULL THEN 1 END),
    'pending', COUNT(CASE WHEN c.first_reply_at IS NOT NULL THEN 1 END)
)
FROM conversations c
INNER JOIN conversation_statuses s ON c.status_id = s.id
WHERE s.name not in ('Resolved', 'Closed') AND 1=1 %s;

-- name: get-dashboard-charts
WITH new_conversations AS (
    SELECT json_agg(row_to_json(agg)) AS data
    FROM (
        SELECT
            TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date,
            COUNT(*) AS count
        FROM
            conversations c
        WHERE 1=1 %s
        GROUP BY
            date
        ORDER BY
            date
    ) agg
),
resolved_conversations AS (
    SELECT json_agg(row_to_json(agg)) AS data
    FROM (
        SELECT
            TO_CHAR(resolved_at::date, 'YYYY-MM-DD') AS date,
            COUNT(*) AS count
        FROM
            conversations c
        WHERE c.resolved_at IS NOT NULL AND 1=1 %s
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
            s.name as status,
            COUNT(*) FILTER (WHERE p.name = 'Low') AS "Low",
            COUNT(*) FILTER (WHERE p.name = 'Medium') AS "Medium",
            COUNT(*) FILTER (WHERE p.name = 'High') AS "High"
        FROM 
            conversations c
        LEFT join conversation_statuses s on s.id = c.status_id
        LEFT join conversation_priorities p on p.id = c.priority_id
        WHERE 1=1 AND s.name > '' %s
        GROUP BY 
            s.name
    ) agg
),
messages_sent as (
    SELECT json_agg(row_to_json(agg)) AS data
    FROM (
        SELECT
            TO_CHAR(created_at::date, 'YYYY-MM-DD') AS date,
            COUNT(*) AS count
        FROM
            conversation_messages c
        WHERE status = 'sent' AND 1=1 %s
        GROUP BY
            date
        ORDER BY
            date
    ) agg
)
SELECT json_build_object(
    'new_conversations', (SELECT data FROM new_conversations),
    'resolved_conversations', (SELECT data FROM resolved_conversations),
    'messages_sent', (SELECT data FROM messages_sent),
    'status_summary', (SELECT data FROM status_summary)
) AS result;


-- name: update-conversation-first-reply-at
UPDATE conversations
SET first_reply_at = $2
WHERE first_reply_at IS NULL AND id = $1;

-- name: upsert-conversation-tags
WITH conversation_id AS (
    SELECT id FROM conversations WHERE uuid = $1
),
inserted AS (
    INSERT INTO conversation_tags (conversation_id, tag_id)
    SELECT conversation_id.id, t.id
    FROM conversation_id, tags t
    WHERE t.name = ANY($2::text[])
    ON CONFLICT (conversation_id, tag_id) DO UPDATE SET tag_id = EXCLUDED.tag_id
)
DELETE FROM conversation_tags
WHERE conversation_id = (SELECT id FROM conversation_id) 
AND tag_id NOT IN (
    SELECT id FROM tags WHERE name = ANY($2::text[])
);

-- name: get-to-address
SELECT cc.identifier 
FROM conversations c 
INNER JOIN contact_channels cc ON cc.id = c.contact_channel_id 
WHERE c.id = $1;

-- name: get-conversation-uuid-from-message-uuid
SELECT c.uuid AS conversation_uuid
FROM conversation_messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.uuid = $1;

-- name: unassign-open-conversations
UPDATE conversations
SET assigned_user_id = NULL,
    updated_at = now()
WHERE assigned_user_id = $1 AND status_id in (SELECT id FROM conversation_statuses WHERE name NOT IN ('Resolved', 'Closed'));

-- MESSAGE queries.
-- name: get-latest-received-message-source-id
SELECT source_id
FROM conversation_messages
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
    c.subject
FROM conversation_messages m
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
    m.sender_id,
    m.meta,
    COALESCE(
        json_agg(
            json_build_object(
                'name', media.filename,
                'content_type', media.content_type,
                'uuid', media.uuid,
                'size', media.size,
                'content_id', media.content_id,
                'disposition', media.disposition
            ) ORDER BY media.filename
        ) FILTER (WHERE media.id IS NOT NULL),
        '[]'::json
    ) AS attachments
FROM conversation_messages m
LEFT JOIN media ON media.model_id = m.id AND media.model_type = 'messages'
WHERE m.uuid = $1
GROUP BY 
    m.id, m.created_at, m.updated_at, m.status, m.type, m.content, m.uuid, m.private, m.sender_type
ORDER BY m.created_at;

-- name: get-messages
SELECT
   COUNT(*) OVER() AS total,
   m.created_at,
   m.updated_at,
   m.status,
   m.type, 
   m.content,
   m.uuid,
   m.private,
   m.sender_id,
   m.sender_type,
   m.meta,
   COALESCE(
     (SELECT json_agg(
       json_build_object(
         'name', filename,
         'content_type', content_type, 
         'uuid', uuid,
         'size', size,
         'content_id', content_id,
         'disposition', disposition
       ) ORDER BY filename
     ) FROM media 
     WHERE model_type = 'messages' AND model_id = m.id),
   '[]'::json) AS attachments
FROM conversation_messages m
WHERE m.conversation_id = (
   SELECT id FROM conversations WHERE uuid = $1 LIMIT 1
)
ORDER BY m.created_at DESC %s

-- name: insert-message
WITH conversation_id AS (
    SELECT id 
    FROM conversations 
    WHERE CASE 
        WHEN $3 > 0 THEN id = $3 
        ELSE uuid = $4 
    END
)
INSERT INTO conversation_messages (
    "type",
    status,
    conversation_id,
    "content",
    text_content,
    sender_id,
    sender_type,
    private,
    content_type,
    source_id,
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

-- name: message-exists-by-source-id
SELECT conversation_id
FROM conversation_messages
WHERE source_id = ANY($1::text []);

-- name: get-conversation-by-message-id
SELECT
    c.id,
    c.uuid,
    c.assigned_team_id,
    c.assigned_user_id
FROM conversation_messages m
JOIN conversations c ON m.conversation_id = c.id
WHERE m.id = $1;

-- name: update-message-status
update conversation_messages set status = $1, updated_at = now() where uuid = $2;
