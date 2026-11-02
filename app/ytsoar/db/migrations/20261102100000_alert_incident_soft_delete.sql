-- +goose Up
ALTER TABLE alerts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE incidents ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- a deleted open alert must not absorb new copies of its fingerprint
DROP INDEX IF EXISTS alerts_open_fingerprint_idx;
CREATE UNIQUE INDEX alerts_open_fingerprint_idx ON alerts (fingerprint)
    WHERE status IN ('new', 'investigating') AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS alerts_open_fingerprint_idx;
CREATE UNIQUE INDEX alerts_open_fingerprint_idx ON alerts (fingerprint)
    WHERE status IN ('new', 'investigating');

ALTER TABLE incidents DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE alerts DROP COLUMN IF EXISTS deleted_at;
