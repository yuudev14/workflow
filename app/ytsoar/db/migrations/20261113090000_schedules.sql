-- +goose Up
-- Not in production, so the table is rebuilt without a data move.
DROP TABLE IF EXISTS schedulers;
CREATE TABLE schedulers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name VARCHAR(200) NOT NULL,
    playbook_id UUID NOT NULL REFERENCES playbooks (id) ON DELETE CASCADE,
    cron TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    parameters JSONB NOT NULL DEFAULT '{}',
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX schedulers_due_idx ON schedulers (next_run_at) WHERE enabled;
CREATE INDEX schedulers_playbook_idx ON schedulers (playbook_id);

ALTER TABLE playbook_history
    ADD COLUMN schedule_id UUID REFERENCES schedulers (id) ON DELETE SET NULL;

INSERT INTO role_permissions (role_id, module, action)
SELECT role_id, 'schedules', 'delete' FROM role_permissions
WHERE module = 'schedules' AND action = 'update'
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM role_permissions WHERE module = 'schedules' AND action = 'delete'
  AND role_id IN (SELECT id FROM roles WHERE name <> 'admin');
ALTER TABLE playbook_history DROP COLUMN IF EXISTS schedule_id;
DROP TABLE IF EXISTS schedulers;
CREATE TABLE schedulers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    playbook_id UUID REFERENCES playbooks (id) ON DELETE CASCADE,
    cron VARCHAR(20),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
