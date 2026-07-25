-- +goose Up
CREATE TABLE IF NOT EXISTS team_roles (
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,

    PRIMARY KEY (team_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_team_roles_role_id ON team_roles (role_id);

-- +goose Down
DROP INDEX IF EXISTS idx_team_roles_role_id;
DROP TABLE IF EXISTS team_roles;