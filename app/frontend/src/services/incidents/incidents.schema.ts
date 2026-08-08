// Incident types - snake_case, matching the API wire format exactly.

import type { EventType, Severity, SLAState, SourceKind, LinkSource } from "@/services/alerts/alerts.schema";
import type { ResolvedRange, WindowCount } from "@/services/common/range";

export type { Severity } from "@/services/alerts/alerts.schema";
export type IncidentStatus = "open" | "investigating" | "contained" | "resolved" | "closed";

export interface IncidentLinkedAlert {
  id: string;
  title: string;
  severity: Severity;
  source_kind: SourceKind;
  reporter?: string | null;
  link_source: LinkSource;
}

export interface IncidentEvent {
  id: string;
  incident_id: string;
  type: EventType;
  actor_id?: string | null;
  actor_username?: string | null;
  body?: Record<string, unknown> | null;
  created_at: string;
}

export interface IncidentNote {
  id: string;
  incident_id: string;
  author_id?: string | null;
  author_username?: string | null;
  body: string;
  created_at: string;
  updated_at: string;
}

export interface IncidentRun {
  playbook_history_id: string;
  playbook_id: string;
  playbook: string;
  status: string;
  trigger_type?: string | null;
  triggered_by?: string | null;
  created_at: string;
}

export interface Ioc {
  type: string;
  value: string;
}

export interface Incident {
  id: string;
  title: string;
  severity: Severity;
  status: IncidentStatus;
  assignee_id?: string | null;
  assignee?: string | null;
  team_id?: string | null;
  tags?: string[] | null;
  alert_count: number;
  run_count: number;
  resolved_at?: string | null;
  sla_deadline?: string | null;
  sla_state: SLAState;
  created_at: string;
  updated_at: string;
  // detail-only
  linked_alerts?: IncidentLinkedAlert[];
  timeline?: IncidentEvent[];
  notes?: IncidentNote[];
  runs?: IncidentRun[];
  iocs?: Ioc[];
}

export interface IncidentFilter {
  status?: IncidentStatus[];
  severity?: Severity[];
  sla_state?: SLAState[];
  assignee_id?: string[];
  team_id?: string[];
  /** Unions with assignee_id rather than contradicting it. */
  unassigned?: boolean;
  tags?: string[];
  /** RFC3339 instants with an offset - see services/common/range.ts. */
  created_from?: string;
  created_to?: string;
  q?: string;
  /** Paging is cursor XOR offset; sending both is a 400. */
  cursor?: string;
  offset?: number;
  limit?: number;
  open?: boolean;
}

export interface CreateIncidentPayload {
  title: string;
  severity: Severity;
  status?: IncidentStatus;
  assignee_id?: string;
  team_id?: string;
  tags?: string[];
  alert_ids?: string[];
}

export interface UpdateIncidentPayload {
  severity?: Severity;
  assignee_id?: string | null;
  team_id?: string | null;
  tags?: string[];
}

/**
 * One point on the resolution-time series. It carries its own timestamp
 * because the range is caller-picked - see VolumePoint in alerts.schema.ts.
 */
export interface MTTRPoint {
  bucket_start: string;
  avg_seconds: number;
}

/**
 * `open_total`, `status_mix` and `severity_mix` are all-time-open regardless of
 * the range - the queue header and its filter counts read them. The range
 * applies to `mttr_trend` and the window counts only.
 */
export interface IncidentsSummary {
  open_total: number;
  status_mix: { status: IncidentStatus; count: number }[];
  severity_mix: { severity: Severity; count: number }[];
  mttr_trend: MTTRPoint[];
  sla_at_risk: { id: string; title: string; sla_deadline?: string | null; breached: boolean }[];
  range: ResolvedRange;
  created: WindowCount;
  resolved: WindowCount;
  /** Mean time to resolve in seconds across the whole window. */
  mttr_seconds: WindowCount;
}
