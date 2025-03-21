-- name: get-sla-policy
SELECT id, name, description, first_response_time, resolution_time, notifications, created_at, updated_at FROM sla_policies WHERE id = $1;

-- name: get-all-sla-policies
SELECT id, name, created_at, updated_at FROM sla_policies ORDER BY updated_at DESC;

-- name: insert-sla-policy
INSERT INTO sla_policies (
   name,
   description, 
   first_response_time,
   resolution_time,
   notifications
) VALUES ($1, $2, $3, $4, $5);

-- name: update-sla-policy
UPDATE sla_policies SET
   name = $2,
   description = $3,
   first_response_time = $4,
   resolution_time = $5,
   notifications = $6,
   updated_at = NOW()
WHERE id = $1;

-- name: delete-sla-policy
DELETE FROM sla_policies WHERE id = $1;

-- name: apply-sla
WITH new_sla AS (
  INSERT INTO applied_slas (
    conversation_id,
    sla_policy_id,
    first_response_deadline_at,
    resolution_deadline_at
  ) VALUES ($1, $2, $3, $4)
  RETURNING conversation_id, id
)
UPDATE conversations c
SET sla_policy_id = $2,
    next_sla_deadline_at = LEAST($3, $4)
FROM new_sla ns
WHERE c.id = ns.conversation_id
RETURNING ns.id;

-- name: get-pending-slas
-- Get all the applied SLAs (applied to a conversation) that are pending
SELECT a.id, a.first_response_deadline_at, c.first_reply_at as conversation_first_response_at, a.sla_policy_id,
a.resolution_deadline_at, c.resolved_at as conversation_resolved_at, c.id as conversation_id, a.first_response_met_at, a.resolution_met_at, a.first_response_breached_at, a.resolution_breached_at
FROM applied_slas a 
JOIN conversations c ON a.conversation_id = c.id and c.sla_policy_id = a.sla_policy_id
WHERE a.status = 'pending'::applied_sla_status;

-- name: update-breach
UPDATE applied_slas SET
   first_response_breached_at = CASE WHEN $2 = 'first_response' THEN NOW() ELSE first_response_breached_at END,
   resolution_breached_at = CASE WHEN $2 = 'resolution' THEN NOW() ELSE resolution_breached_at END,
   updated_at = NOW()
WHERE id = $1;

-- name: update-met
UPDATE applied_slas SET
   first_response_met_at = CASE WHEN $2 = 'first_response' THEN NOW() ELSE first_response_met_at END,
   resolution_met_at = CASE WHEN $2 = 'resolution' THEN NOW() ELSE resolution_met_at END,
   updated_at = NOW()
WHERE id = $1;

-- name: set-next-sla-deadline
UPDATE conversations c
SET next_sla_deadline_at = CASE 
    WHEN c.status_id IN (SELECT id from conversation_statuses where name in ('Resolved', 'Closed')) THEN NULL
    WHEN c.first_reply_at IS NOT NULL AND c.resolved_at IS NULL AND a.resolution_deadline_at IS NOT NULL THEN a.resolution_deadline_at
    WHEN c.first_reply_at IS NULL AND c.resolved_at IS NULL AND a.first_response_deadline_at IS NOT NULL THEN a.first_response_deadline_at
    WHEN a.first_response_deadline_at IS NOT NULL AND a.resolution_deadline_at IS NOT NULL THEN LEAST(a.first_response_deadline_at, a.resolution_deadline_at)
    ELSE NULL
END
FROM applied_slas a
WHERE a.conversation_id = c.id
AND c.id = $1;

-- name: update-sla-status
UPDATE applied_slas
SET
  status = CASE 
     WHEN first_response_met_at IS NOT NULL AND resolution_met_at IS NOT NULL THEN 'met'::applied_sla_status
     WHEN first_response_breached_at IS NOT NULL AND resolution_breached_at IS NOT NULL THEN 'breached'::applied_sla_status
     WHEN (first_response_met_at IS NOT NULL OR first_response_breached_at IS NOT NULL) 
          AND (resolution_met_at IS NOT NULL OR resolution_breached_at IS NOT NULL) THEN 'partially_met'::applied_sla_status
     WHEN first_response_met_at IS NULL AND first_response_breached_at IS NULL THEN 'pending'::applied_sla_status
     ELSE 'pending'::applied_sla_status
  END,
  updated_at = NOW()
WHERE applied_slas.id = $1;

-- name: insert-scheduled-sla-notification
INSERT INTO scheduled_sla_notifications (
   applied_sla_id,
   metric,
   notification_type,
   recipients,
   send_at
) VALUES ($1, $2, $3, $4, $5);

-- name: get-scheduled-sla-notifications
SELECT id, created_at, updated_at, applied_sla_id, metric, notification_type, recipients, send_at, processed_at
FROM scheduled_sla_notifications
WHERE send_at <= NOW() AND processed_at IS NULL;

-- name: get-applied-sla
SELECT a.id,
   a.created_at,
   a.updated_at,
   a.conversation_id,
   a.sla_policy_id,
   a.first_response_deadline_at,
   a.resolution_deadline_at,
   a.first_response_met_at,
   a.resolution_met_at,
   a.first_response_breached_at,
   a.resolution_breached_at,
   a.status,
   c.first_reply_at as conversation_first_response_at,
   c.resolved_at as conversation_resolved_at,
   c.uuid as conversation_uuid,
   c.reference_number as conversation_reference_number,
   c.subject as conversation_subject,
   c.assigned_user_id as conversation_assigned_user_id
FROM applied_slas a inner join conversations c on a.conversation_id = c.id
WHERE a.id = $1;

-- name: mark-notification-processed
UPDATE scheduled_sla_notifications
SET processed_at = NOW(),
      updated_at = NOW()
WHERE id = $1;