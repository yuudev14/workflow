-- name: InsertAPIKey :one
INSERT INTO api_keys (user_id, name, key_hash, prefix, created_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetActiveAPIKeyByHash :one
-- the join drops the keys of a deactivated user, so deactivation stops them at once
SELECT sqlc.embed(k), u.username
FROM api_keys k
JOIN users u ON u.id = k.user_id
WHERE k.key_hash = $1 AND u.is_active;

-- name: ListAPIKeys :many
SELECT sqlc.embed(k), u.username
FROM api_keys k
JOIN users u ON u.id = k.user_id
WHERE sqlc.narg('user_id')::uuid IS NULL OR k.user_id = sqlc.narg('user_id')
ORDER BY k.created_at DESC;

-- name: TouchAPIKey :exec
UPDATE api_keys SET last_used_at = $2 WHERE id = $1;

-- name: DeleteAPIKey :execrows
DELETE FROM api_keys WHERE id = $1;
