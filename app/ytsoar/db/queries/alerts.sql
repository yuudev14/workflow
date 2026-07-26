-- name: GetAlertById :one
SELECT * FROM alerts WHERE id = $1;

-- Dedup lives in the unique partial index alerts_open_fingerprint_idx, so the
-- ON CONFLICT predicate must repeat that index's WHERE verbatim to target it.
-- xmax is 0 only on a freshly inserted row, which is how the caller tells a new
-- alert from a deduped one without a second query.
-- name: UpsertAlert :one
INSERT INTO alerts (
    title, severity, source_kind, reporter, assignee_id, team_id,
    payload, tags, fingerprint, created_at, last_seen
) VALUES (
    sqlc.arg('title'),
    sqlc.arg('severity'),
    sqlc.arg('source_kind'),
    sqlc.narg('reporter'),
    sqlc.narg('assignee_id'),
    sqlc.narg('team_id'),
    sqlc.arg('payload'),
    sqlc.arg('tags'),
    sqlc.arg('fingerprint'),
    COALESCE(sqlc.narg('created_at')::timestamp, NOW()),
    COALESCE(sqlc.narg('created_at')::timestamp, NOW())
)
ON CONFLICT (fingerprint) WHERE status IN ('new', 'investigating')
DO UPDATE SET
    dedup_count = alerts.dedup_count + 1,
    last_seen = NOW(),
    updated_at = NOW()
RETURNING *, (xmax = 0) AS inserted;

-- name: UpdateAlert :one
UPDATE alerts
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


-- name: UpdateAlertStatus :one
UPDATE alerts
SET
    status = sqlc.arg('status'),
    triaged_at = CASE
        WHEN triaged_at IS NULL AND status = 'new' AND sqlc.arg('status')::alert_status <> 'new'
        THEN NOW()
        ELSE triaged_at
    END,
    closure_note = CASE
        WHEN sqlc.arg('status')::alert_status IN ('new', 'investigating') THEN NULL
        ELSE COALESCE(sqlc.narg('closure_note')::text, closure_note)
    END,
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: InsertAlertEvent :one
INSERT INTO alert_events (alert_id, type, actor_id, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListAlertEvents :many
SELECT e.*, u.username AS actor_username
FROM alert_events e
LEFT JOIN users u ON u.id = e.actor_id
WHERE e.alert_id = $1
ORDER BY e.created_at, e.id;

-- name: InsertAlertNote :one
INSERT INTO alert_notes (alert_id, author_id, body)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetAlertNoteById :one
SELECT * FROM alert_notes WHERE id = $1;

-- name: UpdateAlertNote :one
UPDATE alert_notes
SET body = sqlc.arg('body'),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteAlertNote :execrows
DELETE FROM alert_notes WHERE id = $1;

-- name: ListAlertNotes :many
SELECT n.*, u.username AS author_username
FROM alert_notes n
LEFT JOIN users u ON u.id = n.author_id
WHERE n.alert_id = $1
ORDER BY n.created_at, n.id;