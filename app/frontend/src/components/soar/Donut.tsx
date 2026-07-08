import * as React from "react";
import { cn } from "@/lib/utils";

export interface DonutSlice {
  label: string;
  value: number;
  /** css color, e.g. var(--moss-dot) */
  color: string;
}

export function Donut({
  slices,
  centerValue,
  centerLabel,
  className,
}: {
  slices: DonutSlice[];
  centerValue: React.ReactNode;
  centerLabel: string;
  className?: string;
}) {
  const total = slices.reduce((s, x) => s + x.value, 0) || 1;
  let acc = 0;
  const stops = slices
    .map((s) => {
      const from = (acc / total) * 100;
      acc += s.value;
      const to = (acc / total) * 100;
      return `${s.color} ${from}% ${to}%`;
    })
    .join(", ");

  return (
    <div className={cn("flex items-center gap-[18px]", className)}>
      <div
        className="flex size-[108px] shrink-0 items-center justify-center rounded-full"
        style={{ background: `conic-gradient(${stops})` }}
      >
        <div className="flex size-[62px] flex-col items-center justify-center rounded-full bg-card">
          <b className="text-base tnum">{centerValue}</b>
          <span className="text-[11px] text-ink-faint">{centerLabel}</span>
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        {slices.map((s) => (
          <div key={s.label} className="flex items-center gap-2 text-[12.5px] text-ink-soft">
            <span className="size-2 shrink-0 rounded-[2px]" style={{ background: s.color }} />
            {s.label}
            <span className="ml-auto pl-2.5 text-ink-faint tnum">{s.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
