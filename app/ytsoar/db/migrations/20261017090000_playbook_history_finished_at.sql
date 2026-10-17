-- +goose Up
-- The event loop guard needs to know if a run was still going when an event happened.
ALTER TABLE playbook_history ADD COLUMN IF NOT EXISTS finished_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE playbook_history DROP COLUMN IF EXISTS finished_at;
