-- name: insert-media
INSERT INTO media (store, filename, content_type, size, meta)
VALUES($1, $2, $3, $4, $5)
RETURNING id;

-- name: get-media
SELECT *
from media
where id = $1;

-- name: delete-media
DELETE from media
where id = $1;

-- name: attach-to-model
UPDATE media
SET model_type = $2,
    model_id = $3
WHERE id = $1;

-- name: get-model-media
SELECT *
FROM media
WHERE model_type = $1
    AND model_id = $2;

-- name: get-unlinked-media
SELECT id
FROM media
WHERE (
        model_type IS NULL
        OR model_id IS NULL
    )
    AND created_at > $1;