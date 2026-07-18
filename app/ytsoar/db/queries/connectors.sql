-- name: UpsertConnector :one
INSERT INTO connectors (id, name, runtime, version, checksum, uploaded_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    runtime = EXCLUDED.runtime,
    version = EXCLUDED.version,
    checksum = EXCLUDED.checksum,
    uploaded_by = EXCLUDED.uploaded_by,
    updated_at = now()
RETURNING *;

-- name: GetConnectorRecord :one
SELECT * FROM connectors WHERE id = $1;

-- name: DeleteConnectorRecord :execrows
DELETE FROM connectors WHERE id = $1;
