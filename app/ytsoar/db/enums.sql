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


CREATE TYPE alert_severity AS ENUM (
    'critical',
    'high',
    'medium',
    'low'
);

CREATE TYPE alert_status AS ENUM (
    'new',
    'investigating',
    'resolved',
    'falsepos',
    'closed'
);

CREATE TYPE incident_status AS ENUM (
    'open',
    'investigating',
    'contained',
    'resolved',
    'closed'
);

CREATE TYPE source_kind AS ENUM (
    'edr',
    'identity',
    'email',
    'firewall',
    'dlp'
);

CREATE TYPE event_type AS ENUM (
    'created',
    'status_changed',
    'escalated',
    'triage',
    'sla',
    'correlation',
    'attack_tag',
    'linked',
    'unlinked',
    'updated'
);

CREATE TYPE link_source AS ENUM (
    'manual',
    'escalate',
    'correlation'
);

CREATE TYPE sla_state AS ENUM (
    'ok',
    'warning',
    'breached',
    'met'
);
