-- name: GetPlaybookById :one
SELECT * FROM playbooks WHERE id = $1;

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
INSERT INTO playbooks (name, description, trigger_type, trigger_parameters)
VALUES ($1, $2, $3, $4)
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
    trigger_parameters = CASE
        WHEN sqlc.arg('trigger_parameters_set')::boolean THEN sqlc.narg('trigger_parameters')
        ELSE trigger_parameters
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: CreatePlaybookHistory :one
INSERT INTO playbook_history (playbook_id, triggered_at, edges, trigger_type, triggered_by, input, status)
VALUES ($1, NOW(), $2, sqlc.narg('trigger_type'), sqlc.narg('triggered_by'), sqlc.narg('input'), 'queued')
RETURNING *;

-- name: CreatePlaybookRunRecord :exec
INSERT INTO playbook_run_records (playbook_history_id, module_type, record_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: UpdatePlaybookHistoryStatus :one
UPDATE playbook_history
SET status = $2,
    finished_at = CASE WHEN $2::playbook_status IN ('success', 'failed') THEN NOW() ELSE finished_at END
WHERE id = $1 AND status::text = ANY(sqlc.arg('allowed_from')::text[])
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
    result = sqlc.arg('result'),
    finished_at = CASE
        WHEN sqlc.arg('status_set')::boolean AND sqlc.narg('status')::playbook_status IN ('success', 'failed') THEN NOW()
        ELSE finished_at
    END
WHERE id = sqlc.arg('id')
  -- State rule: the new status must be allowed from the current status.
  AND (NOT sqlc.arg('status_set')::boolean OR status::text = ANY(sqlc.arg('allowed_from')::text[]))
RETURNING *;

-- name: GetPlaybookHistoryById :one
SELECT playbook_history.*, to_jsonb(playbooks) AS playbook_data
FROM playbook_history
JOIN playbooks ON playbooks.id = playbook_history.playbook_id
WHERE playbook_history.id = $1;


-- name: ListEventPlaybookIDs :many
-- A playbook is skipped when its run on this record was going at the time of the event,
-- so a run that updates its own record does not start itself again.
SELECT p.id FROM playbooks p
WHERE p.trigger_type = sqlc.arg('trigger_type')::trigger_type
  AND p.trigger_parameters ->> 'module' = sqlc.arg('module')::text
  AND NOT EXISTS (
    SELECT 1 FROM playbook_history h
    JOIN playbook_run_records r ON r.playbook_history_id = h.id
    WHERE h.playbook_id = p.id
      AND r.module_type = sqlc.arg('module')::text
      AND r.record_id = sqlc.arg('record_id')
      AND h.triggered_at <= sqlc.arg('occurred_at')::timestamptz
      AND (h.status IN ('queued', 'in_progress') OR h.finished_at >= sqlc.arg('occurred_at')::timestamptz)
  );
