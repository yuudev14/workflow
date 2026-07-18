-- name: UpsertEdge :one
INSERT INTO edges (destination_id, source_id, playbook_id, source_handle, destination_handle)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (destination_id, source_id) DO UPDATE
SET source_handle = EXCLUDED.source_handle,
    destination_handle = EXCLUDED.destination_handle
RETURNING *;

-- name: DeleteEdges :exec
DELETE FROM edges WHERE id = ANY(sqlc.arg('ids')::uuid[]);

-- name: DeleteAllPlaybookEdges :exec
DELETE FROM edges
WHERE destination_id IN (SELECT id FROM tasks WHERE tasks.playbook_id = $1)
   OR source_id IN (SELECT id FROM tasks WHERE tasks.playbook_id = $1);

-- name: GetEdgesByPlaybookId :many
SELECT e.*, t1.name AS source_task_name, t2.name AS destination_task_name
FROM edges e
JOIN tasks t1 ON e.source_id = t1.id
JOIN tasks t2 ON e.destination_id = t2.id
WHERE t1.playbook_id = $1;
