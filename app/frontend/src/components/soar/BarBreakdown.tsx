import * as React from "react";
import { cn } from "@/lib/utils";

export type BarTone = "signal" | "moss" | "amber" | "rose" | "slate" | "ink";

const TONE: Record<BarTone, string> = {
  signal: "bg-signal-dot",
  moss: "bg-moss-dot",
  amber: "bg-amber-dot",
  rose: "bg-rose-dot",
  slate: "bg-slate-dot",
  ink: "bg-ink-soft",
};

export interface BarRow {
  label: string;
  /** 0..1 fill fraction */
  value: number;
  display: React.ReactNode;
  tone?: BarTone;
}

export function BarBreakdown({
  rows,
  className,
}: {
  rows: BarRow[];
  className?: string;
}) {
  return (
    <div className={cn("flex flex-col gap-2.5", className)}>
      {rows.map((r, i) => (
        <div
          key={i}
          className="grid grid-cols-[96px_1fr_34px] items-center gap-2.5 text-[12.5px]"
        >
          <span className="truncate font-semibold text-ink-soft">{r.label}</span>
          <div className="h-[7px] overflow-hidden rounded bg-paper-sunken">
            <div
              className={cn("h-full rounded", TONE[r.tone ?? "ink"])}
              style={{ width: `${Math.round(r.value * 100)}%` }}
            />
          </div>
          <span className="text-right text-ink-faint tnum">{r.display}</span>
        </div>
      ))}
    </div>
  );
}
