-- name: insert-media
INSERT INTO media
(store, filename, content_type)
VALUES($1, $2, $3)
RETURNING uuid;