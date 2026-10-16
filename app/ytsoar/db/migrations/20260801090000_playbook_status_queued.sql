-- +goose NO TRANSACTION
-- +goose Up
-- a run waits as 'queued' until a worker moves it to 'in_progress'.
-- ADD VALUE cannot run inside a transaction block.
ALTER TYPE playbook_status ADD VALUE IF NOT EXISTS 'queued' BEFORE 'in_progress';

-- +goose Down
-- postgres cannot drop enum values; leaving 'queued' in place is harmless
SELECT 1;
