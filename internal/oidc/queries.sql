-- name: get-all-oidc
SELECT id, name, disabled, provider, updated_at FROM oidc order by updated_at desc;

-- name: get-oidc
SELECT * FROM oidc WHERE id = $1;

-- name: insert-oidc
INSERT INTO oidc (name, provider, provider_url, client_id, client_secret) 
VALUES ($1, $2, $3, $4, $5);

-- name: update-oidc
UPDATE oidc 
SET name = $2, provider = $3, provider_url = $4, client_id = $5, client_secret = $6, disabled = $7, updated_at = now()
WHERE id = $1;

-- name: delete-oidc
DELETE FROM oidc WHERE id = $1;
