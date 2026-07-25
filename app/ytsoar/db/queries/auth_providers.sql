-- name: ListEnabledAuthProviders :many
SELECT * FROM auth_providers WHERE enabled = TRUE ORDER BY name;

-- name: ListAuthProviders :many
SELECT * FROM auth_providers ORDER BY name;

-- name: GetAuthProviderByID :one
SELECT * FROM auth_providers WHERE id = $1;

-- name: CreateAuthProvider :one
INSERT INTO auth_providers (type, name, config, enabled)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAuthProvider :one
UPDATE auth_providers
SET
    name = CASE WHEN sqlc.arg('name_set')::bool THEN sqlc.narg('name')::text ELSE name END,
    config = CASE WHEN sqlc.arg('config_set')::bool THEN sqlc.narg('config')::jsonb ELSE config END,
    enabled = CASE WHEN sqlc.arg('enabled_set')::bool THEN sqlc.narg('enabled')::bool ELSE enabled END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;
