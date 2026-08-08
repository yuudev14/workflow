-- +goose Up
ALTER TABLE playbook_history
    ADD COLUMN IF NOT EXISTS trigger_type trigger_type,
    ADD COLUMN IF NOT EXISTS triggered_by UUID REFERENCES users (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS input JSONB;

-- A join table rather than columns on playbook_history: one run can act on N
-- records, which a column cannot express. module_type holds the singular names
-- already used as module.events routing keys ('alert', 'incident').
CREATE TABLE IF NOT EXISTS playbook_run_records (
    playbook_history_id UUID NOT NULL REFERENCES playbook_history (id) ON DELETE CASCADE,
    module_type TEXT NOT NULL,
    record_id UUID NOT NULL,
    PRIMARY KEY (playbook_history_id, module_type, record_id)
);

-- Drives the "runs on this record" read from the alert/incident side, which is
-- the direction the UI asks in.
CREATE INDEX IF NOT EXISTS playbook_run_records_record_idx
    ON playbook_run_records (module_type, record_id);

-- +goose Down
DROP TABLE IF EXISTS playbook_run_records;

ALTER TABLE playbook_history
    DROP COLUMN IF EXISTS input,
    DROP COLUMN IF EXISTS triggered_by,
    DROP COLUMN IF EXISTS trigger_type;
