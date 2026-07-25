-- name: InsertAuditLog :one
INSERT INTO audit_logs (actor_id, module, action, entity_id, detail)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;
