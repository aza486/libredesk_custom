-- name: insert-media
INSERT INTO media (store, filename, content_type, size, meta, model_id, model_type)
VALUES(
  $1, 
  $2, 
  $3, 
  $4, 
  $5, 
  NULLIF($6, 0),
  NULLIF($7, '')
)
RETURNING id;

-- name: get-media
SELECT *
FROM media
WHERE id = $1;

-- name: get-media-by-filename
SELECT *
FROM media
WHERE filename = $1;

-- name: delete-media
DELETE FROM media
WHERE filename = $1;

-- name: attach-to-model
UPDATE media
SET model_type = $2,
    model_id = $3
WHERE id = $1;

-- name: get-model-media
SELECT id, created_at, "uuid", store, filename, content_type, model_id, model_type, "size", COALESCE(meta->>'disposition', '') AS disposition
FROM media
WHERE model_type = $1
    AND model_id = $2;

-- name: get-unlinked-media
SELECT filename
FROM media
WHERE (
        model_type IS NULL
        OR model_id IS NULL
    )
    AND created_at < $1;
