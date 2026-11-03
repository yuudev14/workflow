-- +goose Up
-- the run that made the change when no user did; the playbook name comes from a join
ALTER TABLE alert_events ADD COLUMN playbook_run_id UUID REFERENCES playbook_history(id) ON DELETE SET NULL;
ALTER TABLE alert_notes ADD COLUMN playbook_run_id UUID REFERENCES playbook_history(id) ON DELETE SET NULL;
ALTER TABLE incident_events ADD COLUMN playbook_run_id UUID REFERENCES playbook_history(id) ON DELETE SET NULL;
ALTER TABLE incident_notes ADD COLUMN playbook_run_id UUID REFERENCES playbook_history(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE incident_notes DROP COLUMN playbook_run_id;
ALTER TABLE incident_events DROP COLUMN playbook_run_id;
ALTER TABLE alert_notes DROP COLUMN playbook_run_id;
ALTER TABLE alert_events DROP COLUMN playbook_run_id;
