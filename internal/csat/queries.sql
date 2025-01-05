-- name: insert
INSERT INTO csat_responses (
        conversation_id,
        assigned_agent_id
    )
VALUES ($1, $2);

-- name: get
SELECT id,
    uuid,
    created_at,
    updated_at,
    conversation_id,
    assigned_agent_id,
    rating,
    feedback,
    response_timestamp
FROM csat_responses
WHERE uuid = $1;

-- name: update
UPDATE csat_responses
SET rating = $2,
    feedback = $3,
    response_timestamp = NOW()
WHERE uuid = $1;
