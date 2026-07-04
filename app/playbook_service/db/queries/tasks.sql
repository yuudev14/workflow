-- name: GetTasksByPlaybookId :many
SELECT * FROM tasks WHERE playbook_id = $1;

-- name: UpsertTask :one
INSERT INTO tasks (playbook_id, name, description, parameters, config, connector_name, connector_id, operation, x, y)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (playbook_id, name) DO UPDATE
SET description = EXCLUDED.description,
    parameters = EXCLUDED.parameters,
    config = EXCLUDED.config,
    connector_name = EXCLUDED.connector_name,
    connector_id = EXCLUDED.connector_id,
    operation = EXCLUDED.operation,
    x = EXCLUDED.x,
    y = EXCLUDED.y,
    updated_at = NOW()
RETURNING *;

-- name: DeleteTasks :exec
DELETE FROM tasks WHERE id = ANY(sqlc.arg('ids')::uuid[]);

-- name: CreateTaskHistory :one
INSERT INTO task_history (
    playbook_history_id, task_id, triggered_at, name, description, parameters,
    config, x, y, connector_name, connector_id, operation, destination_ids
)
VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING *;

-- name: UpdateTaskStatus :one
UPDATE task_history
SET status = $3
WHERE playbook_history_id = $1 AND task_id = $2
RETURNING *;

-- name: UpdateTaskHistory :one
UPDATE task_history
SET
    name = sqlc.arg('name'),
    description = sqlc.arg('description'),
    parameters = sqlc.arg('parameters'),
    result = sqlc.arg('result'),
    x = sqlc.arg('x'),
    y = sqlc.arg('y'),
    operation = sqlc.arg('operation'),
    status = CASE
        WHEN sqlc.arg('status_set')::boolean THEN sqlc.narg('status')::task_status
        ELSE status
    END,
    error = CASE
        WHEN sqlc.arg('error_set')::boolean THEN sqlc.narg('error')
        ELSE error
    END,
    connector_name = CASE
        WHEN sqlc.arg('connector_name_set')::boolean THEN sqlc.narg('connector_name')
        ELSE connector_name
    END,
    connector_id = CASE
        WHEN sqlc.arg('connector_id_set')::boolean THEN sqlc.narg('connector_id')
        ELSE connector_id
    END,
    config = CASE
        WHEN sqlc.arg('config_set')::boolean THEN sqlc.narg('config')
        ELSE config
    END
WHERE playbook_history_id = sqlc.arg('playbook_history_id') AND task_id = sqlc.arg('task_id')
RETURNING *;

-- name: GetTaskHistoryByPlaybookHistoryId :many
SELECT * FROM task_history WHERE playbook_history_id = $1;

