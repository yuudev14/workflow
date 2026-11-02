-- +goose Up
CREATE TABLE connector_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    -- the folder id in the connectors tree; built-ins have no connectors row, so no foreign key
    connector_id TEXT NOT NULL,
    name VARCHAR(50) NOT NULL,
    -- each string value is sealed with the secret box
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    updated_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (connector_id, name)
);

CREATE UNIQUE INDEX connector_configs_default_idx ON connector_configs (connector_id) WHERE is_default;

-- +goose Down
DROP TABLE connector_configs;
