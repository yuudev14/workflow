"use client";

import * as React from "react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { Bucket } from "@/services/common/range";
import { bucketLabel } from "@/lib/delta";
import { cn } from "@/lib/utils";

export interface TrendPoint {
  bucket_start: string;
  value: number;
}

type Tone = "signal" | "rose" | "moss" | "amber";

// Colours are read as CSS variables rather than hex so the chart follows the
// light/dark theme the same way everything else does.
const FILL: Record<Tone, string> = {
  signal: "var(--signal-dot)",
  rose: "var(--rose-dot)",
  moss: "var(--moss-dot)",
  amber: "var(--amber-dot)",
};

/**
 * Bucketed trend. Every point carries the instant it covers, because the range
 * and the bucket are both caller-picked - the axis cannot be derived from the
 * array's length, which is what the old hardcoded "14 days ago → today"
 * captions assumed.
 */
export function TrendChart({
  data,
  bucket,
  tone = "signal",
  formatValue,
  className,
}: {
  data: TrendPoint[];
  bucket: Bucket;
  tone?: Tone;
  /** e.g. humanDuration for a seconds-valued series */
  formatValue?: (v: number) => string;
  className?: string;
}) {
  if (!data || data.length === 0) {
    return (
      <div
        className={cn(
          "flex h-[168px] items-center justify-center text-[12.5px] text-ink-faint",
          className,
        )}
      >
        No data in this range.
      </div>
    );
  }

  const show = formatValue ?? ((v: number) => String(v));

  return (
    <div className={cn("h-[168px] w-full", className)}>
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data} margin={{ top: 8, right: 4, bottom: 0, left: -8 }}>
          <CartesianGrid vertical={false} stroke="var(--line)" />
          <XAxis
            dataKey="bucket_start"
            tickFormatter={(v: string) => bucketLabel(v, bucket)}
            tick={{ fill: "var(--ink-faint)", fontSize: 11 }}
            tickLine={false}
            axisLine={{ stroke: "var(--line)" }}
            minTickGap={16}
          />
          <YAxis
            tickFormatter={show}
            tick={{ fill: "var(--ink-faint)", fontSize: 11 }}
            tickLine={false}
            axisLine={false}
            width={52}
            allowDecimals={false}
          />
          <Tooltip
            cursor={{ fill: "var(--line)", fillOpacity: 0.4 }}
            content={({ active, payload, label }) => {
              if (!active || !payload?.length) return null;
              return (
                <div className="rounded-sm border border-line bg-card px-2.5 py-1.5 text-[12px] shadow-sm">
                  <div className="font-semibold">{bucketLabel(String(label), bucket)}</div>
                  <div className="text-ink-soft tnum">{show(Number(payload[0].value))}</div>
                </div>
              );
            }}
          />
          <Bar dataKey="value" fill={FILL[tone]} radius={[2, 2, 0, 0]} maxBarSize={38} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}
