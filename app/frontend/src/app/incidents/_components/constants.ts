import { AlertTriangle } from "lucide-react";
import type { StatusOption, GlyphTone } from "@/components/soar";
import type { Severity } from "@/services/incidents/incidents.schema";

export const INCIDENT_ICON = AlertTriangle;

export const INCIDENT_STATUS_OPTIONS: StatusOption[] = [
  { value: "open" },
  { value: "investigating" },
  { value: "contained" },
  { value: "resolved" },
  { value: "closed", divider: true },
];

export const SEV_GLYPH_TONE: Record<Severity, GlyphTone> = {
  critical: "rose",
  high: "amber",
  medium: "signal",
  low: "slate",
};

export const SEV_STRIPE: Record<Severity, string> = {
  critical: "bg-rose-dot",
  high: "bg-amber-dot",
  medium: "bg-signal-dot",
  low: "bg-slate-dot",
};
