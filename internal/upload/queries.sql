-- name: insert-upload
INSERT INTO uploads
(filename)
VALUES($1)
RETURNING id;

-- name: delete-upload
DELETE FROM uploads WHERE id = $1;
