import { Globe, MessageSquare, Shield, type LucideIcon } from "lucide-react";
import type { Severity, SourceKind } from "@/services/alerts/alerts.schema";
import type { GlyphTone } from "@/components/soar";

export const SEV_STRIPE: Record<Severity, string> = {
  critical: "bg-rose-dot",
  high: "bg-amber-dot",
  medium: "bg-signal-dot",
  low: "bg-slate-dot",
};

/**
 * Display label for a source kind. The API dropped the free-text `source`
 * column - `source_kind` + `reporter` cover it - so the label lives here.
 */
export const SOURCE_LABEL: Record<SourceKind, string> = {
  edr: "EDR",
  identity: "Identity",
  email: "Email",
  firewall: "Firewall",
  dlp: "DLP",
};

export function sourceGlyph(kind: SourceKind): { icon: LucideIcon; tone: GlyphTone } {
  switch (kind) {
    case "edr":
      return { icon: Shield, tone: "rose" };
    case "identity":
      return { icon: Globe, tone: "signal" };
    case "email":
      return { icon: MessageSquare, tone: "amber" };
    case "firewall":
      return { icon: Globe, tone: "rose" };
    case "dlp":
      return { icon: Globe, tone: "slate" };
    default:
      return { icon: Globe, tone: "slate" };
  }
}
