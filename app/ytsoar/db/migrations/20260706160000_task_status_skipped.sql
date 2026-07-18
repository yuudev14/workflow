-- +goose NO TRANSACTION
-- +goose Up
-- conditional branching: nodes on a branch that was not taken complete as
-- 'skipped'. ADD VALUE cannot run inside a transaction block.
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'skipped';

-- +goose Down
-- postgres cannot drop enum values; leaving 'skipped' in place is harmless
SELECT 1;
