import { relativeAge } from "@/lib/utils";
import type { FieldChange } from "@/services/alerts/alerts.schema";
import type { TimelineTone } from "./Timeline";

/**
 * Renders an audit event for the Timeline.
 *
 * The API returns the raw row - a `type` plus an opaque per-type `body` blob -
 * rather than a prose sentence, so the wording lives here. Alert and incident
 * events share the `event_type` enum and this mapper; `body` keys differ per
 * type and are read defensively because several types have no writer yet.
 */

export interface SoarEvent {
  type: string;
  actor_username?: string | null;
  body?: Record<string, unknown> | null;
  created_at: string;
}

const TONE: Record<string, TimelineTone> = {
  created: "signal",
  updated: "signal",
  status_changed: "amber",
  escalated: "rose",
  triage: "signal",
  sla: "rose",
  correlation: "signal",
  attack_tag: "amber",
  linked: "moss",
  unlinked: "amber",
};

const str = (body: Record<string, unknown> | null | undefined, key: string): string | undefined => {
  const v = body?.[key];
  return typeof v === "string" && v.length > 0 ? v : undefined;
};

const FIELD_LABEL: Record<string, string> = {
  severity: "Severity",
  assignee_id: "Assignee",
  team_id: "Team",
  tags: "Tags",
};

/**
 * Renders one side of a field change. The server resolves assignee uuids to
 * usernames, so prefer those; an unresolved id is still better than "[object
 * Object]", and an absent value reads as "unassigned" rather than "null".
 */
function changeSide(value: unknown, username?: string | null): string {
  if (username) return username;
  if (value === null || value === undefined || value === "") return "none";
  if (Array.isArray(value)) return value.length ? value.join(", ") : "none";
  return String(value);
}

function describeChanges(changes: FieldChange[]): { title: string; detail?: string } {
  const parts = changes.map((c) => {
    const label = FIELD_LABEL[c.field] ?? c.field.replace(/_/g, " ");
    return `${label} ${changeSide(c.from, c.from_username)} → ${changeSide(c.to, c.to_username)}`;
  });
  return { title: parts[0], detail: parts.length > 1 ? parts.slice(1).join(" · ") : undefined };
}

function describe(e: SoarEvent): { title: string; detail?: string } {
  const b = e.body;
  switch (e.type) {
    case "created": {
      const reporter = str(b, "reporter");
      if (str(b, "escalated_from_alert_id")) {
        return { title: "Incident opened", detail: "escalated from an alert" };
      }
      return { title: "Created", detail: reporter ? `reported by ${reporter}` : undefined };
    }
    case "updated": {
      const changes = b?.changes;
      if (!Array.isArray(changes) || changes.length === 0) return { title: "Updated" };
      return describeChanges(changes as FieldChange[]);
    }
    case "status_changed": {
      const from = str(b, "from");
      const to = str(b, "to");
      return {
        title: from && to ? `Status ${from} → ${to}` : "Status changed",
        detail: str(b, "closure_note"),
      };
    }
    case "escalated":
      return { title: "Escalated to incident", detail: str(b, "incident_title") };
    case "linked":
      return {
        title: "Linked",
        detail: str(b, "incident_title") ?? str(b, "alert_id"),
      };
    case "unlinked":
      return {
        title: "Unlinked",
        detail: str(b, "incident_title") ?? str(b, "alert_id"),
      };
    case "triage":
      return { title: "Triaged", detail: str(b, "summary") };
    case "sla":
      return { title: "SLA", detail: str(b, "state") };
    case "correlation":
      return { title: "Correlated", detail: str(b, "reason") };
    case "attack_tag":
      return { title: "ATT&CK tagged", detail: str(b, "technique") };
    default:
      return { title: e.type.replace(/_/g, " ") };
  }
}

export function eventEntry(e: SoarEvent): { title: string; detail?: string; tone: TimelineTone } {
  const { title, detail } = describe(e);
  const who = e.actor_username ?? "system";
  return {
    title,
    detail: [detail, `${who} · ${relativeAge(e.created_at)} ago`].filter(Boolean).join(" · "),
    tone: TONE[e.type] ?? "signal",
  };
}
