# Phase 2 — Missing APIs

APIs the frontend already expects but the backend doesn't provide yet. Derived
from the mock services (`Promise.resolve` bodies), the no-op UI actions, and the
Go routes that exist today. Paths are suggestions; response shapes come from the
frontend `*.schema.ts` files.

## 1. Metrics / KPI aggregates — entirely missing
Backs `MetricsService` (all mock). Each returns `KpiMetric[]`
(`{key, label, value, delta, deltaDirection, deltaNegative, spark[], sparkTone}`).

| Method | Path | Feeds |
|---|---|---|
| GET | `/api/metrics/v1/playbooks` | Playbooks dashboard KPI row |
| GET | `/api/metrics/v1/alerts` | Alerts dashboard KPI row |
| GET | `/api/metrics/v1/incidents` | Incidents dashboard KPI row |

## 2. Alerts — entirely missing domain
Backs `AlertService`. No alerts table/module in the backend yet.

| Method | Path | Returns / does |
|---|---|---|
| GET | `/api/alerts/v1` | `Alert[]` (queue list) |
| GET | `/api/alerts/v1/:id` | `Alert` (fields, payload, timeline, relatedAlerts, tags) |
| GET | `/api/alerts/v1/summary` | `AlertsSummary` (bySeverity, bySource, topPlaybooks, volume[], total) |
| PATCH | `/api/alerts/v1/:id/status` | change status (new/investigating/resolved/…) |
| POST | `/api/alerts/v1/:id/escalate` | Escalate button (alert → incident) |

## 3. Incidents — entirely missing domain
Backs `IncidentService`.

| Method | Path | Returns / does |
|---|---|---|
| GET | `/api/incidents/v1` | `Incident[]` (queue) |
| GET | `/api/incidents/v1/:id` | `Incident` (linkedAlerts, runs, timeline, notes, iocs, tags) |
| GET | `/api/incidents/v1/summary` | `IncidentsSummary` (statusMix, severityMix, mttrTrend[], slaAtRisk[]) |
| PATCH | `/api/incidents/v1/:id/status` | status stepper/menu (open→contained→resolved…) |
| POST | `/api/incidents/v1/:id/notes` | "Add note" — notes thread |

## 4. Connectors — partially wired
`GET`, `GET/:id`, `POST`, `DELETE` already exist on the backend.

- **Frontend-wiring only** (no new backend): `ConnectorService` only has
  `getConnectors`. Wire `uploadConnector()` → existing `POST /api/connectors/v1`
  (upload dropzone) and `deleteConnector()` → existing
  `DELETE /api/connectors/v1/:id`.
- **New backend endpoint:** enable/disable toggle. The `connectors` audit row
  has an `enabled` column but no route to flip it. Add
  `PATCH /api/connectors/v1/:id` (or `/:id/enabled`). The `<Toggle>` is
  local-only state today.

## 5. Playbooks — one gap
Full CRUD + history + trigger already exist. Missing:

- **Retry from step** — the "Retry from here" button in the run-detail panel is
  a no-op. Add `POST /api/playbooks/v1/history/:playbook_history_id/retry` taking
  a from-node/task id, re-publishing a `TaskMessage` for the sub-DAG.

## 6. Module-event triggers — cross-cutting, depends on #2/#3
The trigger config stores `trigger_module` (`alert`/`incident`) + `trigger_event`
(`created`/`updated`) on the start node, but nothing subscribes a playbook to
alert/incident lifecycle events. Needs a new trigger type in
`GetPlaybookTriggerTypes` + an event-dispatch path that publishes to queue
`playbook` when an alert/incident is created/updated. Blocked on #2 and #3.

---

**Net-new backend work:** 3 metrics + 5 alerts + 5 incidents + 1 connector-toggle
+ 1 playbook-retry endpoints, plus module-event trigger wiring. Connector
upload/delete are frontend-wiring only (backend already there).
