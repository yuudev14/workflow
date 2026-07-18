import type { Severity, TimelineTone } from "@/services/alerts/alerts.schema";

export type { Severity } from "@/services/alerts/alerts.schema";
export type IncidentStatus = "open" | "investigating" | "contained" | "resolved" | "closed";

export interface IncidentLinkedAlert {
  id: string;
  title: string;
  source: string;
  severity: Severity;
}

export interface IncidentRun {
  playbook: string;
  detail: string;
  outcome: "success" | "failed";
}

export interface IncidentTimelineEntry {
  title: string;
  detail?: string;
  tone?: TimelineTone;
}

export interface IncidentNote {
  who: string;
  when: string;
  text: string;
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
  owner?: string | null;
  age: string;
  openedAgo?: string;
  slaLeft?: string;
  alertCount: number;
  runCount: number;
  // detail-only
  linkedAlerts?: IncidentLinkedAlert[];
  runs?: IncidentRun[];
  timeline?: IncidentTimelineEntry[];
  notes?: IncidentNote[];
  iocs?: Ioc[];
  tags?: string[];
}

export interface IncidentsSummary {
  openTotal: number;
  statusMix: { status: IncidentStatus; count: number }[];
  severityMix: { severity: Severity; count: number }[];
  mttrTrend: number[];
  slaAtRisk: { id: string; title: string; left: string; breached?: boolean }[];
}
