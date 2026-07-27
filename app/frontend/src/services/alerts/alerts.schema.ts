// Alert types - field names match the API wire format exactly (snake_case, no
// mapper layer), so anything renamed here must be renamed in the Go DTO too.

export type Severity = "critical" | "high" | "medium" | "low";
export type AlertStatus = "new" | "investigating" | "resolved" | "falsepos" | "closed";
export type SourceKind = "edr" | "identity" | "email" | "firewall" | "dlp";
export type SLAState = "ok" | "warning" | "breached" | "met";
export type LinkSource = "manual" | "escalate" | "correlation";

export type EventType =
  | "created"
  | "status_changed"
  | "escalated"
  | "triage"
  | "sla"
  | "correlation"
  | "attack_tag"
  | "linked"
  | "unlinked";

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
  // detail-only
  payload?: Record<string, unknown>;
  closure_note?: string | null;
  fingerprint?: string;
  timeline?: AlertEvent[];
  notes?: AlertNote[];
  linked_incidents?: IncidentRef[];
  related_alerts?: RelatedAlert[];
}

export interface AlertFilter {
  status?: AlertStatus[];
  severity?: Severity[];
  source_kind?: SourceKind[];
  assignee_id?: string;
  team_id?: string;
  q?: string;
  cursor?: string;
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

export interface AlertsSummary {
  total: number;
  by_severity: SeverityBucket[];
  by_source: SourceBucket[];
  top_playbooks: { label: string; success_rate: number }[];
  /** 14 daily counts, oldest first. */
  volume: number[];
}
