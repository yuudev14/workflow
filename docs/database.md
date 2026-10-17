# Database

Postgres is the only data service. This page shows the tables as diagrams. The truth is in `app/ytsoar/db/migrations/`. The SQL queries are in `app/ytsoar/db/queries/`. To change the schema, follow the "Database" section in [CLAUDE.md](../CLAUDE.md).

## 1. Map

There are four groups of tables. Groups link to each other mainly through `users` and `teams`.

```mermaid
flowchart LR
    subgraph Auth
        users
        teams
        roles
    end
    subgraph Playbooks
        playbooks
        playbook_history
    end
    subgraph Alerts
        alerts
        incidents
    end
    subgraph Connectors
        connectors
    end
    users -- "triggered_by" --> playbook_history
    users -- "assignee, author, actor" --> alerts
    teams -- "team_id" --> alerts
    users --> incidents
    playbook_history -. "playbook_run_records<br/>(module_type, record_id)" .-> alerts
    playbooks -. "tasks.connector_id" .-> connectors
```

A solid line is a foreign key. A dotted line is a link with no foreign key.

## 2. Playbooks and runs

```mermaid
erDiagram
    playbooks ||--o{ tasks : "has nodes"
    playbooks ||--o{ edges : "has links"
    tasks ||--o{ edges : "source / destination"
    playbooks ||--o{ schedulers : "cron"
    playbooks ||--o{ playbook_history : "runs"
    playbook_history ||--o{ task_history : "one row per node"
    playbook_history ||--o{ playbook_run_records : "records it acted on"
    users |o--o{ playbook_history : "triggered_by"

    playbooks {
        uuid id PK
        varchar name
        text description
        trigger_type trigger_type
        jsonb trigger_parameters
    }
    tasks {
        uuid id PK
        uuid playbook_id FK
        varchar name "unique in the playbook"
        jsonb parameters
        varchar connector_id "no FK"
        varchar operation
        varchar config
        float x
        float y
    }
    edges {
        uuid id PK
        uuid source_id FK
        uuid destination_id FK
        uuid playbook_id FK
        varchar source_handle "branch label"
    }
    playbook_history {
        uuid id PK
        uuid playbook_id FK
        playbook_status status
        text error
        jsonb result
        jsonb edges "copy of the graph"
        jsonb input "trigger data"
        trigger_type trigger_type
        uuid triggered_by FK
        timestamp triggered_at
    }
    task_history {
        uuid id PK
        uuid playbook_history_id FK
        uuid task_id "no FK"
        task_status status
        jsonb parameters
        jsonb result
        text error
        uuid_array destination_ids
    }
    playbook_run_records {
        uuid playbook_history_id PK
        text module_type PK "alert or incident"
        uuid record_id PK "no FK"
    }
    schedulers {
        uuid id PK
        uuid playbook_id FK
        varchar cron
    }
```

Why this shape:

- **Design vs history.** `playbooks`, `tasks` and `edges` are what the user edits. `playbook_history` and `task_history` are a **snapshot** of one run. They copy the graph and the parameters, so a later edit does not change an old run.
- **The queue holds only the run id.** The worker loads `playbook_history` (edges, input) and `task_history` (task snapshots) and builds the message. See [river-guide.md](river-guide.md).
- **`task_history.task_id` has no foreign key.** The user can delete a task from the design. The history stays.
- **`playbook_run_records`** is a join table. One run can act on many alerts or incidents. `record_id` points to `alerts` or `incidents` by `module_type`, so it has no foreign key.
- **Status rules.** Run: `queued` -> `in_progress` -> `success` or `failed`. Task: `pending` -> `in_progress` -> `success` or `failed`, or `pending` -> `skipped`. The database checks each change. See [diagram/statuses.md](diagram/statuses.md).
- `schedulers` exists, but no program reads it yet.

## 3. Auth

The flow is explained in [auth.md](auth.md).

```mermaid
erDiagram
    users ||--o{ user_roles : ""
    roles ||--o{ user_roles : ""
    roles ||--o{ role_permissions : "grants"
    teams ||--o{ team_members : ""
    users ||--o{ team_members : ""
    teams ||--o{ team_roles : ""
    roles ||--o{ team_roles : ""
    users ||--o{ refresh_tokens : "sessions"
    users |o--o{ audit_logs : "actor_id"

    users {
        uuid id PK
        text username UK
        text email UK
        text password_hash "null for SSO users"
        auth_provider_type auth_provider
        text external_id "provider id + subject"
        boolean is_active
        timestamp last_login_at
    }
    roles {
        uuid id PK
        text name UK
        boolean is_builtin
    }
    role_permissions {
        uuid role_id PK
        text module PK
        text action PK
    }
    user_roles {
        uuid user_id PK
        uuid role_id PK
    }
    teams {
        uuid id PK
        text name UK
    }
    team_members {
        uuid team_id PK
        uuid user_id PK
    }
    team_roles {
        uuid team_id PK
        uuid role_id PK
    }
    refresh_tokens {
        uuid id PK
        uuid user_id FK
        text token_hash UK "sha256, never the token"
        timestamp expires_at
        timestamp revoked_at
    }
    auth_providers {
        uuid id PK
        auth_provider_type type
        text name UK
        jsonb config "issuer, client, role mapping"
        boolean enabled
    }
    audit_logs {
        uuid id PK
        uuid actor_id FK
        text module
        text action
        text entity_id
        jsonb detail
    }
```

Notes:

