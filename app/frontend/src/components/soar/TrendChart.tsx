import * as React from "react";
import { cn } from "@/lib/utils";

/**
 * Area + line trend. Values are plotted left→right and normalized to the
 * chart height; the final point gets an end dot.
 */
export function TrendChart({
  values,
  startLabel,
  endLabel,
  className,
}: {
  values: number[];
  startLabel?: string;
  endLabel?: string;
  className?: string;
}) {
  const W = 420;
  const H = 108;
  const pad = 14;

  // Nothing to plot yet (e.g. data still loading) — render an empty frame
  // instead of destructuring an out-of-range point.
  if (!values || values.length === 0) {
    return <div className={cn("h-[108px]", className)} />;
  }

  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  const span = max - min || 1;
  const step = values.length > 1 ? W / (values.length - 1) : W;

  const pts = values.map((v, i) => {
    const x = Math.round(i * step);
    const y = Math.round(pad + (1 - (v - min) / span) * (H - pad * 2));
    return [x, y] as const;
  });
  const line = pts.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x},${y}`).join(" ");
  const area = `${line} L${W},${H} L0,${H} Z`;
  const [ex, ey] = pts[pts.length - 1];

  return (
    <div className={className}>
      <svg
        viewBox={`0 0 ${W} ${H}`}
        preserveAspectRatio="none"
        className="block h-[108px] w-full"
      >
        {[18, 54, 90].map((y) => (
          <line key={y} x1={0} y1={y} x2={W} y2={y} className="stroke-line" strokeWidth={1} />
        ))}
        <path d={area} className="fill-signal-dot/15" />
        <path d={line} className="fill-none stroke-signal-dot" strokeWidth={1.8} />
        <circle cx={ex} cy={ey} r={3.5} className="fill-signal-dot" />
      </svg>
      {(startLabel || endLabel) && (
        <div className={cn("mt-1 flex justify-between text-[11.5px] text-ink-faint")}>
          <span>{startLabel}</span>
          <span>{endLabel}</span>
        </div>
      )}
    </div>
  );
}
