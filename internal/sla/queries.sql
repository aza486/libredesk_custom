-- name: get-sla-policy
SELECT * FROM sla_policies WHERE id = $1;

-- name: get-all-sla-policies
SELECT * FROM sla_policies ORDER BY updated_at DESC;

-- name: insert-sla-policy
INSERT INTO sla_policies (
   name,
   description, 
   first_response_time,
   resolution_time
) VALUES ($1, $2, $3, $4);

-- name: delete-sla-policy
DELETE FROM sla_policies WHERE id = $1;

-- name: update-sla-policy
UPDATE sla_policies SET
   name = $2,
   description = $3,
   first_response_time = $4,
   resolution_time = $5,
   updated_at = NOW()
WHERE id = $1;

-- name: apply-sla
WITH new_sla AS (
 INSERT INTO applied_slas (
   conversation_id,
   sla_policy_id,
   first_response_deadline_at,
   resolution_deadline_at
 ) VALUES ($1, $2, $3, $4)
 RETURNING conversation_id
)
UPDATE conversations 
SET sla_policy_id = $2,
next_sla_deadline_at = LEAST(
   NULLIF($3, NULL),
   NULLIF($4, NULL)
)
WHERE id IN (SELECT conversation_id FROM new_sla);

-- name: get-pending-slas
SELECT a.id, a.first_response_deadline_at, c.first_reply_at as first_response_at,
a.resolution_deadline_at, c.resolved_at as resolved_at
FROM applied_slas a 
JOIN conversations c ON a.conversation_id = c.id and c.sla_policy_id = a.sla_policy_id
WHERE (first_response_breached_at IS NULL AND first_response_met_at IS NULL)
  OR (resolution_breached_at IS NULL AND resolution_met_at IS NULL);

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

-- name: get-latest-sla-deadlines
SELECT first_response_deadline_at, resolution_deadline_at
FROM applied_slas 
WHERE conversation_id = $1 
ORDER BY created_at DESC LIMIT 1;