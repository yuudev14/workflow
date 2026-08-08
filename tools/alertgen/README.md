# alertgen - dev/demo alert generator

Populates the alerts and incidents queues with realistic, ECS-shaped data so the
UI and the summary dashboards have something to render. **Dev tool only** - it
lives outside `app/ytsoar`, nothing imports it, and it has no production role.

Python stdlib only: no venv, no `requirements.txt`. Run it on the host.

## Prerequisites

1. The dev stack is up and migrated:

   ```bash
   docker compose -f docker/dev.docker-compose.yml up -d
   ```

2. Admin credentials. Ingest API-key auth does not exist yet, so the generator
   logs in as a user and lets a cookie jar carry the httpOnly cookies. Defaults
   are `admin` / `admin123!`; override with env vars if yours differ:

   ```bash
   export YTSOAR_API=http://localhost:8080
   export ADMIN_USERNAME=admin
   export ADMIN_PASSWORD='admin123!'
   ```

## Commands

```bash
make seed-alerts          # ~40 alerts, all 5 source kinds, all severities, + 3 incidents
make seed-alerts N=200    # bulk via POST /api/alerts/v1/batch - exercises keyset paging
make drip-alerts          # one alert every ~20s until Ctrl-C - watch the UI update live
make storm-alerts         # 200 byte-identical alerts - must collapse to ONE row
make storm-alerts N=50    # same, smaller
```

Or call the script directly for the flags the make targets don't expose
(`--days`, `--incidents`, `--interval`, `--seed`):

```bash
python3 tools/alertgen/alertgen.py seed --count 80 --days 30 --incidents 5
python3 tools/alertgen/alertgen.py --seed 42 seed    # reproducible data
```

### seed

`created_at` is spread across the last 14 days (weighted toward recent) so the
`volume[]` summary bucket chart has a shape instead of one spike. At `N >= 50`
it switches to the batch endpoint - one transaction, so every row shares a
`created_at`, which is exactly the case the keyset cursor's `id` tiebreaker
exists for.

It also escalates 3 alerts into incidents, adds a note to one, and moves two
along the status stepper, so the incidents queue and `status_mix` are non-empty.

### storm

This is the ingest regression check, not just data. N identical bodies must
produce **one** `alerts` row with `dedup_count: N` and **one** timeline entry (a
recurrence is not a timeline event). It prints PASS/FAIL and exits non-zero on
failure. If it reports N rows, the unique partial index
`alerts_open_fingerprint_idx` is wrong.

On `module.events` the same run should show exactly one `alert.created` followed
by N-1 `alert.updated` - that distinction is what stops a 10k-alert storm from
firing `on_create` playbooks 10k times.

## scenarios.py

The scenario catalog: 11 scenarios across all five `source_kind` values, each
with an ECS-shaped payload. The ECS field names (`threat.technique.id`,
`host.hostname`, `source.ip`, `file.hash.sha256`) are the point rather than
decoration - they are the exact paths the ATT&CK-tagging and IOC-extraction work
will read, so seeding them now makes that testable against real data later. A
future Elasticsearch branch reuses this file verbatim, pointed at an index
instead of the API.
