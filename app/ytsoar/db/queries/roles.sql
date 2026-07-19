-- name: GetRoleById :one
SELECT * FROM roles WHERE id = $1;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE name = $1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY name;

-- name: CreateRole :one
INSERT INTO roles (name, description) VALUES ($1, $2) RETURNING *;

-- name: UpdateRole :one
UPDATE roles
SET
    name = CASE
        WHEN sqlc.arg('name_set')::boolean THEN sqlc.narg('name')
        ELSE name
    END,
    description = CASE
        WHEN sqlc.arg('description_set')::boolean THEN sqlc.narg('description')
        ELSE description
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteRole :execrows
DELETE FROM roles WHERE id = $1 AND is_builtin = FALSE;

-- name: ListRolePermissions :many
SELECT module, action FROM role_permissions WHERE role_id = $1 ORDER BY module, action;

-- name: InsertRolePermission :exec
INSERT INTO role_permissions (role_id, module, action)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: ListRolesForUser :many
SELECT r.* FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1
ORDER BY r.name;

-- name: InsertUserRole :exec
INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteUserRoles :exec
DELETE FROM user_roles WHERE user_id = $1;

-- name: ListPermissionsForUser :many
SELECT DISTINCT rp.module, rp.action
FROM role_permissions rp
JOIN user_roles ur ON ur.role_id = rp.role_id
JOIN users u ON u.id = ur.user_id
WHERE u.id = $1 AND u.is_active = TRUE
ORDER BY rp.module, rp.action;
