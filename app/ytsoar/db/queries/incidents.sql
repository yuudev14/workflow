-- name: GetIncidentById :one
SELECT * FROM incidents WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteIncident :execrows
UPDATE incidents SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateIncident :one
INSERT INTO incidents (title, severity, status, assignee_id, team_id, tags)
VALUES (
    sqlc.arg('title'),
    sqlc.arg('severity'),
    COALESCE(sqlc.narg('status')::incident_status, 'open'),
    sqlc.narg('assignee_id'),
    sqlc.narg('team_id'),
    COALESCE(sqlc.narg('tags')::text[], '{}')
)
RETURNING *;

-- name: UpdateIncident :one
UPDATE incidents
SET
    severity = CASE
        WHEN sqlc.arg('severity_set')::boolean THEN sqlc.narg('severity')::alert_severity
        ELSE severity
    END,
    assignee_id = CASE
        WHEN sqlc.arg('assignee_id_set')::boolean THEN sqlc.narg('assignee_id')::uuid
        ELSE assignee_id
    END,
    team_id = CASE
        WHEN sqlc.arg('team_id_set')::boolean THEN sqlc.narg('team_id')::uuid
        ELSE team_id
    END,
    tags = CASE
        WHEN sqlc.arg('tags_set')::boolean THEN COALESCE(sqlc.narg('tags')::text[], '{}')
        ELSE tags
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: UpdateIncidentStatus :one
UPDATE incidents
SET
    status = sqlc.arg('status'),
    resolved_at = CASE
        WHEN sqlc.arg('status')::incident_status IN ('resolved', 'closed')
            THEN COALESCE(resolved_at, NOW())
        ELSE NULL
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: InsertIncidentEvent :one
INSERT INTO incident_events (incident_id, type, actor_id, body, playbook_run_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListIncidentEvents :many
SELECT e.*, u.username AS actor_username, p.name AS playbook_name
FROM incident_events e
LEFT JOIN users u ON u.id = e.actor_id
LEFT JOIN playbook_history h ON h.id = e.playbook_run_id
LEFT JOIN playbooks p ON p.id = h.playbook_id
WHERE e.incident_id = $1
ORDER BY e.created_at, e.id;

-- name: InsertIncidentAlert :execrows
INSERT INTO incident_alerts (incident_id, alert_id, source)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: DeleteIncidentAlert :execrows
DELETE FROM incident_alerts WHERE incident_id = $1 AND alert_id = $2;

-- name: ListIncidentAlerts :many
SELECT a.id, a.title, a.severity, a.source_kind, a.reporter, ia.source AS link_source
FROM incident_alerts ia
JOIN alerts a ON a.id = ia.alert_id
WHERE ia.incident_id = $1
ORDER BY a.created_at DESC, a.id DESC;

-- name: InsertIncidentNote :one
INSERT INTO incident_notes (incident_id, author_id, body, playbook_run_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetIncidentNoteById :one
SELECT * FROM incident_notes WHERE id = $1;

-- name: UpdateIncidentNote :one
UPDATE incident_notes
SET body = sqlc.arg('body'),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteIncidentNote :execrows
DELETE FROM incident_notes WHERE id = $1;

-- name: ListIncidentNotes :many
SELECT n.*, u.username AS author_username, p.name AS playbook_name
FROM incident_notes n
LEFT JOIN users u ON u.id = n.author_id
LEFT JOIN playbook_history h ON h.id = n.playbook_run_id
LEFT JOIN playbooks p ON p.id = h.playbook_id
WHERE n.incident_id = $1
ORDER BY n.created_at, n.id;