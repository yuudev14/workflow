-- +goose Up
ALTER TABLE alert_events ADD COLUMN actor_label TEXT;
ALTER TABLE alert_notes ADD COLUMN actor_label TEXT;
ALTER TABLE incident_events ADD COLUMN actor_label TEXT;
ALTER TABLE incident_notes ADD COLUMN actor_label TEXT;

-- +goose Down
ALTER TABLE incident_notes DROP COLUMN actor_label;
ALTER TABLE incident_events DROP COLUMN actor_label;
ALTER TABLE alert_notes DROP COLUMN actor_label;
ALTER TABLE alert_events DROP COLUMN actor_label;
