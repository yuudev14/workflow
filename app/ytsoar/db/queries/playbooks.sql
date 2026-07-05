-- name: GetPlaybookById :one
SELECT * FROM playbooks WHERE id = $1;

-- name: GetPlaybookTriggers :many
SELECT * FROM playbook_triggers;

-- name: GetPlaybookGraphById :one
SELECT
    playbooks.*,
    (SELECT JSON_AGG(tasks.*)
        FROM tasks
        WHERE tasks.playbook_id = playbooks.id) AS tasks,
    (SELECT JSON_AGG(edges.*)
        FROM edges
        WHERE edges.playbook_id = playbooks.id) AS edges
FROM playbooks WHERE playbooks.id = $1;

-- name: CreatePlaybook :one
INSERT INTO playbooks (name, description, trigger_type)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdatePlaybook :one
UPDATE playbooks
SET
    name = CASE
        WHEN sqlc.arg('name_set')::boolean THEN sqlc.narg('name')
        ELSE name
    END,
    description = CASE
        WHEN sqlc.arg('description_set')::boolean THEN sqlc.narg('description')
        ELSE description
    END,
    trigger_type = CASE
        WHEN sqlc.arg('trigger_type_set')::boolean THEN sqlc.narg('trigger_type')
        ELSE trigger_type
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: CreatePlaybookHistory :one
INSERT INTO playbook_history (playbook_id, triggered_at, edges)
VALUES ($1, NOW(), $2)
RETURNING *;

-- name: UpdatePlaybookHistoryStatus :one
UPDATE playbook_history
SET status = $2
WHERE id = $1
RETURNING *;

-- name: UpdatePlaybookHistory :one
UPDATE playbook_history
SET
    status = CASE
        WHEN sqlc.arg('status_set')::boolean THEN sqlc.narg('status')::playbook_status
        ELSE status
    END,
    error = CASE
        WHEN sqlc.arg('error_set')::boolean THEN sqlc.narg('error')
        ELSE error
    END,
    result = sqlc.arg('result')
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: GetPlaybookHistoryById :one
SELECT playbook_history.*, to_jsonb(playbooks) AS playbook_data
FROM playbook_history
JOIN playbooks ON playbooks.id = playbook_history.playbook_id
WHERE playbook_history.id = $1;

