// Alert types — shaped like the eventual API so Phase 3 swaps only the loader.

export type Severity = "critical" | "high" | "medium" | "low";
export type AlertStatus = "new" | "investigating" | "resolved" | "falsepos" | "closed";
export type RunOutcome = "success" | "failed" | "running";
export type SourceKind = "edr" | "identity" | "email" | "firewall" | "dlp";
export type TimelineTone = "signal" | "moss" | "rose" | "amber";

export interface LinkedRun {
  playbook: string;
  outcome: RunOutcome;
}

export interface AlertTimelineEntry {
  title: string;
  detail?: string;
  tone?: TimelineTone;
}

export interface Alert {
  id: string;
  title: string;
  source: string;
  sourceKind: SourceKind;
  severity: Severity;
  status: AlertStatus;
  assignee?: string | null;
  age: string;
  reporter?: string;
  linkedRun?: LinkedRun | null;
  // detail-only
  fields?: { k: string; v: string }[];
  payload?: Record<string, unknown>;
  timeline?: AlertTimelineEntry[];
  relatedAlerts?: { title: string; age: string }[];
  tags?: string[];
}

export interface SeverityBucket {
  severity: Severity;
  count: number;
}

export interface AlertsSummary {
  bySeverity: SeverityBucket[];
  bySource: { label: string; count: number }[];
  topPlaybooks: { label: string; successRate: number }[];
  volume: number[];
  total: number;
}
