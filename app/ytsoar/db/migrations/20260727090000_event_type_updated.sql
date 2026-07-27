-- ALTER TYPE ... ADD VALUE cannot run inside a transaction.
-- +goose NO TRANSACTION
-- +goose Up
ALTER TYPE event_type ADD VALUE IF NOT EXISTS 'updated';

-- +goose Down
-- postgres cannot drop enum values; leaving 'updated' in place is harmless
SELECT 1;
