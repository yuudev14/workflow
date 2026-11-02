-- name: ListConnectorConfigs :many
SELECT * FROM connector_configs WHERE connector_id = $1 ORDER BY name;

-- name: GetConnectorConfig :one
SELECT * FROM connector_configs WHERE connector_id = $1 AND id = $2;

-- name: GetConnectorConfigForRun :one
-- a step without a config name uses the default config
SELECT * FROM connector_configs
WHERE connector_id = sqlc.arg('connector_id')
  AND (name = sqlc.narg('name') OR (sqlc.narg('name') IS NULL AND is_default));

-- name: InsertConnectorConfig :one
INSERT INTO connector_configs (connector_id, name, config, is_default, created_by, updated_by)
VALUES ($1, $2, $3, $4, $5, $5)
RETURNING *;

-- name: UpdateConnectorConfig :one
UPDATE connector_configs SET
    name = COALESCE(sqlc.narg('name'), name),
    config = COALESCE(sqlc.narg('config'), config),
    is_default = COALESCE(sqlc.narg('is_default'), is_default),
    updated_by = sqlc.narg('updated_by'),
    updated_at = now()
WHERE connector_id = sqlc.arg('connector_id') AND id = sqlc.arg('id')
RETURNING *;

-- name: ClearDefaultConnectorConfig :exec
UPDATE connector_configs SET is_default = FALSE WHERE connector_id = $1 AND is_default;

-- name: DeleteConnectorConfig :execrows
DELETE FROM connector_configs WHERE connector_id = $1 AND id = $2;
