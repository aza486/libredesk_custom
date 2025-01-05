-- name: get-sla-policy
SELECT id,
    created_at,
    updated_at,
    "name",
    description,
    first_response_time,
    resolution_time
FROM sla_policies
WHERE id = $1;

-- name: get-all-sla-policies
SELECT id,
    created_at,
    updated_at,
    "name",
    description,
    first_response_time,
    resolution_time
FROM sla_policies
ORDER BY updated_at DESC;

-- name: insert-sla-policy
INSERT INTO sla_policies (
        "name",
        description,
        first_response_time,
        resolution_time
    )
VALUES ($1, $2, $3, $4);

-- name: delete-sla-policy
DELETE FROM sla_policies
WHERE id = $1;

-- name: update-sla-policy
UPDATE sla_policies
SET "name" = $2,
    description = $3,
    first_response_time = $4,
    resolution_time = $5,
    updated_at = NOW()
WHERE id = $1;

-- name: apply-sla-policy
INSERT INTO applied_slas (
    status,
    conversation_id,
    sla_policy_id
    )
VALUES ($1, $2, $3);

-- name: get-unbreached-slas
SELECT 
	cs.id,
    cs.sla_policy_id, 
    cs.sla_type, 
    cs.breached_at,
    cs.due_at,
    c.created_at as conversation_created_at,
    c.first_reply_at as conversation_first_reply_at,
    c.last_message_at as conversation_last_message_at,
    c.resolved_at as conversation_resolved_at,
    c.assigned_team_id as conversation_assigned_team_id
FROM conversation_slas cs
LEFT JOIN conversations c ON cs.conversation_id = c.id
WHERE cs.breached_at is NULL AND cs.met_at is NULL;

-- name: update-breached-at
UPDATE conversation_slas
SET breached_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: update-due-at
WITH updated_slas AS (
    UPDATE conversation_slas
    SET due_at = $2,
        updated_at = NOW()
    WHERE id = $1
    RETURNING conversation_id
)
-- Also set in conversations table.
UPDATE conversations
SET next_sla_deadline_at = $2
WHERE id IN (SELECT conversation_id FROM updated_slas)
  AND (next_sla_deadline_at IS NULL OR next_sla_deadline_at > $2);

-- name: update-met-at
UPDATE conversation_slas
SET met_at = $2, updated_at = NOW()
WHERE id = $1;

-- name: insert-conversation-sla
WITH inserted AS (
    INSERT INTO conversation_slas (conversation_id, sla_policy_id, sla_type)
    VALUES ($1, $2, $3)
    RETURNING conversation_id, sla_policy_id
)
UPDATE conversations
SET sla_policy_id = inserted.sla_policy_id
FROM inserted
WHERE conversations.id = inserted.conversation_id;