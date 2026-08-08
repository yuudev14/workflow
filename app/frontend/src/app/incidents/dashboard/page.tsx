"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { ChevronRight, Clock, X } from "lucide-react";

import IncidentService from "@/services/incidents/incidents";
import type { DateRangeParams } from "@/services/common/range";
import {
  BarBreakdown,
  DateRangePicker,
  Donut,
  KpiCard,
  Panel,
  PanelTitle,
  StatusPill,
  TrendChart,
  defaultRange,
  type BarRow,
  type BarTone,
  type DonutSlice,
} from "@/components/soar";
import { countDelta, percentDelta } from "@/lib/delta";
import { humanDuration, relativeAge } from "@/lib/utils";

const SEV_TONE: Record<string, BarTone> = { critical: "rose", high: "amber", medium: "signal", low: "slate" };
const STATUS_COLOR: Record<string, string> = {
  investigating: "var(--amber-dot)",
  contained: "var(--signal-dot)",
  resolved: "var(--moss-dot)",
  open: "var(--rose-dot)",
  closed: "var(--slate-dot)",
};

export default function Page() {
  const [range, setRange] = React.useState<DateRangeParams>(defaultRange);

  const summaryQuery = useQuery({
    queryKey: ["incidents-summary", range],
    queryFn: () => IncidentService.getIncidentsSummary(range),
  });
  const summary = summaryQuery.data;

  const maxSev = Math.max(...(summary?.severity_mix.map((s) => s.count) ?? [1]));
  const sevRows: BarRow[] =
    summary?.severity_mix.map((s) => ({
      label: s.severity[0].toUpperCase() + s.severity.slice(1),
      value: s.count / maxSev,
      display: s.count,
      tone: SEV_TONE[s.severity],
    })) ?? [];

  const slices: DonutSlice[] =
    summary?.status_mix.map((s) => ({
      label: s.status[0].toUpperCase() + s.status.slice(1),
      value: s.count,
      color: STATUS_COLOR[s.status],
    })) ?? [];

  const mttrSeries =
    summary?.mttr_trend.map((p) => ({ bucket_start: p.bucket_start, value: p.avg_seconds })) ?? [];
  const bucket = summary?.range.bucket ?? "day";
  const created = countDelta(summary?.created);
  const resolved = countDelta(summary?.resolved);
  const mttr = percentDelta(summary?.mttr_seconds);
  const breached = summary?.sla_at_risk.filter((s) => s.breached).length ?? 0;

  return (
    <div className="flex justify-center">
      <div className="flex w-full flex-col gap-5 px-6 py-8">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1>Incidents dashboard</h1>
            <p className="mt-1 text-[15px] text-ink-soft">
              Deltas compare against the preceding window of equal length.
            </p>
          </div>
          <Link
            href="/incidents"
            className="inline-flex items-center gap-2 rounded-sm bg-primary px-3.5 py-2 text-[13.5px] font-semibold text-primary-foreground hover:brightness-110"
          >
            Open cases <ChevronRight className="size-3.5" />
          </Link>
        </div>

        <DateRangePicker value={range} onChange={setRange} />

        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <KpiCard label="Open incidents" value={summary?.open_total ?? "-"} />
          <KpiCard
            label="Opened"
            value={summary?.created.current ?? "-"}
            delta={created?.label}
            deltaDirection={created?.direction}
            deltaNegative
          />
          <KpiCard
            label="Resolved"
            value={summary?.resolved.current ?? "-"}
            delta={resolved?.label}
            deltaDirection={resolved?.direction}
          />
          <KpiCard
            label="Mean time to resolve"
            value={summary ? humanDuration(summary.mttr_seconds.current) : "-"}
            delta={mttr?.label}
            deltaDirection={mttr?.direction}
            deltaNegative
          />
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-[1.3fr_1fr]">
          <Panel>
            <PanelTitle aside={summary ? humanDuration(summary.mttr_seconds.current) : undefined}>
              Mean time to resolve
            </PanelTitle>
            <TrendChart
              data={mttrSeries}
              bucket={bucket}
              tone="moss"
              formatValue={(v) => humanDuration(v)}
            />
          </Panel>
          <Panel>
            <PanelTitle>By status</PanelTitle>
            <Donut slices={slices} centerValue={summary?.open_total ?? 0} centerLabel="open" />
          </Panel>
        </div>

        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <Panel>
            <PanelTitle>By severity</PanelTitle>
            <BarBreakdown rows={sevRows} />
          </Panel>
          <Panel>
            <PanelTitle aside={breached ? `${breached} breached` : undefined}>SLA at risk</PanelTitle>
            <div className="flex flex-col gap-2.5">
              {summary?.sla_at_risk.length === 0 && (
                <p className="text-[12.5px] text-ink-faint">
                  Nothing at risk - no SLA policy is stamping deadlines yet.
                </p>
              )}
              {summary?.sla_at_risk.map((s) => (
                <div key={s.id} className="flex items-center justify-between text-[12.5px]">
                  <span className="flex items-center gap-1.5">
                    {s.breached ? (
                      <X className="size-3.5 text-rose-dot" />
                    ) : (
                      <Clock className="size-3.5 text-amber-dot" />
                    )}
                    {s.title}
                  </span>
                  <StatusPill variant={s.breached ? "critical" : "high"} noDot className="px-2">
                    {relativeAge(s.sla_deadline)}
                  </StatusPill>
                </div>
              ))}
            </div>
          </Panel>
        </div>
      </div>
    </div>
  );
}
