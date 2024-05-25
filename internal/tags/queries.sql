-- name: insert-conversation-tag
INSERT INTO conversation_tags (conversation_id, tag_id)
VALUES(
    (
        SELECT id
        from conversations
        where uuid = $1
    ),
    $2
) ON CONFLICT DO NOTHING;

-- name: delete-conversation-tags
DELETE FROM conversation_tags
WHERE conversation_id = (
    SELECT id
    from conversations
    where uuid = $1
) AND tag_id NOT IN (SELECT unnest($2::int[]));

-- name: get-all-tags
select id,
    created_at,
    name
from tags;