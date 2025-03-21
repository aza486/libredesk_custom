-- name: search-conversations
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject
FROM conversations
WHERE reference_number::text = $1;

-- name: search-messages
SELECT
    c.created_at as "conversation_created_at",
    c.reference_number as "conversation_reference_number",
    c.uuid as "conversation_uuid",
    m.text_content
FROM conversation_messages m
    JOIN conversations c ON m.conversation_id = c.id
WHERE m.type != 'activity' and m.text_content ILIKE '%' || $1 || '%'
LIMIT 30;

-- name: search-contacts
SELECT 
    id,
    created_at,
    first_name,
    last_name,
    email
FROM users
WHERE type = 'contact'
AND deleted_at IS NULL
AND email ILIKE '%' || $1 || '%'
LIMIT 15;
