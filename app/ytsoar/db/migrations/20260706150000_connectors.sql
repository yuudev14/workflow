-- +goose Up
-- Audit + control row for every uploaded connector. The filesystem tree
-- stays the source of truth for metadata/execution; this table records who
-- uploaded what (checksum of the zip) and carries the disable flag.
CREATE TABLE connectors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    runtime TEXT NOT NULL DEFAULT 'python',
    version TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL,
    uploaded_by TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE connectors;
