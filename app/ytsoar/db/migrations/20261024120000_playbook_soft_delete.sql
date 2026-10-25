-- +goose Up
ALTER TABLE playbooks ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Playbooks are only soft deleted. RESTRICT keeps a manual DELETE from removing the run history.
ALTER TABLE playbook_history DROP CONSTRAINT IF EXISTS playbook_history_playbook_id_fkey;
ALTER TABLE playbook_history ADD CONSTRAINT playbook_history_playbook_id_fkey
    FOREIGN KEY (playbook_id) REFERENCES playbooks (id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE playbook_history DROP CONSTRAINT IF EXISTS playbook_history_playbook_id_fkey;
ALTER TABLE playbook_history ADD CONSTRAINT playbook_history_playbook_id_fkey
    FOREIGN KEY (playbook_id) REFERENCES playbooks (id) ON DELETE CASCADE;

ALTER TABLE playbooks DROP COLUMN IF EXISTS deleted_at;
