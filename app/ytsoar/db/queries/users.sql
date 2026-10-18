-- name: GetUserById :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByExternalId :one
SELECT * FROM users WHERE auth_provider = $1 AND external_id = $2;

-- name: CreateUser :one
INSERT INTO users (
    username, email, password_hash, first_name, last_name,
    auth_provider, external_id, is_active, email_verified_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET
    email = CASE
        WHEN sqlc.arg('email_set')::boolean THEN sqlc.narg('email')
        ELSE email
    END,
    first_name = CASE
        WHEN sqlc.arg('first_name_set')::boolean THEN sqlc.narg('first_name')
        ELSE first_name
    END,
    last_name = CASE
        WHEN sqlc.arg('last_name_set')::boolean THEN sqlc.narg('last_name')
        ELSE last_name
    END,
    is_active = CASE
        WHEN sqlc.arg('is_active_set')::boolean THEN sqlc.narg('is_active')
        ELSE is_active
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: MarkUserEmailVerified :exec
UPDATE users SET email_verified_at = $2 WHERE id = $1 AND email_verified_at IS NULL;

-- name: TouchUserLastLogin :exec
UPDATE users SET last_login_at = NOW() WHERE id = $1;

-- name: CountUsersWithRole :one
SELECT COUNT(*) FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id
WHERE r.name = $1;

-- a WHERE ... sent_at < resend_after skips the update when the last link is too new, so a user cannot get many emails

-- name: SetUserInvitationToken :execrows
UPDATE users SET invitation_token = $2, invitation_sent_at = $3
WHERE id = $1 AND (invitation_sent_at IS NULL OR invitation_sent_at < sqlc.arg('resend_after'));

-- name: SetUserResetPasswordToken :execrows
UPDATE users SET reset_password_token = $2, reset_password_sent_at = $3
WHERE id = $1 AND (reset_password_sent_at IS NULL OR reset_password_sent_at < sqlc.arg('resend_after'));

-- the Take queries clear the token and return its old values, so a link works one time

-- name: TakeUserInvitationToken :one
UPDATE users u SET invitation_token = NULL, invitation_sent_at = NULL
FROM (SELECT o.id, o.invitation_sent_at FROM users o WHERE o.invitation_token = $1 FOR UPDATE) old
WHERE u.id = old.id
RETURNING old.id, old.invitation_sent_at;

-- name: TakeUserResetPasswordToken :one
UPDATE users u SET reset_password_token = NULL, reset_password_sent_at = NULL
FROM (SELECT o.id, o.reset_password_sent_at FROM users o WHERE o.reset_password_token = $1 FOR UPDATE) old
WHERE u.id = old.id
RETURNING old.id, old.reset_password_sent_at;
