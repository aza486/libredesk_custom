-- name: get-all-oidc
SELECT id, provider_url FROM oidc;

-- name: get-oidc
SELECT * FROM oidc WHERE id = $1;

-- name: insert-oidc
INSERT INTO oidc (provider_url, client_id, client_secret, redirect_uri) 
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: update-oidc
UPDATE oidc 
SET provider_url = $2, client_id = $3, client_secret = $4, redirect_uri = $5, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: delete-oidc
DELETE FROM oidc WHERE id = $1;
