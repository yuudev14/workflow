import * as React from "react";
import { cn } from "@/lib/utils";
import type { KpiMetric } from "@/services/metrics/metrics.schema";

export function Sparkline({
  points,
  className,
  tone = "signal",
}: {
  /** bar heights 0..1; the last bar is highlighted */
  points: number[];
  className?: string;
  tone?: "signal" | "rose" | "moss" | "amber";
}) {
  const hi =
    tone === "rose"
      ? "bg-rose-dot"
      : tone === "moss"
      ? "bg-moss-dot"
      : tone === "amber"
      ? "bg-amber-dot"
      : "bg-signal-dot";
  return (
    <div className={cn("mt-2 flex h-5 items-end gap-0.5", className)}>
      {points.map((p, i) => (
        <span
          key={i}
          className={cn(
            "w-1 rounded-[1px]",
            i === points.length - 1 ? hi : "bg-line-strong"
          )}
          style={{ height: `${Math.max(6, Math.round(p * 100))}%` }}
        />
      ))}
    </div>
  );
}

export interface KpiCardProps {
  label: string;
  value: React.ReactNode;
  delta?: string;
  deltaDirection?: "up" | "down";
  /** treat an "up" delta as bad (e.g. failures rising) */
  deltaNegative?: boolean;
  spark?: number[];
  sparkTone?: "signal" | "rose" | "moss" | "amber";
  className?: string;
}

export function KpiCard({
  label,
  value,
  delta,
  deltaDirection = "up",
  deltaNegative,
  spark,
  sparkTone,
  className,
}: KpiCardProps) {
  const good = deltaNegative ? deltaDirection === "down" : deltaDirection === "up";
  return (
    <div className={cn("rounded-md border border-line bg-card px-3.5 py-3 shadow-sm", className)}>
      <div className="text-[12px] font-semibold uppercase tracking-wide text-ink-faint">
        {label}
      </div>
      <div className="mt-1.5 flex items-baseline gap-2">
        <span className="text-[26px] font-bold tnum">{value}</span>
        {delta && (
          <span
            className={cn(
              "text-[12.5px] font-semibold",
              good ? "text-moss-text" : "text-rose-text"
            )}
          >
            {delta}
          </span>
        )}
      </div>
      {spark && <Sparkline points={spark} tone={sparkTone} />}
    </div>
  );
}

/** Renders a KpiMetric[] as the 4-up dashboard tile row. */
export function KpiRow({
  metrics,
  loading,
  className,
}: {
  metrics?: KpiMetric[];
  loading?: boolean;
  className?: string;
}) {
  return (
    <div className={cn("grid grid-cols-2 gap-3 md:grid-cols-4", className)}>
      {loading || !metrics
        ? Array.from({ length: 4 }).map((_, i) => (
            <div
              key={i}
              className="h-[92px] animate-pulse rounded-md border border-line bg-paper-sunken"
            />
          ))
        : metrics.map((m) => (
            <KpiCard
              key={m.key}
              label={m.label}
              value={m.value}
              delta={m.delta}
              deltaDirection={m.deltaDirection}
              deltaNegative={m.deltaNegative}
              spark={m.spark}
              sparkTone={m.sparkTone}
            />
          ))}
    </div>
  );
}
