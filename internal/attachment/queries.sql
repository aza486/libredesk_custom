-- name: insert-attachment
INSERT INTO attachments
(store, filename, content_type, size, message_id, content_disposition)
VALUES($1, $2, $3, $4, (SELECT COALESCE((SELECT id FROM messages WHERE uuid = $5), NULL)), $6)
RETURNING uuid, id;

-- name: delete-attachment
DELETE from attachments where id = $1;

-- name: attach-message
update attachments set message_id = $2 where uuid = $1;

-- name: get-message-attachments
select store, "filename", content_type, size, uuid, content_disposition from attachments where message_id = $1;