- `module` and `action` are plain text. The allowed values are Go constants (`internal/domain/auth.go`), not an enum.
- `auth_providers` has no foreign key to `users`. A user links to a provider through `external_id` (`<provider id>|<subject>`). The unique index is on `(auth_provider, external_id)`.
- `audit_logs.actor_id` is `ON DELETE SET NULL`. The trail stays when a user goes.

## 4. Alerts and incidents

```mermaid
erDiagram
    alerts ||--o{ alert_events : "timeline"
    alerts ||--o{ alert_notes : ""
    incidents ||--o{ incident_events : "timeline"
    incidents ||--o{ incident_notes : ""
    alerts ||--o{ incident_alerts : ""
    incidents ||--o{ incident_alerts : ""
    users |o--o{ alerts : "assignee_id"
    teams |o--o{ alerts : "team_id"
    users |o--o{ incidents : "assignee_id"
    teams |o--o{ incidents : "team_id"

    alerts {
        uuid id PK
        text title
        alert_severity severity
        alert_status status
        source_kind source_kind
        jsonb payload "raw event"
        text fingerprint "dedup key"
        int dedup_count
        timestamp triaged_at
        sla_state sla_state
    }
    incidents {
        uuid id PK
        text title
        alert_severity severity
        incident_status status
        timestamp resolved_at
        sla_state sla_state
    }
    incident_alerts {
        uuid incident_id PK
        uuid alert_id PK
        link_source source "manual, escalate, correlation"
    }
    alert_events {
        uuid id PK
        uuid alert_id FK
        event_type type
        uuid actor_id FK
        jsonb body
    }
    incident_events {
        uuid id PK
        uuid incident_id FK
        event_type type
        uuid actor_id FK
        jsonb body
    }
    alert_notes {
        uuid id PK
        uuid alert_id FK
        uuid author_id FK
        text body
    }
    incident_notes {
        uuid id PK
        uuid incident_id FK
        uuid author_id FK
        text body
    }
```

Notes:

- **Dedup.** The unique partial index `alerts_open_fingerprint_idx` allows one **open** alert for each fingerprint. A repeat adds to `dedup_count` and `last_seen`. After the alert is closed, the same fingerprint can open a new alert.
- **Timeline.** `alert_events` and `incident_events` are the audit trail of each record. They use `clock_timestamp()`, not `NOW()`. `NOW()` is fixed for the whole transaction, so events of one escalation would share a time and sort in a random order.
- **Paging.** Queue indexes end with `id DESC`, for keyset paging on `(created_at, id)`. A batch insert gives many rows the same `created_at`.
- **Open set.** Partial indexes (`alerts_open_idx`, `incidents_open_idx`) keep live counts of open items cheap.
- User, team, actor and author links use `ON DELETE SET NULL`. A deleted user leaves the record.

## 5. Connectors

```mermaid
erDiagram
    connectors {
        text id PK "folder name in the tree"
        text name
        text runtime "python or node"
        text version
        text checksum "sha256 of the zip"
        text uploaded_by
        boolean enabled
    }
```

The table is an **audit row** for uploaded connectors. The folder tree is the truth for what runs. Built-in connectors (`http_request`, `condition`, snippets) have no row. `tasks.connector_id` is the connector folder name. It is text, not a foreign key. See [connector-upload.md](connector-upload.md).

## 6. Enums

| Type | Values |
|---|---|
| `playbook_status` | `queued`, `in_progress`, `success`, `failed` |
| `task_status` | `pending`, `in_progress`, `success`, `failed`, `skipped` |
| `trigger_type` | `manual`, `webhook`, `referenced`, `on_create`, `on_update`, `on_delete` |
| `auth_provider_type` | `local`, `oidc`, `ldap` (no LDAP code yet) |
| `alert_severity` | `critical`, `high`, `medium`, `low` (incidents use it too) |
| `alert_status` | `new`, `investigating`, `resolved`, `falsepos`, `closed` |
| `incident_status` | `open`, `investigating`, `contained`, `resolved`, `closed` |
| `source_kind` | `edr`, `identity`, `email`, `firewall`, `dlp` |
| `event_type` | `created`, `status_changed`, `escalated`, `triage`, `sla`, `correlation`, `attack_tag`, `linked`, `unlinked`, `updated` |
| `link_source` | `manual`, `escalate`, `correlation` |
| `sla_state` | `ok`, `warning`, `breached`, `met` |

When you add an enum value, also add it to `db/enums.sql`. sqlc reads the enums from that file.

## 7. Tables that we do not own

River (the job queue) owns the `river_*` tables. Do not query them and do not edit them. See [river-guide.md](river-guide.md).

## 8. Migrations

| File | What it adds |
|---|---|
| `20260704134500_init.sql` | Playbooks, tasks, edges, history, schedulers |
| `20260706150000_connectors.sql` | `connectors` |
| `20260706160000_task_status_skipped.sql` | `skipped` task status |
| `20260719100000_auth.sql` | Users, roles, teams, sessions, providers, audit, built-in roles |
| `20260725120000_team_roles.sql` | `team_roles` |
| `20260726090000_alerts_incidents.sql` | Alerts, incidents, events, notes, indexes |
| `20260727090000_event_type_updated.sql` | `updated` event type |
| `20260727091000_playbook_run_input.sql` | Run input, `triggered_by`, `playbook_run_records` |
| `20260801090000_playbook_status_queued.sql` | `queued` run status |
| `20260801100000_river.sql` | River tables |
