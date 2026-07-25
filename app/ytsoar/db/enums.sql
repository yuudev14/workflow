-- put you enums here as it cannot be retrived from the migration file because of DO clause

CREATE TYPE playbook_status AS ENUM (
    'in_progress',
    'success',
    'failed'
);

CREATE TYPE task_status AS ENUM (
    'pending',
    'in_progress',
    'success',
    'failed',
    'skipped'
);

CREATE TYPE trigger_type AS ENUM (
    'manual',
    'webhook',
    'referenced',
    'on_create',
    'on_update',
    'on_delete'
);

CREATE TYPE auth_provider_type AS ENUM (
    'local',
    'oidc',
    'ldap'
);
