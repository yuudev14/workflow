-- +goose Up
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE auth_provider_type AS ENUM (
        'local',
        'oidc',
        'ldap'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd


CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    -- null for users that authenticate against an external provider
    password_hash TEXT,
    first_name TEXT,
    last_name TEXT,
    auth_provider auth_provider_type NOT NULL DEFAULT 'local',
    external_id TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_provider_external_id_idx
    ON users (auth_provider, external_id)
    WHERE external_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_builtin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    module TEXT NOT NULL,
    action TEXT NOT NULL,

    PRIMARY KEY (role_id, module, action)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,

    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS team_members (
    team_id UUID NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE TABLE IF NOT EXISTS auth_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    type auth_provider_type NOT NULL,
    name TEXT NOT NULL UNIQUE,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    actor_id UUID REFERENCES users (id) ON DELETE SET NULL,
    module TEXT NOT NULL,
    action TEXT NOT NULL,
    entity_id TEXT,
    detail JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS audit_logs_created_at_idx ON audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_actor_id_idx ON audit_logs (actor_id);

INSERT INTO roles (name, description, is_builtin) VALUES
    ('admin', 'Full access to every module', TRUE),
    ('analyst', 'Day-to-day SOC work', TRUE),
    ('viewer', 'Read-only access', TRUE)
ON CONFLICT (name) DO NOTHING;

-- admin: every module x every action
INSERT INTO role_permissions (role_id, module, action)
SELECT r.id, m.module, a.action
FROM roles r
CROSS JOIN (VALUES ('playbooks'), ('alerts'), ('incidents'), ('connectors'),
                   ('agent'), ('schedules'), ('settings')) AS m(module)
CROSS JOIN (VALUES ('read'), ('create'), ('update'), ('delete'),
                   ('execute'), ('approve')) AS a(action)
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- analyst
INSERT INTO role_permissions (role_id, module, action)
SELECT r.id, p.module, p.action
FROM roles r
CROSS JOIN (VALUES
    ('playbooks', 'read'), ('playbooks', 'create'), ('playbooks', 'update'), ('playbooks', 'execute'),
    ('alerts', 'read'), ('alerts', 'create'), ('alerts', 'update'), ('alerts', 'approve'),
    ('incidents', 'read'), ('incidents', 'create'), ('incidents', 'update'),
    ('connectors', 'read'),
    ('agent', 'read'), ('agent', 'execute'),
    ('schedules', 'read'), ('schedules', 'create'), ('schedules', 'update')
) AS p(module, action)
WHERE r.name = 'analyst'
ON CONFLICT DO NOTHING;

-- viewer: read on everything except settings
INSERT INTO role_permissions (role_id, module, action)
SELECT r.id, m.module, 'read'
FROM roles r
CROSS JOIN (VALUES ('playbooks'), ('alerts'), ('incidents'), ('connectors'),
                   ('agent'), ('schedules')) AS m(module)
WHERE r.name = 'viewer'
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS auth_providers;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS team_members;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS auth_provider_type;
