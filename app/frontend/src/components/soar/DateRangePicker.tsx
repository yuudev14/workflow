"use client";

import * as React from "react";
import type { Bucket, DateRangeParams } from "@/services/common/range";
import { cn } from "@/lib/utils";

const PRESETS: { label: string; days: number }[] = [
  { label: "7d", days: 7 },
  { label: "14d", days: 14 },
  { label: "30d", days: 30 },
  { label: "90d", days: 90 },
];

const BUCKETS: Bucket[] = ["day", "week", "month", "year"];

/**
 * `datetime-local` renders and parses in the browser's own zone but its value
 * carries no offset, so it has to be round-tripped through Date to become the
 * RFC3339 instant the API binds. Going the other way, toISOString() is UTC and
 * would shift the displayed clock, hence the manual local formatting.
 */
const toLocalInput = (iso?: string): string => {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
};

const fromLocalInput = (v: string): string | undefined =>
  v ? new Date(v).toISOString() : undefined;

const presetRange = (days: number): DateRangeParams => {
  const to = new Date();
  const from = new Date(to.getTime() - days * 24 * 60 * 60 * 1000);
  return { from: from.toISOString(), to: to.toISOString() };
};

/**
 * Range + granularity control for the dashboards.
 *
 * `bucket` is a request parameter rather than something derived from the span,
 * because a weekly view of a 30-day range is a legitimate ask. Leaving it unset
 * lets the server pick a tier from the span; the server rejects combinations
 * that would produce an unreadable series (day buckets over a decade).
 */
export function DateRangePicker({
  value,
  onChange,
  className,
}: {
  value: DateRangeParams;
  onChange: (next: DateRangeParams) => void;
  className?: string;
}) {
  const activePreset = React.useMemo(() => {
    if (!value.from || !value.to) return null;
    const spanDays = (new Date(value.to).getTime() - new Date(value.from).getTime()) / 86_400_000;
    return PRESETS.find((p) => Math.abs(p.days - spanDays) < 0.5)?.label ?? null;
  }, [value.from, value.to]);

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      <div className="flex gap-1 rounded-sm border border-line bg-paper-sunken p-[3px]">
        {PRESETS.map((p) => (
          <button
            key={p.label}
            onClick={() => onChange({ ...value, ...presetRange(p.days) })}
            className={cn(
              "rounded-[6px] px-2.5 py-1 text-[12.5px] font-semibold transition-colors",
              activePreset === p.label
                ? "bg-card text-foreground shadow-sm"
                : "text-ink-soft hover:text-foreground",
            )}
          >
            {p.label}
          </button>
        ))}
      </div>

      <input
        type="datetime-local"
        aria-label="Range start"
        value={toLocalInput(value.from)}
        onChange={(e) => onChange({ ...value, from: fromLocalInput(e.target.value) })}
        className="rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12.5px] outline-none focus:border-signal-dot"
      />
      <span className="text-[12.5px] text-ink-faint">to</span>
      <input
        type="datetime-local"
        aria-label="Range end"
        value={toLocalInput(value.to)}
        onChange={(e) => onChange({ ...value, to: fromLocalInput(e.target.value) })}
        className="rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12.5px] outline-none focus:border-signal-dot"
      />

      <select
        aria-label="Bucket"
        value={value.bucket ?? ""}
        onChange={(e) =>
          onChange({ ...value, bucket: (e.target.value || undefined) as Bucket | undefined })
        }
        className="rounded-sm border border-line-strong bg-background px-2 py-1.5 text-[12.5px] font-semibold text-ink-soft outline-none focus:border-signal-dot"
      >
        <option value="">Auto</option>
        {BUCKETS.map((b) => (
          <option key={b} value={b}>
            {b[0].toUpperCase() + b.slice(1)}
          </option>
        ))}
      </select>
    </div>
  );
}

/** Default view: the last 14 days, matching the server's own fallback span. */
export const defaultRange = (): DateRangeParams => presetRange(14);
