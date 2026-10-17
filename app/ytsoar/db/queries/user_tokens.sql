-- name: UpsertUserToken :execrows
-- a user has at most one link for each purpose. The WHERE skips the insert when the last
-- link is newer than the limit, so a user cannot get many emails.
INSERT INTO user_tokens (token_hash, user_id, purpose, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, purpose) DO UPDATE
SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at
WHERE user_tokens.created_at < sqlc.arg('resend_after');

-- name: TakeUserToken :one
DELETE FROM user_tokens WHERE token_hash = $1
RETURNING *;

-- name: DeleteExpiredUserTokens :exec
DELETE FROM user_tokens WHERE expires_at < $1;
