// Incident types - snake_case, matching the API wire format exactly.

import type { EventType, Severity, SLAState, SourceKind, LinkSource } from "@/services/alerts/alerts.schema";

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
  playbook: string;
  status: string;
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
  assignee_id?: string;
  team_id?: string;
  q?: string;
  cursor?: string;
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

export interface IncidentsSummary {
  open_total: number;
  status_mix: { status: IncidentStatus; count: number }[];
  severity_mix: { severity: Severity; count: number }[];
  /** Mean resolution time in SECONDS, one point per week, oldest first. */
  mttr_trend: number[];
  sla_at_risk: { id: string; title: string; sla_deadline?: string | null; breached: boolean }[];
}
