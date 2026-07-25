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
    auth_provider, external_id, is_active
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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

-- name: TouchUserLastLogin :exec
UPDATE users SET last_login_at = NOW() WHERE id = $1;

-- name: CountUsersWithRole :one
SELECT COUNT(*) FROM users u
JOIN user_roles ur ON ur.user_id = u.id
JOIN roles r ON r.id = ur.role_id
WHERE r.name = $1;
