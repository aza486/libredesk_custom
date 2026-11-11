-- name: get-notifications
SELECT
    n.id, n.created_at, n.updated_at, n.user_id, n.notification_type,
    n.title, n.body, n.is_read, n.conversation_id, n.message_id, n.actor_id, n.meta,
    u.first_name as actor_first_name, u.last_name as actor_last_name, u.avatar_url as actor_avatar_url,
    c.uuid as conversation_uuid, m.uuid as message_uuid
FROM user_notifications n
LEFT JOIN users u ON u.id = n.actor_id
LEFT JOIN conversations c ON c.id = n.conversation_id
LEFT JOIN conversation_messages m ON m.id = n.message_id
WHERE n.user_id = $1 AND (($4::uuid[] IS NULL AND NOT (COALESCE((c.custom_attributes->>'customer_visibility')::boolean,false) OR EXISTS(SELECT 1 FROM conversation_tags ct JOIN tags t ON t.id=ct.tag_id WHERE ct.conversation_id=c.id AND t.name='🦽Kundenticket'))) OR n.conversation_id IS NULL OR c.uuid=ANY($4::uuid[]))
 AND (
   COALESCE(c.custom_attributes->>'access_mode','public') <> 'personal'
   OR (c.custom_attributes->'visible_users') @> jsonb_build_array($1::bigint)
   OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='Admin')
 )
ORDER BY n.created_at DESC
LIMIT $2 OFFSET $3;

-- name: get-notification-stats
SELECT
    COUNT(*) FILTER (WHERE is_read = false) as unread_count,
    COUNT(*) as total_count
FROM user_notifications n
LEFT JOIN conversations c ON c.id=n.conversation_id
WHERE n.user_id = $1 AND (($2::uuid[] IS NULL AND NOT (COALESCE((c.custom_attributes->>'customer_visibility')::boolean,false) OR EXISTS(SELECT 1 FROM conversation_tags ct JOIN tags t ON t.id=ct.tag_id WHERE ct.conversation_id=c.id AND t.name='🦽Kundenticket'))) OR n.conversation_id IS NULL OR c.uuid=ANY($2::uuid[]))
 AND (
   COALESCE(c.custom_attributes->>'access_mode','public') <> 'personal'
   OR (c.custom_attributes->'visible_users') @> jsonb_build_array($1::bigint)
   OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='Admin')
 );

-- name: insert-notification
INSERT INTO user_notifications (user_id, notification_type, title, body, conversation_id, message_id, actor_id, meta)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, created_at, updated_at, user_id, notification_type, title, body, is_read, conversation_id, message_id, actor_id, meta;

-- name: mark-as-read
UPDATE user_notifications SET is_read = true, updated_at = now() WHERE id = $1 AND user_id = $2 RETURNING id;

-- name: mark-assignment-as-read
UPDATE user_notifications
SET is_read = true, updated_at = now()
WHERE conversation_id = $1
  AND user_id = $2
  AND notification_type = 'assignment'
  AND is_read = false;

-- name: mark-all-as-read
UPDATE user_notifications SET is_read = true, updated_at = now() WHERE user_id = $1 AND is_read = false;

-- name: delete-notification
DELETE FROM user_notifications WHERE id = $1 AND user_id = $2;

-- name: delete-all-notifications
DELETE FROM user_notifications WHERE user_id = $1;

-- name: delete-old-notifications
DELETE FROM user_notifications WHERE created_at < NOW() - INTERVAL '30 days';

-- name: can-receive-conversation
SELECT EXISTS (
 SELECT 1 FROM conversations c WHERE c.id=$2 AND NOT (COALESCE((c.custom_attributes->>'customer_visibility')::boolean,false) OR EXISTS(SELECT 1 FROM conversation_tags ct JOIN tags t ON t.id=ct.tag_id WHERE ct.conversation_id=c.id AND t.name='🦽Kundenticket')) AND (
   COALESCE(c.custom_attributes->>'access_mode','public') <> 'personal'
   OR (c.custom_attributes->'visible_users') @> jsonb_build_array($1::bigint)
   OR EXISTS (SELECT 1 FROM user_roles ur JOIN roles r ON r.id=ur.role_id WHERE ur.user_id=$1 AND r.name='Admin')
 )
);
