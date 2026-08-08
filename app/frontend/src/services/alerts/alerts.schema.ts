// Alert types - field names match the API wire format exactly (snake_case, no
// mapper layer), so anything renamed here must be renamed in the Go DTO too.

import type { ResolvedRange, WindowCount } from "@/services/common/range";

export type Severity = "critical" | "high" | "medium" | "low";
export type AlertStatus = "new" | "investigating" | "resolved" | "falsepos" | "closed";
export type SourceKind = "edr" | "identity" | "email" | "firewall" | "dlp";
export type SLAState = "ok" | "warning" | "breached" | "met";
export type LinkSource = "manual" | "escalate" | "correlation";

export type EventType =
  | "created"
  | "updated"
  | "status_changed"
  | "escalated"
  | "triage"
  | "sla"
  | "correlation"
  | "attack_tag"
  | "linked"
  | "unlinked";

/**
 * One entry in an `updated` event's `body.changes`. The server fills the
 * username fields for id-valued fields so the timeline reads as a sentence.
 */
export interface FieldChange {
  field: string;
  from: unknown;
  to: unknown;
  from_username?: string | null;
  to_username?: string | null;
}

/** One immutable audit row. `body` is an opaque per-type blob. */
export interface AlertEvent {
  id: string;
  alert_id: string;
  type: EventType;
  actor_id?: string | null;
  actor_username?: string | null;
  body?: Record<string, unknown> | null;
  created_at: string;
}

/** Mutable analyst content. `updated_at != created_at` means it was edited. */
export interface AlertNote {
  id: string;
  alert_id: string;
  author_id?: string | null;
  author_username?: string | null;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface IncidentRef {
  id: string;
  title: string;
  status: string;
  severity: Severity;
  link_source: LinkSource;
}

export interface RelatedAlert {
  id: string;
  title: string;
  score: number;
  shared_iocs: string[];
  created_at: string;
}

/**
 * One playbook run that acted on this record. `playbook_id` is what makes the
 * deep link to the run possible; the name is nullable because the playbook may
 * since have been deleted, and `triggered_by` is null for runs no user started.
 */
export interface RunRef {
  playbook_history_id: string;
  playbook_id: string;
  playbook?: string | null;
  status: string;
  trigger_type?: string | null;
  triggered_by?: string | null;
  created_at: string;
}

export interface Alert {
  id: string;
  title: string;
  severity: Severity;
  status: AlertStatus;
  source_kind: SourceKind;
  reporter?: string | null;
  assignee_id?: string | null;
  assignee?: string | null;
  team_id?: string | null;
  tags?: string[] | null;
  dedup_count: number;
  last_seen: string;
  triaged_at?: string | null;
  sla_deadline?: string | null;
  sla_state: SLAState;
  created_at: string;
  updated_at: string;
  run_count: number;
  // detail-only
  payload?: Record<string, unknown>;
  closure_note?: string | null;
  fingerprint?: string;
  timeline?: AlertEvent[];
  notes?: AlertNote[];
  linked_incidents?: IncidentRef[];
  related_alerts?: RelatedAlert[];
  runs?: RunRef[];
}

export interface AlertFilter {
  status?: AlertStatus[];
  severity?: Severity[];
  source_kind?: SourceKind[];
  sla_state?: SLAState[];
  assignee_id?: string[];
  team_id?: string[];
  /** Unions with assignee_id rather than contradicting it. */
  unassigned?: boolean;
  tags?: string[];
  /** RFC3339 instants with an offset - see services/common/range.ts. */
  created_from?: string;
  created_to?: string;
  /** Only alerts seen at least this many times. */
  dedup_min?: number;
  triaged?: boolean;
  q?: string;
  /** Paging is cursor XOR offset; sending both is a 400. */
  cursor?: string;
  offset?: number;
  limit?: number;
}

export interface CreateAlertPayload {
  title: string;
  severity: Severity;
  source_kind: SourceKind;
  reporter?: string;
  assignee_id?: string;
  team_id?: string;
  payload?: Record<string, unknown>;
  tags?: string[];
  created_at?: string;
}

/**
 * Partial update. The API reads absent-vs-null through a `Nullable[T]` wrapper,
 * so omitting a key leaves the column alone while an explicit null clears it.
 */
export interface UpdateAlertPayload {
  severity?: Severity;
  assignee_id?: string | null;
  team_id?: string | null;
  tags?: string[];
}

export interface UpdateAlertStatusPayload {
  status: AlertStatus;
  /** Disposition text. Cleared server-side when reopening to new/investigating. */
  closure_note?: string;
}

export interface SeverityBucket {
  severity: Severity;
  count: number;
}

export interface SourceBucket {
  source_kind: SourceKind;
  count: number;
}

/**
 * One point on the volume series. It carries its own timestamp because the
 * range is caller-picked: length, start and bucket width all vary per request,
 * so the chart cannot derive its own x-axis labels.
 */
export interface VolumePoint {
  bucket_start: string;
  count: number;
}

/**
 * `total`, `by_severity` and `by_source` are all-time-open regardless of the
 * range - the queue header and its filter counts read them, and range-scoping
 * would silently turn "47 open" into "opened in the last 14 days". The range
 * applies to `volume` and the window counts only.
 */
export interface AlertsSummary {
  total: number;
  by_severity: SeverityBucket[];
  by_source: SourceBucket[];
  /** success_rate is a 0..1 fraction, not a percentage. */
  top_playbooks: { label: string; success_rate: number }[];
  volume: VolumePoint[];
  range: ResolvedRange;
  created: WindowCount;
  resolved: WindowCount;
  /** Mean time to triage in seconds: created_at to triaged_at. */
  mttt_seconds: WindowCount;
}
