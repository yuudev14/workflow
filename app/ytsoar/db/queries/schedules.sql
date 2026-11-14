-- name: CreateSchedule :one
INSERT INTO schedulers (name, playbook_id, cron, timezone, enabled, parameters, next_run_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, sqlc.narg('created_by'))
RETURNING id;

-- name: GetSchedule :one
SELECT s.*, p.name AS playbook_name, COALESCE(p.trigger_type::text, '')::text AS playbook_trigger_type,
       (p.deleted_at IS NOT NULL)::boolean AS playbook_deleted
FROM schedulers s JOIN playbooks p ON p.id = s.playbook_id
WHERE s.id = $1;

-- name: UpdateSchedule :one
UPDATE schedulers SET
    name = $2, cron = $3, timezone = $4, enabled = $5, parameters = $6, next_run_at = $7,
    updated_at = NOW()
WHERE id = $1
RETURNING id;

-- name: DeleteSchedule :execrows
DELETE FROM schedulers WHERE id = $1;

-- name: ListDueScheduleIDs :many
-- skips schedules whose playbook is deleted or no longer manual; they work again when it is
-- restored or set back to manual
SELECT s.id FROM schedulers s JOIN playbooks p ON p.id = s.playbook_id
WHERE s.enabled AND s.next_run_at <= $1
  AND p.deleted_at IS NULL AND p.trigger_type = 'manual'
ORDER BY s.next_run_at;

-- name: LockDueSchedule :one
SELECT * FROM schedulers WHERE id = $1 AND enabled AND next_run_at <= $2
FOR UPDATE SKIP LOCKED;

-- name: SetScheduleNextRun :exec
UPDATE schedulers SET last_run_at = COALESCE(sqlc.narg('last_run_at'), last_run_at), next_run_at = $2
WHERE id = $1;
