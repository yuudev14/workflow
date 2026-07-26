-- +goose Up

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE alert_severity AS ENUM (
        'critical',
        'high',
        'medium',
        'low'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE alert_status AS ENUM (
        'new',
        'investigating',
        'resolved',
        'falsepos',
        'closed'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE incident_status AS ENUM (
        'open',
        'investigating',
        'contained',
        'resolved',
        'closed'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE source_kind AS ENUM (
        'edr',
        'identity',
        'email',
        'firewall',
        'dlp'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE event_type AS ENUM (
        'created',
        'status_changed',
        'escalated',
        'triage',
        'sla',
        'correlation',
        'attack_tag',
        'linked',
        'unlinked'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE link_source AS ENUM (
        'manual',
        'escalate',
        'correlation'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd

-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE sla_state AS ENUM (
        'ok',
        'warning',
        'breached',
        'met'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
-- +goose StatementEnd


CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    title TEXT NOT NULL,
    severity alert_severity NOT NULL,
    status alert_status NOT NULL DEFAULT 'new',
    source_kind source_kind NOT NULL,
    reporter TEXT,
    assignee_id UUID REFERENCES users (id) ON DELETE SET NULL,
    team_id UUID REFERENCES teams (id) ON DELETE SET NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    tags TEXT[] NOT NULL DEFAULT '{}',
    triage JSONB,
    closure_note TEXT,
    fingerprint TEXT NOT NULL,
    dedup_count INTEGER NOT NULL DEFAULT 1,
    last_seen TIMESTAMP NOT NULL DEFAULT NOW(),
    triaged_at TIMESTAMP,
    sla_deadline TIMESTAMP,
    sla_breached_at TIMESTAMP,
    sla_state sla_state NOT NULL DEFAULT 'ok',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    title TEXT NOT NULL,
    severity alert_severity NOT NULL,
    status incident_status NOT NULL DEFAULT 'open',
    assignee_id UUID REFERENCES users (id) ON DELETE SET NULL,
    team_id UUID REFERENCES teams (id) ON DELETE SET NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    resolved_at TIMESTAMP,
    sla_deadline TIMESTAMP,
    sla_breached_at TIMESTAMP,
    sla_state sla_state NOT NULL DEFAULT 'ok',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- clock_timestamp(), not NOW(): NOW() is fixed at transaction start, so the
-- several events one escalation writes would share a timestamp and the timeline
-- would order them by random uuid. clock_timestamp() advances mid-transaction.
CREATE TABLE IF NOT EXISTS alert_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    alert_id UUID NOT NULL REFERENCES alerts (id) ON DELETE CASCADE,
    type event_type NOT NULL,
    actor_id UUID REFERENCES users (id) ON DELETE SET NULL,
    body JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS incident_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    incident_id UUID NOT NULL REFERENCES incidents (id) ON DELETE CASCADE,
    type event_type NOT NULL,
    actor_id UUID REFERENCES users (id) ON DELETE SET NULL,
    body JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP NOT NULL DEFAULT clock_timestamp()
);

CREATE TABLE IF NOT EXISTS incident_alerts (
    incident_id UUID NOT NULL REFERENCES incidents (id) ON DELETE CASCADE,
    alert_id UUID NOT NULL REFERENCES alerts (id) ON DELETE CASCADE,
    source link_source NOT NULL DEFAULT 'manual',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (incident_id, alert_id)
);

CREATE TABLE IF NOT EXISTS alert_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    alert_id UUID NOT NULL REFERENCES alerts (id) ON DELETE CASCADE,
    author_id UUID REFERENCES users (id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS incident_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    incident_id UUID NOT NULL REFERENCES incidents (id) ON DELETE CASCADE,
    author_id UUID REFERENCES users (id) ON DELETE SET NULL,
    body TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);


-- Queue indexes. The trailing `id DESC` is not decoration: keyset pagination
-- compares the row value `(created_at, id) < ($ts, $id)` because NOW() is
-- fixed at transaction start, so a batch insert gives every row an identical
-- created_at and a timestamp-only cursor cannot page through it. Including id
-- keeps that predicate a pure index scan.
CREATE INDEX IF NOT EXISTS alerts_status_created_idx ON alerts (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS alerts_severity_created_idx ON alerts (severity, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS alerts_assignee_created_idx ON alerts (assignee_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS alerts_team_created_idx ON alerts (team_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS alerts_created_idx ON alerts (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS incidents_status_created_idx ON incidents (status, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS incidents_severity_created_idx ON incidents (severity, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS incidents_assignee_created_idx ON incidents (assignee_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS incidents_team_created_idx ON incidents (team_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS incidents_created_idx ON incidents (created_at DESC, id DESC);

-- The open set is small by definition even when the table is huge, so live
-- counts over these partial indexes stay cheap and never drift the way
-- incremented counters do.
CREATE INDEX IF NOT EXISTS alerts_open_idx ON alerts (severity)
    WHERE status IN ('new', 'investigating');
CREATE INDEX IF NOT EXISTS incidents_open_idx ON incidents (status, severity)
    WHERE status NOT IN ('resolved', 'closed');

-- Dedup authority. Scoped to open alerts so a fingerprint can recur once the
-- previous occurrence is closed. ON CONFLICT must repeat this predicate
-- verbatim to target the index.
CREATE UNIQUE INDEX IF NOT EXISTS alerts_open_fingerprint_idx ON alerts (fingerprint)
    WHERE status IN ('new', 'investigating');

-- Duration reads: median time-to-triage and average time-to-resolve scan only
-- the window's rows, so the index matters more than the table size.
CREATE INDEX IF NOT EXISTS alerts_triaged_at_idx ON alerts (triaged_at)
    WHERE triaged_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS incidents_resolved_at_idx ON incidents (resolved_at)
    WHERE resolved_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS alert_events_alert_created_idx ON alert_events (alert_id, created_at);
CREATE INDEX IF NOT EXISTS incident_events_incident_created_idx ON incident_events (incident_id, created_at);
CREATE INDEX IF NOT EXISTS alert_notes_alert_created_idx ON alert_notes (alert_id, created_at);
CREATE INDEX IF NOT EXISTS incident_notes_incident_created_idx ON incident_notes (incident_id, created_at);
CREATE INDEX IF NOT EXISTS incident_alerts_alert_idx ON incident_alerts (alert_id);

-- +goose Down
DROP TABLE IF EXISTS incident_notes;
DROP TABLE IF EXISTS alert_notes;
DROP TABLE IF EXISTS incident_alerts;
DROP TABLE IF EXISTS incident_events;
DROP TABLE IF EXISTS alert_events;
DROP TABLE IF EXISTS incidents;
DROP TABLE IF EXISTS alerts;
DROP TYPE IF EXISTS sla_state;
DROP TYPE IF EXISTS link_source;
DROP TYPE IF EXISTS event_type;
DROP TYPE IF EXISTS source_kind;
DROP TYPE IF EXISTS incident_status;
DROP TYPE IF EXISTS alert_status;
DROP TYPE IF EXISTS alert_severity;