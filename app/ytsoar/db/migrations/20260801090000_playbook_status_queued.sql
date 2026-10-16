-- +goose NO TRANSACTION
-- +goose Up
-- A run stays 'queued' until a worker sets it to 'in_progress'.
-- ADD VALUE does not run inside a transaction block.
ALTER TYPE playbook_status ADD VALUE IF NOT EXISTS 'queued' BEFORE 'in_progress';

-- +goose Down
-- Postgres cannot remove an enum value. The 'queued' value stays. This is safe.
SELECT 1;
