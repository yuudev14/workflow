import * as React from "react";
import {
  Shield,
  Ticket,
  MessageSquare,
  Globe,
  GitBranch,
  Code,
  Zap,
  Layers,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/utils";

export type GlyphTone = "signal" | "moss" | "amber" | "rose" | "slate";

const TONE: Record<GlyphTone, string> = {
  signal: "bg-signal-soft text-signal-text",
  moss: "bg-moss-soft text-moss-text",
  amber: "bg-amber-soft text-amber-text",
  rose: "bg-rose-soft text-rose-text",
  slate: "bg-slate-soft text-slate-text",
};

const SIZE = { sm: "size-[30px] rounded-sm", md: "size-[34px] rounded-lg", lg: "size-9 rounded-[10px]" };

export function Glyph({
  icon: Icon,
  tone = "signal",
  size = "sm",
  className,
}: {
  icon: LucideIcon;
  tone?: GlyphTone;
  size?: keyof typeof SIZE;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "flex shrink-0 items-center justify-center [&_svg]:size-4",
        SIZE[size],
        TONE[tone],
        className
      )}
    >
      <Icon />
    </span>
  );
}

// Best-effort connector → icon + color mapping (visual only; refined once the
// backend exposes a real category field).
export function connectorGlyph(hint?: string | null): { icon: LucideIcon; tone: GlyphTone } {
  const s = (hint ?? "").toLowerCase();
  if (/(firewall|block)/.test(s)) return { icon: Shield, tone: "rose" };
  if (/(virustotal|threat|sandbox|malware|detonat|reputation|shield)/.test(s))
    return { icon: Shield, tone: "moss" };
  if (/(jira|ticket|servicenow)/.test(s)) return { icon: Ticket, tone: "signal" };
  if (/(slack)/.test(s)) return { icon: MessageSquare, tone: "amber" };
  if (/(teams|chat|notify|email|mail|comm)/.test(s)) return { icon: MessageSquare, tone: "slate" };
  if (/(condition|branch|switch)/.test(s)) return { icon: GitBranch, tone: "amber" };
  if (/(code|snippet|python|script)/.test(s)) return { icon: Code, tone: "signal" };
  if (/(webhook|trigger)/.test(s)) return { icon: Zap, tone: "signal" };
  if (/(http|request|rest|globe|url)/.test(s)) return { icon: Globe, tone: "slate" };
  return { icon: Layers, tone: "signal" };
}
