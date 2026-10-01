-- name: search-conversations
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    users.email AS contact_email,
    cs.name AS status
FROM
    conversations
    LEFT JOIN users ON conversations.contact_id = users.id
    LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE
    conversations.reference_number::text = $1::text
    OR conversations.subject ILIKE '%' || $1::text || '%'
    OR users.email ILIKE '%' || $1::text || '%'
ORDER BY
    (conversations.reference_number::text = $1::text) DESC,
    conversations.created_at DESC
LIMIT 300;

-- name: search-messages
SELECT
    created_at,
    conversation_created_at,
    conversation_reference_number,
    conversation_uuid,
    conversation_subject,
    contact_email,
    text_content,
    conversation_status
FROM (
    SELECT DISTINCT ON (c.id)
        m.created_at,
        c.created_at AS conversation_created_at,
        c.reference_number AS conversation_reference_number,
        c.uuid AS conversation_uuid,
        c.subject AS conversation_subject,
        u.email AS contact_email,
        LEFT(m.text_content, 200) AS text_content,
        cs.name AS conversation_status
    FROM
        conversation_messages m
        JOIN conversations c ON m.conversation_id = c.id
        LEFT JOIN users u ON c.contact_id = u.id
        LEFT JOIN conversation_statuses cs ON c.status_id = cs.id
    WHERE
        m.type != 'activity'
        AND length(trim($1::text)) > 0
        AND m.text_content ILIKE ALL (
            ARRAY(
                SELECT '%' || w || '%'
                FROM unnest(regexp_split_to_array(trim($1::text), '\s+')) AS w
            )
        )
    ORDER BY c.id, m.created_at DESC
) hits
ORDER BY created_at DESC
LIMIT 300;

-- name: search-contacts
SELECT
    id,
    created_at,
    first_name,
    last_name,
    email,
    external_user_id
FROM users
WHERE
    type = 'contact'
    AND deleted_at IS NULL
    AND email ILIKE '%' || $1 || '%'
LIMIT 15;