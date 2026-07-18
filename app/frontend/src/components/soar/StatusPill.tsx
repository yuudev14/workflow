import * as React from "react";
import { cn } from "@/lib/utils";

// Every status/severity value the app renders as a pill, mapped to a color
// family. Stable string keys so alerts/incidents/runs can share one component.
export type PillVariant =
  | "success"
  | "failed"
  | "running"
  | "skipped"
  | "new"
  | "investigating"
  | "resolved"
  | "open"
  | "contained"
  | "escalated"
  | "closed"
  | "falsepos"
  | "critical"
  | "high"
  | "medium"
  | "low"
  | "neutral";

type FamilyStyle = { wrap: string; dot: string; pulse?: boolean };

const FAMILY: Record<PillVariant, FamilyStyle> = {
  success: { wrap: "bg-moss-soft text-moss-text", dot: "bg-moss-dot" },
  resolved: { wrap: "bg-moss-soft text-moss-text", dot: "bg-moss-dot" },
  failed: { wrap: "bg-rose-soft text-rose-text", dot: "bg-rose-dot" },
  critical: { wrap: "bg-rose-soft text-rose-text", dot: "bg-rose-dot" },
  open: { wrap: "bg-rose-soft text-rose-text", dot: "bg-rose-dot" },
  running: { wrap: "bg-amber-soft text-amber-text", dot: "bg-amber-dot", pulse: true },
  investigating: { wrap: "bg-amber-soft text-amber-text", dot: "bg-amber-dot", pulse: true },
  high: { wrap: "bg-amber-soft text-amber-text", dot: "bg-amber-dot" },
  medium: { wrap: "bg-signal-soft text-signal-text", dot: "bg-signal-dot" },
  contained: { wrap: "bg-signal-soft text-signal-text", dot: "bg-signal-dot" },
  escalated: { wrap: "bg-signal-soft text-signal-text", dot: "bg-signal-dot" },
  skipped: { wrap: "bg-slate-soft text-slate-text", dot: "bg-slate-dot" },
  low: { wrap: "bg-slate-soft text-slate-text", dot: "bg-slate-dot" },
  new: { wrap: "bg-slate-soft text-slate-text", dot: "bg-slate-dot" },
  closed: { wrap: "bg-slate-soft text-slate-text", dot: "bg-slate-dot" },
  falsepos: { wrap: "bg-slate-soft text-slate-text", dot: "bg-slate-dot" },
  neutral: { wrap: "bg-paper-sunken text-ink-soft border border-line", dot: "bg-ink-faint" },
};

const LABELS: Partial<Record<PillVariant, string>> = {
  falsepos: "False positive",
};

export function pillLabel(variant: PillVariant) {
  return LABELS[variant] ?? variant.charAt(0).toUpperCase() + variant.slice(1);
}

export interface StatusPillProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant: PillVariant;
  /** hide the leading dot */
  noDot?: boolean;
  /** override the auto-generated label */
  children?: React.ReactNode;
}

export function StatusPill({
  variant,
  noDot,
  children,
  className,
  ...props
}: StatusPillProps) {
  const style = FAMILY[variant];
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full py-0.5 pl-2 pr-2.5 text-[12px] font-semibold whitespace-nowrap",
        style.wrap,
        className
      )}
      {...props}
    >
      {!noDot && (
        <span
          className={cn(
            "size-1.5 rounded-full",
            style.dot,
            style.pulse && "motion-safe:animate-pulse-dot"
          )}
        />
      )}
      {children ?? pillLabel(variant)}
    </span>
  );
}
